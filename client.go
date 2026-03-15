package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var csrfTokenPattern = regexp.MustCompile(`name="_token"\s+value="([^"]+)"`)

type siteClient struct {
	baseURL  *url.URL
	email    string
	password string

	httpClient *http.Client

	mu       sync.Mutex
	loggedIn bool
}

type rawPage struct {
	RequestedPath string
	FinalURL      string
	StatusCode    int
	Body          []byte
	Document      *goquery.Document
}

type getChapterInput struct {
	CourseID  int    `json:"course_id,omitempty" jsonschema:"Course numeric id, for example 157"`
	StepID    int    `json:"step_id,omitempty" jsonschema:"Any step id from the chapter, for example 3760"`
	ChapterID int    `json:"chapter_id,omitempty" jsonschema:"Legacy alias for step_id"`
	Path      string `json:"path,omitempty" jsonschema:"Optional full chapter step path like /insider/courses/157/steps/3760"`
}

type getPartInput struct {
	CourseID int    `json:"course_id,omitempty" jsonschema:"Course numeric id, for example 153"`
	StepID   int    `json:"step_id,omitempty" jsonschema:"Part or step numeric id, for example 2334"`
	Path     string `json:"path,omitempty" jsonschema:"Optional full part path like /insider/courses/153/steps/2334"`
}

type getTaskInput struct {
	CourseID int    `json:"course_id,omitempty" jsonschema:"Course numeric id, for example 153"`
	StepID   int    `json:"step_id,omitempty" jsonschema:"Step numeric id containing the task"`
	TaskID   int    `json:"task_id" jsonschema:"Task numeric id, for example 1356 or 2297"`
	Path     string `json:"path,omitempty" jsonschema:"Optional full part path like /insider/courses/153/steps/2334"`
}

type submitManualTaskInput struct {
	CourseID int    `json:"course_id,omitempty" jsonschema:"Course numeric id, for example 133 or 153"`
	TaskID   int    `json:"task_id" jsonschema:"Task numeric id for manual review submission"`
	Text     string `json:"text" jsonschema:"Submission text or URL to the solution"`
}

type submitAITaskInput struct {
	CourseID int    `json:"course_id,omitempty" jsonschema:"Course numeric id, defaults to 153 when omitted for known AI tasks"`
	TaskID   int    `json:"task_id" jsonschema:"Task numeric id for AI review submission"`
	FilePath string `json:"file_path" jsonschema:"Path to the local file to upload, usually an .ipynb notebook"`
	PasteURL string `json:"paste_url,omitempty" jsonschema:"Optional explicit paste URL when already known"`
	StepID   int    `json:"step_id,omitempty" jsonschema:"Optional step id when you want a direct wrapper page lookup too"`
	PartPath string `json:"part_path,omitempty" jsonschema:"Optional full step path like /insider/courses/153/steps/3626"`
}

type taskSubmissionResult struct {
	Mode             string          `json:"mode"`
	TaskID           int             `json:"task_id"`
	StepID           int             `json:"step_id,omitempty"`
	CourseID         int             `json:"course_id,omitempty"`
	Submitted        bool            `json:"submitted"`
	ResponseStatus   int             `json:"response_status"`
	ResponseText     string          `json:"response_text,omitempty"`
	Mark             string          `json:"mark,omitempty"`
	Comment          string          `json:"comment,omitempty"`
	Task             *courseTask     `json:"task,omitempty"`
	LatestSubmission *taskSubmission `json:"latest_submission,omitempty"`
}

func newSiteClient(cfg serverConfig) (*siteClient, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse CODINGPROJECTS_BASE_URL: %w", err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	return &siteClient{
		baseURL:  baseURL,
		email:    cfg.Email,
		password: cfg.Password,
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 90 * time.Second,
		},
	}, nil
}

func (c *siteClient) FetchPage(ctx context.Context, input fetchPageInput) (pageSnapshot, error) {
	page, err := c.fetchAuthenticatedPage(ctx, input.Path)
	if err != nil {
		return pageSnapshot{}, err
	}

	maxLinks := input.MaxLinks
	if maxLinks <= 0 {
		maxLinks = 50
	}
	textLimit := input.TextLimit
	if textLimit <= 0 {
		textLimit = 2000
	}

	return parsePageSnapshot(page, maxLinks, textLimit), nil
}

func (c *siteClient) GetProfile(ctx context.Context, input getProfileInput) (profilePage, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" {
		path = "/insider/profile"
	}

	page, err := c.fetchAuthenticatedPage(ctx, path)
	if err != nil {
		return profilePage{}, err
	}

	return parseProfilePage(page), nil
}

func (c *siteClient) ListCourses(ctx context.Context, input listCoursesInput) (coursesPage, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" {
		path = "/insider/courses"
	}

	page, err := c.fetchAuthenticatedPage(ctx, path)
	if err != nil {
		return coursesPage{}, err
	}

	return parseCoursesPage(page), nil
}

func (c *siteClient) GetCourse(ctx context.Context, input getCourseInput) (courseDetailPage, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" && input.CourseID > 0 {
		path = fmt.Sprintf("/insider/courses/%d", input.CourseID)
	}
	if path == "" {
		return courseDetailPage{}, fmt.Errorf("provide course_id or path")
	}

	page, err := c.fetchAuthenticatedPage(ctx, path)
	if err != nil {
		return courseDetailPage{}, err
	}

	return parseCourseDetailPage(page), nil
}

func (c *siteClient) GetChapter(ctx context.Context, input getChapterInput) (courseChapterPage, error) {
	path := strings.TrimSpace(input.Path)
	stepID := input.StepID
	if stepID == 0 {
		stepID = input.ChapterID
	}
	if path == "" && input.CourseID > 0 && stepID > 0 {
		path = fmt.Sprintf("/insider/courses/%d/steps/%d", input.CourseID, stepID)
	}
	if path == "" {
		return courseChapterPage{}, fmt.Errorf("provide course_id plus step_id, or path")
	}
	page, err := c.fetchAuthenticatedPage(ctx, path)
	if err != nil {
		return courseChapterPage{}, err
	}
	return parseCourseChapterPage(page), nil
}

func (c *siteClient) GetPart(ctx context.Context, input getPartInput) (coursePartPage, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" && input.CourseID > 0 && input.StepID > 0 {
		path = fmt.Sprintf("/insider/courses/%d/steps/%d", input.CourseID, input.StepID)
	}
	if path == "" {
		return coursePartPage{}, fmt.Errorf("provide course_id plus step_id, or path")
	}
	page, err := c.fetchAuthenticatedPage(ctx, path)
	if err != nil {
		return coursePartPage{}, err
	}
	return parseCoursePartPage(page), nil
}

func (c *siteClient) GetTask(ctx context.Context, input getTaskInput) (courseTask, error) {
	part, err := c.GetPart(ctx, getPartInput{
		CourseID: input.CourseID,
		StepID:   input.StepID,
		Path:     input.Path,
	})
	if err != nil {
		return courseTask{}, err
	}
	for _, task := range part.Tasks {
		if task.ID == input.TaskID {
			return task, nil
		}
	}
	return courseTask{}, fmt.Errorf("task %d not found on part %d", input.TaskID, part.StepID)
}

func (c *siteClient) SubmitManualTask(ctx context.Context, input submitManualTaskInput) (taskSubmissionResult, error) {
	if input.TaskID <= 0 {
		return taskSubmissionResult{}, fmt.Errorf("task_id is required")
	}
	text := strings.TrimSpace(input.Text)
	if text == "" {
		return taskSubmissionResult{}, fmt.Errorf("text is required")
	}

	courseID := input.CourseID
	if courseID == 0 {
		courseID = 153
	}

	values := url.Values{}
	values.Set("text", text)
	requestPath := fmt.Sprintf("/insider/courses/%d/tasks/%d/solution", courseID, input.TaskID)

	page, status, _, err := c.postForm(ctx, requestPath, values, map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded; charset=UTF-8",
		"X-Requested-With": "XMLHttpRequest",
	})
	if err != nil {
		return taskSubmissionResult{}, err
	}

	var payload struct {
		Mark    any `json:"mark"`
		Comment any `json:"comment"`
	}
	if err := json.Unmarshal(page.Body, &payload); err != nil {
		return taskSubmissionResult{}, fmt.Errorf("decode manual submission response: %w", err)
	}

	result := taskSubmissionResult{
		Mode:           submissionModeManual,
		TaskID:         input.TaskID,
		CourseID:       courseID,
		Submitted:      true,
		ResponseStatus: status,
		ResponseText:   excerpt(string(page.Body), 1000),
		Mark:           normalizeAny(payload.Mark),
		Comment:        normalizeAny(payload.Comment),
	}

	if stepID := c.findStepIDForTask(ctx, courseID, input.TaskID); stepID > 0 {
		result.StepID = stepID
		if task, err := c.GetTask(ctx, getTaskInput{CourseID: courseID, StepID: stepID, TaskID: input.TaskID}); err == nil {
			result.Task = &task
			result.LatestSubmission = task.LatestSubmission
		}
	}

	return result, nil
}

func (c *siteClient) SubmitAITask(ctx context.Context, input submitAITaskInput) (taskSubmissionResult, error) {
	if input.TaskID <= 0 {
		return taskSubmissionResult{}, fmt.Errorf("task_id is required")
	}
	if strings.TrimSpace(input.FilePath) == "" {
		return taskSubmissionResult{}, fmt.Errorf("file_path is required")
	}
	if err := validateUploadFile(input.FilePath); err != nil {
		return taskSubmissionResult{}, err
	}

	courseID := input.CourseID
	if courseID == 0 {
		courseID = 153
	}

	pasteURL := strings.TrimSpace(input.PasteURL)
	if pasteURL == "" {
		pasteURL = fmt.Sprintf("https://paste.geekclass.ru/?task_id=%d&course_id=%d", input.TaskID, courseID)
	}

	pastePage, err := c.fetchPasteTaskPage(ctx, pasteURL)
	if err != nil {
		return taskSubmissionResult{}, err
	}
	if pastePage.NotFound {
		return taskSubmissionResult{}, fmt.Errorf("paste task page reported not found for task %d", input.TaskID)
	}
	if !pastePage.HasForm {
		return taskSubmissionResult{}, fmt.Errorf("paste task page does not expose a submission form")
	}

	page, status, _, err := c.postMultipartFile(ctx, pastePage.SubmitURL, input.FilePath)
	if err != nil {
		return taskSubmissionResult{}, err
	}

	result := taskSubmissionResult{
		Mode:           submissionModeAI,
		TaskID:         input.TaskID,
		CourseID:       courseID,
		Submitted:      true,
		ResponseStatus: status,
		ResponseText:   excerpt(normalizeSpace(page.Document.Find("body").Text()), 1200),
	}

	stepID := input.StepID
	if stepID == 0 {
		stepID = c.findStepIDForTask(ctx, courseID, input.TaskID)
	}
	result.StepID = stepID
	if stepID > 0 {
		if task, err := c.GetTask(ctx, getTaskInput{CourseID: courseID, StepID: stepID, TaskID: input.TaskID}); err == nil {
			result.Task = &task
			result.LatestSubmission = task.LatestSubmission
			if task.LatestSubmission != nil {
				result.Mark = task.LatestSubmission.Mark
				result.Comment = task.LatestSubmission.ReviewerComment
			}
		}
	}

	return result, nil
}

func (c *siteClient) ListProjects(ctx context.Context, input listProjectsInput) (projectsPage, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" {
		path = "/insider/projects"
	}

	page, err := c.fetchAuthenticatedPage(ctx, path)
	if err != nil {
		return projectsPage{}, err
	}

	return parseProjectsPage(page), nil
}

func (c *siteClient) GetProject(ctx context.Context, input getProjectInput) (projectDetailPage, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" && input.ProjectID > 0 {
		path = fmt.Sprintf("/insider/projects/%d", input.ProjectID)
	}
	if path == "" {
		return projectDetailPage{}, fmt.Errorf("provide project_id or path")
	}

	page, err := c.fetchAuthenticatedPage(ctx, path)
	if err != nil {
		return projectDetailPage{}, err
	}

	return parseProjectDetailPage(page), nil
}

func (c *siteClient) fetchAuthenticatedPage(ctx context.Context, rawPath string) (rawPage, error) {
	targetURL, err := c.resolveURL(rawPath)
	if err != nil {
		return rawPage{}, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureLoggedInLocked(ctx); err != nil {
		return rawPage{}, err
	}

	page, err := c.getPageLocked(ctx, targetURL, nil)
	if err != nil {
		return rawPage{}, err
	}

	if c.isLoginURL(page.FinalURL) && targetURL.Path != "/login" {
		c.loggedIn = false
		if err := c.ensureLoggedInLocked(ctx); err != nil {
			return rawPage{}, err
		}
		page, err = c.getPageLocked(ctx, targetURL, nil)
		if err != nil {
			return rawPage{}, err
		}
		if c.isLoginURL(page.FinalURL) {
			return rawPage{}, fmt.Errorf("request redirected back to login for %s", targetURL.String())
		}
	}

	return page, nil
}

func (c *siteClient) fetchPasteTaskPage(ctx context.Context, pasteURL string) (pasteTaskPage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureLoggedInLocked(ctx); err != nil {
		return pasteTaskPage{}, err
	}

	jwtURL := c.baseURL.ResolveReference(&url.URL{Path: "/insider/jwt"})
	query := jwtURL.Query()
	query.Set("redirect_url", pasteURL)
	jwtURL.RawQuery = query.Encode()

	page, err := c.getPageLocked(ctx, jwtURL, nil)
	if err != nil {
		return pasteTaskPage{}, err
	}

	return parsePasteTaskPage(page), nil
}

func (c *siteClient) findStepIDForTask(ctx context.Context, courseID, taskID int) int {
	if courseID == 0 || taskID == 0 {
		return 0
	}
	course, err := c.GetCourse(ctx, getCourseInput{CourseID: courseID})
	if err != nil {
		return 0
	}
	for _, grade := range course.Grades {
		if strings.Contains(grade.Value, fmt.Sprintf("#task%d", taskID)) || strings.Contains(grade.Label, fmt.Sprintf("#task%d", taskID)) {
			if stepID := extractNthNumericID(grade.Value, 2); stepID > 0 {
				return stepID
			}
		}
	}
	for _, part := range course.Parts {
		if part.ID == 0 {
			continue
		}
		partPage, err := c.GetPart(ctx, getPartInput{CourseID: courseID, StepID: part.ID})
		if err != nil {
			continue
		}
		for _, task := range partPage.Tasks {
			if task.ID == taskID {
				return partPage.StepID
			}
		}
	}
	return 0
}

func (c *siteClient) ensureLoggedInLocked(ctx context.Context) error {
	if c.loggedIn {
		return nil
	}
	return c.loginLocked(ctx)
}

func (c *siteClient) loginLocked(ctx context.Context) error {
	loginURL := c.baseURL.ResolveReference(&url.URL{Path: "/login"})

	loginPage, err := c.getPageLocked(ctx, loginURL, nil)
	if err != nil {
		return err
	}

	csrfMatch := csrfTokenPattern.FindSubmatch(loginPage.Body)
	if len(csrfMatch) < 2 {
		return fmt.Errorf("login page did not contain csrf token")
	}

	form := url.Values{}
	form.Set("_token", string(csrfMatch[1]))
	form.Set("email", c.email)
	form.Set("password", c.password)
	form.Set("remember", "on")

	_, _, _, err = c.postFormLocked(ctx, loginURL.String(), form, map[string]string{
		"Referer": loginURL.String(),
		"Origin":  c.baseURL.String(),
	})
	if err != nil {
		return err
	}
	c.loggedIn = true
	return nil
}

func (c *siteClient) postForm(ctx context.Context, path string, form url.Values, headers map[string]string) (rawPage, int, http.Header, error) {
	targetURL, err := c.resolveURL(path)
	if err != nil {
		return rawPage{}, 0, nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureLoggedInLocked(ctx); err != nil {
		return rawPage{}, 0, nil, err
	}
	return c.postFormLocked(ctx, targetURL.String(), form, headers)
}

func (c *siteClient) postFormLocked(ctx context.Context, rawURL string, form url.Values, headers map[string]string) (rawPage, int, http.Header, error) {
	page, status, respHeaders, err := c.doRequestLocked(ctx, http.MethodPost, rawURL, strings.NewReader(form.Encode()), mergeHeaders(map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, headers))
	if err != nil {
		return rawPage{}, 0, nil, err
	}
	return page, status, respHeaders, nil
}

func (c *siteClient) postMultipartFile(ctx context.Context, rawURL, filePath string) (rawPage, int, http.Header, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return rawPage{}, 0, nil, fmt.Errorf("read upload file: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return rawPage{}, 0, nil, fmt.Errorf("create multipart part: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return rawPage{}, 0, nil, fmt.Errorf("write multipart file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return rawPage{}, 0, nil, fmt.Errorf("close multipart writer: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureLoggedInLocked(ctx); err != nil {
		return rawPage{}, 0, nil, err
	}
	return c.doRequestLocked(ctx, http.MethodPost, rawURL, bytes.NewReader(body.Bytes()), map[string]string{
		"Content-Type": writer.FormDataContentType(),
	})
}

func (c *siteClient) getPageLocked(ctx context.Context, pageURL *url.URL, headers map[string]string) (rawPage, error) {
	page, _, _, err := c.doRequestLocked(ctx, http.MethodGet, pageURL.String(), nil, headers)
	return page, err
}

func (c *siteClient) doRequestLocked(ctx context.Context, method, rawURL string, body io.Reader, headers map[string]string) (rawPage, int, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return rawPage{}, 0, nil, fmt.Errorf("create %s request for %s: %w", method, rawURL, err)
	}
	for key, value := range headers {
		if strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return rawPage{}, 0, nil, fmt.Errorf("%s %s: %w", method, rawURL, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return rawPage{}, 0, nil, fmt.Errorf("read %s %s: %w", method, rawURL, err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(responseBody))
	if err != nil {
		doc, _ = goquery.NewDocumentFromReader(strings.NewReader("<html></html>"))
	}

	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	return rawPage{
		RequestedPath: req.URL.Path,
		FinalURL:      finalURL,
		StatusCode:    resp.StatusCode,
		Body:          responseBody,
		Document:      doc,
	}, resp.StatusCode, resp.Header.Clone(), nil
}

func (c *siteClient) resolveURL(rawPath string) (*url.URL, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return nil, fmt.Errorf("path is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse path %q: %w", trimmed, err)
	}

	if parsed.IsAbs() {
		if !sameHost(c.baseURL, parsed) {
			return nil, fmt.Errorf("absolute URL must stay on %s", c.baseURL.Host)
		}
		return parsed, nil
	}

	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}

	relative, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse relative path %q: %w", trimmed, err)
	}

	return c.baseURL.ResolveReference(relative), nil
}

func (c *siteClient) isLoginURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return parsed.Path == "/login"
}

func sameHost(left, right *url.URL) bool {
	return strings.EqualFold(left.Hostname(), right.Hostname())
}

func mergeHeaders(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func normalizeAny(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return normalizeSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	default:
		return normalizeSpace(fmt.Sprint(v))
	}
}

func validateUploadFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat upload file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("file_path must be a file, got directory")
	}
	return nil
}
