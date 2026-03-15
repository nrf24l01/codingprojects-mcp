package main

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	submissionModeManual = "manual"
	submissionModeAI     = "ai"
	submissionModeRead   = "read_only"
)

var numericIDPattern = regexp.MustCompile(`/([0-9]+)(?:$|[/?#])`)

type linkRef struct {
	Text string `json:"text,omitempty"`
	URL  string `json:"url"`
}

type namedValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type pageSnapshot struct {
	Path        string    `json:"path"`
	FinalURL    string    `json:"final_url"`
	StatusCode  int       `json:"status_code"`
	Title       string    `json:"title,omitempty"`
	Headings    []string  `json:"headings,omitempty"`
	Links       []linkRef `json:"links,omitempty"`
	TextExcerpt string    `json:"text_excerpt,omitempty"`
}

type profilePage struct {
	Path          string       `json:"path"`
	FinalURL      string       `json:"final_url"`
	Title         string       `json:"title,omitempty"`
	Name          string       `json:"name,omitempty"`
	EditURL       string       `json:"edit_url,omitempty"`
	CoreURL       string       `json:"core_url,omitempty"`
	Fields        []namedValue `json:"fields,omitempty"`
	ExternalLinks []namedValue `json:"external_links,omitempty"`
	CourseLinks   []linkRef    `json:"course_links,omitempty"`
	TextExcerpt   string       `json:"text_excerpt,omitempty"`
}

type courseSummary struct {
	ID          int    `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Progress    string `json:"progress,omitempty"`
	URL         string `json:"url"`
}

type coursesPage struct {
	Path         string          `json:"path"`
	FinalURL     string          `json:"final_url"`
	Title        string          `json:"title,omitempty"`
	InviteAction string          `json:"invite_action,omitempty"`
	Courses      []courseSummary `json:"courses,omitempty"`
}

type courseChapterSummary struct {
	ID       int    `json:"id,omitempty"`
	Title    string `json:"title,omitempty"`
	Progress string `json:"progress,omitempty"`
	URL      string `json:"url"`
	Selected bool   `json:"selected,omitempty"`
}

type coursePartSummary struct {
	ID          int    `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Progress    string `json:"progress,omitempty"`
	IsTask      bool   `json:"is_task,omitempty"`
	URL         string `json:"url"`
}

type participantProgress struct {
	Name       string `json:"name,omitempty"`
	URL        string `json:"url"`
	Progress   string `json:"progress,omitempty"`
	Experience string `json:"experience,omitempty"`
}

type courseDetailPage struct {
	ID             int                    `json:"id,omitempty"`
	CourseID       int                    `json:"course_id,omitempty"`
	ChapterID      int                    `json:"chapter_id,omitempty"`
	Path           string                 `json:"path"`
	FinalURL       string                 `json:"final_url"`
	Title          string                 `json:"title,omitempty"`
	Description    string                 `json:"description,omitempty"`
	Chapters       []courseChapterSummary `json:"chapters,omitempty"`
	CurrentChapter *courseChapterSummary  `json:"current_chapter,omitempty"`
	Parts          []coursePartSummary    `json:"parts,omitempty"`
	Steps          []linkRef              `json:"steps,omitempty"`
	Participants   []participantProgress  `json:"participants,omitempty"`
	Grades         []namedValue           `json:"grades,omitempty"`
	TextExcerpt    string                 `json:"text_excerpt,omitempty"`
}

type courseChapterPage struct {
	ID           int                 `json:"id,omitempty"`
	CourseID     int                 `json:"course_id,omitempty"`
	StepID       int                 `json:"step_id,omitempty"`
	Path         string              `json:"path"`
	FinalURL     string              `json:"final_url"`
	Title        string              `json:"title,omitempty"`
	CourseTitle  string              `json:"course_title,omitempty"`
	CurrentPart  *coursePartSummary  `json:"current_part,omitempty"`
	Parts        []coursePartSummary `json:"parts,omitempty"`
	PreviousPart string              `json:"previous_part,omitempty"`
	NextPart     string              `json:"next_part,omitempty"`
	TextExcerpt  string              `json:"text_excerpt,omitempty"`
}

type taskSubmission struct {
	SubmittedAt     string    `json:"submitted_at,omitempty"`
	Body            string    `json:"body,omitempty"`
	Mark            string    `json:"mark,omitempty"`
	ReviewedAt      string    `json:"reviewed_at,omitempty"`
	Reviewer        string    `json:"reviewer,omitempty"`
	ReviewerComment string    `json:"reviewer_comment,omitempty"`
	Status          string    `json:"status,omitempty"`
	Links           []linkRef `json:"links,omitempty"`
}

type courseTask struct {
	ID                int              `json:"id,omitempty"`
	Title             string           `json:"title,omitempty"`
	Description       string           `json:"description,omitempty"`
	MaxMark           string           `json:"max_mark,omitempty"`
	SubmissionMode    string           `json:"submission_mode,omitempty"`
	SubmitURL         string           `json:"submit_url,omitempty"`
	PasteURL          string           `json:"paste_url,omitempty"`
	AcceptedFileTypes []string         `json:"accepted_file_types,omitempty"`
	LatestSubmission  *taskSubmission  `json:"latest_submission,omitempty"`
	Submissions       []taskSubmission `json:"submissions,omitempty"`
}

type coursePartPage struct {
	ID           int          `json:"id,omitempty"`
	CourseID     int          `json:"course_id,omitempty"`
	StepID       int          `json:"step_id,omitempty"`
	Path         string       `json:"path"`
	FinalURL     string       `json:"final_url"`
	Title        string       `json:"title,omitempty"`
	CourseTitle  string       `json:"course_title,omitempty"`
	TheoryText   string       `json:"theory_text,omitempty"`
	PreviousPart string       `json:"previous_part,omitempty"`
	NextPart     string       `json:"next_part,omitempty"`
	Tasks        []courseTask `json:"tasks,omitempty"`
	TextExcerpt  string       `json:"text_excerpt,omitempty"`
}

type projectSummary struct {
	ID          int    `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Reward      string `json:"reward,omitempty"`
	URL         string `json:"url"`
	RewardURL   string `json:"reward_url,omitempty"`
}

type projectsPage struct {
	Path      string           `json:"path"`
	FinalURL  string           `json:"final_url"`
	Title     string           `json:"title,omitempty"`
	CreateURL string           `json:"create_url,omitempty"`
	Projects  []projectSummary `json:"projects,omitempty"`
}

type projectDetailPage struct {
	Path        string    `json:"path"`
	FinalURL    string    `json:"final_url"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Reward      string    `json:"reward,omitempty"`
	RewardURL   string    `json:"reward_url,omitempty"`
	Links       []linkRef `json:"links,omitempty"`
	TextExcerpt string    `json:"text_excerpt,omitempty"`
}

type pasteTaskPage struct {
	FinalURL          string   `json:"final_url"`
	Title             string   `json:"title,omitempty"`
	Alert             string   `json:"alert,omitempty"`
	SubmitURL         string   `json:"submit_url,omitempty"`
	AcceptedFileTypes []string `json:"accepted_file_types,omitempty"`
	NotFound          bool     `json:"not_found,omitempty"`
	HasForm           bool     `json:"has_form,omitempty"`
}

func parsePageSnapshot(page rawPage, maxLinks, textLimit int) pageSnapshot {
	return pageSnapshot{
		Path:        page.RequestedPath,
		FinalURL:    page.FinalURL,
		StatusCode:  page.StatusCode,
		Title:       pageTitle(page.Document),
		Headings:    collectHeadings(page.Document, 20),
		Links:       collectLinks(page.Document, page.FinalURL, maxLinks, nil),
		TextExcerpt: excerpt(normalizeSpace(page.Document.Find("body").Text()), textLimit),
	}
}

func parseProfilePage(page rawPage) profilePage {
	out := profilePage{
		Path:     page.RequestedPath,
		FinalURL: page.FinalURL,
		Title:    pageTitle(page.Document),
		Name:     normalizeSpace(page.Document.Find("h2").First().Text()),
		EditURL:  firstLinkMatching(page.Document, page.FinalURL, `/insider/profile/[0-9]+/edit$`),
		CoreURL:  firstLinkMatching(page.Document, page.FinalURL, `/insider/core/[0-9]+$`),
		CourseLinks: collectLinks(page.Document, page.FinalURL, 50, func(text, href string) bool {
			return strings.Contains(href, "/insider/courses/") && !strings.Contains(href, "/steps/")
		}),
		TextExcerpt: excerpt(normalizeSpace(page.Document.Find("body").Text()), 2000),
	}

	seenFields := make(map[string]bool)
	page.Document.Find("p, li").Each(func(_ int, sel *goquery.Selection) {
		text := normalizeSpace(sel.Text())
		if text == "" || !strings.Contains(text, ":") {
			return
		}
		label, value, ok := splitLabelValue(text)
		if !ok {
			return
		}
		key := label + "\x00" + value
		if seenFields[key] {
			return
		}
		seenFields[key] = true
		if looksLikeExternalURL(value) {
			out.ExternalLinks = append(out.ExternalLinks, namedValue{Label: label, Value: value})
			return
		}
		out.Fields = append(out.Fields, namedValue{Label: label, Value: value})
	})

	return out
}

func parseCoursesPage(page rawPage) coursesPage {
	out := coursesPage{
		Path:         page.RequestedPath,
		FinalURL:     page.FinalURL,
		Title:        pageTitle(page.Document),
		InviteAction: formActionMatching(page.Document, page.FinalURL, `/insider/courses$`),
	}

	seen := make(map[string]bool)
	page.Document.Find(`.card a[href*="/insider/courses/"]`).Each(func(_ int, link *goquery.Selection) {
		href, ok := link.Attr("href")
		if !ok || strings.Contains(href, "/steps/") {
			return
		}
		absolute := resolveLink(page.FinalURL, href)
		if absolute == "" || seen[absolute] {
			return
		}
		seen[absolute] = true

		card := link.ParentsFiltered(".card").First()
		title := normalizeSpace(link.Text())
		description := normalizeSpace(card.Find(".card-text").First().Text())
		progress := normalizeSpace(card.Find(".badge").First().Text())

		out.Courses = append(out.Courses, courseSummary{
			ID:          extractNumericID(absolute),
			Title:       title,
			Description: description,
			Progress:    progress,
			URL:         absolute,
		})
	})

	sort.SliceStable(out.Courses, func(i, j int) bool {
		return out.Courses[i].ID < out.Courses[j].ID
	})

	return out
}

func parseCourseDetailPage(page rawPage) courseDetailPage {
	parsed, _ := url.Parse(page.FinalURL)
	chapterID, _ := strconv.Atoi(parsed.Query().Get("chapter"))

	out := courseDetailPage{
		ID:          chapterID,
		CourseID:    extractNumericID(page.FinalURL),
		ChapterID:   chapterID,
		Path:        page.RequestedPath,
		FinalURL:    page.FinalURL,
		Title:       normalizeSpace(page.Document.Find("h2").First().Text()),
		Description: normalizeSpace(page.Document.Find("h2").First().Parent().Find("p").First().Text()),
		TextExcerpt: excerpt(normalizeSpace(page.Document.Find("body").Text()), 2500),
	}

	chapterByID := make(map[int]courseChapterSummary)
	page.Document.Find(`li.list-group-item a[href*="?chapter="]`).Each(func(_ int, link *goquery.Selection) {
		href := resolveLink(page.FinalURL, link.AttrOr("href", ""))
		if href == "" {
			return
		}
		id := extractQueryInt(href, "chapter")
		if id == 0 {
			return
		}
		item := link.Closest("li")
		chapter := courseChapterSummary{
			ID:       id,
			Title:    normalizeSpace(link.Text()),
			Progress: normalizeSpace(item.Find(".badge").First().Text()),
			URL:      href,
			Selected: strings.Contains(item.AttrOr("class", ""), "list-group-item-success") || id == chapterID,
		}
		if _, exists := chapterByID[id]; exists {
			return
		}
		chapterByID[id] = chapter
		out.Chapters = append(out.Chapters, chapter)
	})

	for i := range out.Chapters {
		if out.Chapters[i].Selected {
			chapter := out.Chapters[i]
			out.CurrentChapter = &chapter
			if out.ChapterID == 0 {
				out.ChapterID = chapter.ID
				out.ID = chapter.ID
			}
			break
		}
	}
	if out.CurrentChapter == nil && len(out.Chapters) > 0 {
		chapter := out.Chapters[0]
		out.CurrentChapter = &chapter
		if out.ChapterID == 0 {
			out.ChapterID = chapter.ID
			out.ID = chapter.ID
		}
	}

	seenParts := make(map[string]bool)
	page.Document.Find(`.card-group .card`).Each(func(_ int, card *goquery.Selection) {
		link := card.Find(`a[href*="/steps/"]`).First()
		href := resolveLink(page.FinalURL, link.AttrOr("href", ""))
		if href == "" || seenParts[href] {
			return
		}
		seenParts[href] = true

		body := card.Find(".card-body").First().Clone()
		body.Find("h5").Remove()
		part := coursePartSummary{
			ID:          extractNthNumericID(href, 2),
			Title:       normalizeSpace(link.Text()),
			Description: excerpt(normalizeSpace(body.Text()), 400),
			Progress:    cardProgress(card),
			IsTask:      card.Find(`i.ion-trophy`).Length() > 0,
			URL:         href,
		}
		if len(out.Chapters) == 0 || extractQueryInt(out.Chapters[0].URL, "chapter") == 0 {
			out.Chapters = append(out.Chapters, courseChapterSummary{
				ID:       part.ID,
				Title:    part.Title,
				Progress: part.Progress,
				URL:      part.URL,
			})
		}
		out.Parts = append(out.Parts, part)
		out.Steps = append(out.Steps, linkRef{Text: part.Title, URL: part.URL})
	})

	page.Document.Find("ul li").Each(func(_ int, item *goquery.Selection) {
		profileLink := item.Find(`a[href*="/insider/profile/"]`).First()
		href, ok := profileLink.Attr("href")
		if !ok {
			return
		}
		progress := normalizeSpace(item.Find(".badge").First().Text())
		experience := normalizeSpace(item.Find(".badge").First().AttrOr("title", ""))
		if progress == "" && experience == "" {
			return
		}
		out.Participants = append(out.Participants, participantProgress{
			Name:       normalizeSpace(profileLink.Text()),
			URL:        resolveLink(page.FinalURL, href),
			Progress:   progress,
			Experience: experience,
		})
	})

	page.Document.Find("tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}
		label := normalizeSpace(cells.First().Text())
		value := normalizeSpace(cells.Eq(1).Text())
		if label == "" || value == "" {
			return
		}
		out.Grades = append(out.Grades, namedValue{Label: label, Value: value})
	})

	return out
}

func parseCoursePartPage(page rawPage) coursePartPage {
	out := coursePartPage{
		ID:           extractNthNumericID(page.FinalURL, 2),
		CourseID:     extractNthNumericID(page.FinalURL, 1),
		StepID:       extractNthNumericID(page.FinalURL, 2),
		Path:         page.RequestedPath,
		FinalURL:     page.FinalURL,
		Title:        normalizeSpace(page.Document.Find(`small strong`).First().Text()),
		CourseTitle:  normalizeSpace(page.Document.Find(`small a[href*="/insider/courses/"]`).First().Text()),
		TheoryText:   normalizeSpace(page.Document.Find(`#theory`).First().Text()),
		PreviousPart: firstButtonLink(page.Document, page.FinalURL, "Назад"),
		NextPart:     firstButtonLink(page.Document, page.FinalURL, "Вперед"),
		TextExcerpt:  excerpt(normalizeSpace(page.Document.Find("body").Text()), 2500),
	}

	taskTitles := make(map[int]string)
	page.Document.Find(`a.task-pill`).Each(func(_ int, link *goquery.Selection) {
		href := link.AttrOr("href", "")
		if !strings.HasPrefix(href, "#task") {
			return
		}
		id, err := strconv.Atoi(strings.TrimPrefix(href, "#task"))
		if err != nil {
			return
		}
		taskTitles[id] = normalizeSpace(link.Text())
	})

	page.Document.Find(`.tab-pane[id^="task"]`).Each(func(_ int, pane *goquery.Selection) {
		id, err := strconv.Atoi(strings.TrimPrefix(pane.AttrOr("id", ""), "task"))
		if err != nil || id == 0 {
			return
		}
		task := parseCourseTaskPane(page.FinalURL, pane, id, taskTitles[id])
		out.Tasks = append(out.Tasks, task)
	})

	return out
}

func parseCourseChapterPage(page rawPage) courseChapterPage {
	out := courseChapterPage{
		CourseID:     extractNthNumericID(page.FinalURL, 1),
		StepID:       extractNthNumericID(page.FinalURL, 2),
		Path:         page.RequestedPath,
		FinalURL:     page.FinalURL,
		Title:        normalizeSpace(page.Document.Find(`small strong`).First().Text()),
		CourseTitle:  normalizeSpace(page.Document.Find(`small a[href*="/insider/courses/"]`).First().Text()),
		PreviousPart: firstButtonLink(page.Document, page.FinalURL, "Назад"),
		NextPart:     firstButtonLink(page.Document, page.FinalURL, "Вперед"),
		TextExcerpt:  excerpt(normalizeSpace(page.Document.Find("body").Text()), 2500),
	}

	page.Document.Find(`#stepsSidebar a[href*="/steps/"]`).Each(func(_ int, link *goquery.Selection) {
		href := resolveLink(page.FinalURL, link.AttrOr("href", ""))
		if href == "" {
			return
		}
		part := coursePartSummary{
			ID:     extractNthNumericID(href, 2),
			Title:  normalizeSpace(link.Text()),
			IsTask: link.Find(`i.ion-trophy`).Length() > 0 || strings.Contains(link.Find("i").AttrOr("class", ""), "ion-trophy"),
			URL:    href,
		}
		if part.ID == 0 {
			part.ID = extractNumericID(href)
		}
		out.Parts = append(out.Parts, part)
		if strings.Contains(link.AttrOr("class", ""), "active") || part.ID == out.StepID {
			current := part
			out.CurrentPart = &current
		}
	})

	if len(out.Parts) > 0 {
		out.ID = out.Parts[0].ID
	}
	if out.CurrentPart == nil && len(out.Parts) > 0 {
		current := out.Parts[0]
		out.CurrentPart = &current
	}

	return out
}

func parseCourseTaskPane(base string, pane *goquery.Selection, taskID int, title string) courseTask {
	firstCard := pane.Find(".card").First()
	descriptionSel := firstCard.Find(".card-body.markdown").First()
	if descriptionSel.Length() == 0 {
		descriptionSel = firstCard.Find(".card-body").First()
	}
	descriptionClone := descriptionSel.Clone()
	descriptionClone.Find("form, .badge, .small, .btn").Remove()

	task := courseTask{
		ID:          taskID,
		Title:       title,
		Description: excerpt(normalizeSpace(descriptionClone.Text()), 3000),
		MaxMark:     normalizeSpace(pane.Find(".badge.badge-secondary").First().Text()),
	}

	if task.Title == "" {
		task.Title = normalizeSpace(firstCard.Find(".card-header").First().Text())
	}

	if pasteLink := pane.Find(`a[href*="paste.geekclass.ru"]`).First(); pasteLink.Length() > 0 {
		task.SubmissionMode = submissionModeAI
		task.PasteURL = resolveLink(base, pasteLink.AttrOr("href", ""))
		task.AcceptedFileTypes = []string{".ipynb"}
	} else if form := pane.Find(`form[action*="/tasks/"][action$="/solution"]`).First(); form.Length() > 0 {
		task.SubmissionMode = submissionModeManual
		task.SubmitURL = resolveLink(base, form.AttrOr("action", ""))
	} else {
		task.SubmissionMode = submissionModeRead
	}

	pane.Find(".card-header").Each(func(_ int, header *goquery.Selection) {
		if !strings.Contains(normalizeSpace(header.Text()), "Дата сдачи") {
			return
		}
		card := header.Closest(".card")
		submission := parseTaskSubmissionCard(base, card)
		task.Submissions = append(task.Submissions, submission)
	})

	if len(task.Submissions) > 0 {
		latest := task.Submissions[len(task.Submissions)-1]
		task.LatestSubmission = &latest
	}

	return task
}

func parseTaskSubmissionCard(base string, card *goquery.Selection) taskSubmission {
	header := normalizeSpace(card.Find(".card-header").First().Text())
	body := card.Find(".card-body").First()
	submission := taskSubmission{
		SubmittedAt: extractAfterLabel(header, "Дата сдачи:"),
		Body:        normalizeSpace(body.Text()),
		Mark:        normalizeSpace(card.Find(".card-header .badge.badge-primary").First().Text()),
		Links:       collectLinks(body, base, 20, nil),
	}

	if checked := normalizeSpace(body.Find(".badge.badge-light").First().Text()); checked != "" {
		submission.Status = "reviewed"
		checked = extractAfterLabel(checked, "Проверено:")
		parts := strings.SplitN(checked, ",", 2)
		submission.ReviewedAt = normalizeSpace(parts[0])
		if len(parts) > 1 {
			submission.Reviewer = normalizeSpace(parts[1])
		}
	} else {
		submission.Status = "pending"
	}

	comment := normalizeSpace(body.Find("p.small").First().Text())
	if comment != "" {
		submission.ReviewerComment = comment
	}

	return submission
}

func parsePasteTaskPage(page rawPage) pasteTaskPage {
	result := pasteTaskPage{
		FinalURL:  page.FinalURL,
		Title:     pageTitle(page.Document),
		Alert:     normalizeSpace(page.Document.Find(".alert").First().Text()),
		HasForm:   page.Document.Find("form").Length() > 0,
		SubmitURL: resolveLink(page.FinalURL, page.Document.Find("form").First().AttrOr("action", page.FinalURL)),
	}

	if input := page.Document.Find(`input[type="file"]`).First(); input.Length() > 0 {
		accept := strings.TrimSpace(input.AttrOr("accept", ""))
		for _, part := range strings.Split(accept, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				result.AcceptedFileTypes = append(result.AcceptedFileTypes, part)
			}
		}
	}

	if strings.Contains(strings.ToLower(result.Alert), "не найдена") {
		result.NotFound = true
	}

	return result
}

func parseProjectsPage(page rawPage) projectsPage {
	out := projectsPage{
		Path:      page.RequestedPath,
		FinalURL:  page.FinalURL,
		Title:     pageTitle(page.Document),
		CreateURL: firstLinkMatching(page.Document, page.FinalURL, `/insider/projects/create$`),
	}

	seen := make(map[string]bool)
	page.Document.Find(`.media a[href*="/insider/projects/"]`).Each(func(_ int, link *goquery.Selection) {
		href, ok := link.Attr("href")
		if !ok || strings.Contains(href, "/reward") || strings.HasSuffix(href, "/create") {
			return
		}
		absolute := resolveLink(page.FinalURL, href)
		if absolute == "" || seen[absolute] {
			return
		}
		id := extractNumericID(absolute)
		if id == 0 {
			return
		}
		seen[absolute] = true

		card := link.ParentsFiltered(".media").First()

		title := normalizeSpace(link.Text())
		description := normalizeSpace(card.Find("p").First().Text())
		reward := normalizeSpace(card.Find(".d-inline-flex").First().Text())
		rewardURL := firstLinkFromSelection(card, page.FinalURL, `/insider/projects/[0-9]+/reward$`)

		out.Projects = append(out.Projects, projectSummary{
			ID:          id,
			Title:       title,
			Description: description,
			Reward:      reward,
			URL:         absolute,
			RewardURL:   rewardURL,
		})
	})

	sort.SliceStable(out.Projects, func(i, j int) bool {
		return out.Projects[i].ID < out.Projects[j].ID
	})

	return out
}

func parseProjectDetailPage(page rawPage) projectDetailPage {
	out := projectDetailPage{
		Path:        page.RequestedPath,
		FinalURL:    page.FinalURL,
		Title:       normalizeSpace(page.Document.Find("h3.card-title").First().Text()),
		Description: firstNonEmptyParagraph(page.Document.Find(".card-body").First()),
		Reward:      normalizeSpace(page.Document.Find(".d-flex").First().Text()),
		RewardURL:   firstLinkMatching(page.Document, page.FinalURL, `/insider/projects/[0-9]+/reward$`),
		Links:       collectLinks(page.Document, page.FinalURL, 20, func(_ string, href string) bool { return strings.Contains(href, "/insider/projects/") }),
		TextExcerpt: excerpt(normalizeSpace(page.Document.Find("body").Text()), 2000),
	}

	if out.Description == "" {
		out.Description = normalizeSpace(page.Document.Find("p").First().Text())
	}

	return out
}

func pageTitle(doc *goquery.Document) string {
	return normalizeSpace(doc.Find("title").First().Text())
}

func collectHeadings(doc *goquery.Document, limit int) []string {
	var headings []string
	seen := make(map[string]bool)
	doc.Find("h1, h2, h3, h4").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		text := normalizeSpace(sel.Text())
		if text == "" || seen[text] {
			return true
		}
		seen[text] = true
		headings = append(headings, text)
		return len(headings) < limit
	})
	return headings
}

func collectLinks(node interface {
	Find(string) *goquery.Selection
}, base string, limit int, keep func(text, href string) bool) []linkRef {
	var links []linkRef
	seen := make(map[string]bool)
	node.Find("a[href]").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		href := strings.TrimSpace(sel.AttrOr("href", ""))
		if href == "" {
			return true
		}
		text := normalizeSpace(sel.Text())
		if keep != nil && !keep(text, href) {
			return true
		}
		resolved := resolveLink(base, href)
		if resolved == "" || seen[resolved] {
			return true
		}
		seen[resolved] = true
		links = append(links, linkRef{Text: text, URL: resolved})
		return len(links) < limit
	})
	return links
}

func resolveLink(base, href string) string {
	href = html.UnescapeString(strings.TrimSpace(href))
	if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(strings.ToLower(href), "javascript:") {
		return ""
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}
	relative, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return baseURL.ResolveReference(relative).String()
}

func firstLinkMatching(doc *goquery.Document, base, pattern string) string {
	re := regexp.MustCompile(pattern)
	return firstLinkFromSelection(doc.Selection, base, pattern, re)
}

func firstLinkFromSelection(sel *goquery.Selection, base, pattern string, compiled ...*regexp.Regexp) string {
	var re *regexp.Regexp
	if len(compiled) > 0 {
		re = compiled[0]
	} else {
		re = regexp.MustCompile(pattern)
	}
	result := ""
	sel.Find("a[href]").EachWithBreak(func(_ int, link *goquery.Selection) bool {
		href := resolveLink(base, link.AttrOr("href", ""))
		if href == "" || !re.MatchString(href) {
			return true
		}
		result = href
		return false
	})
	return result
}

func formActionMatching(doc *goquery.Document, base, pattern string) string {
	re := regexp.MustCompile(pattern)
	result := ""
	doc.Find("form[action]").EachWithBreak(func(_ int, form *goquery.Selection) bool {
		action := resolveLink(base, form.AttrOr("action", ""))
		if action == "" || !re.MatchString(action) {
			return true
		}
		result = action
		return false
	})
	return result
}

func firstNonEmptyParagraph(sel *goquery.Selection) string {
	result := ""
	sel.Find("p").EachWithBreak(func(_ int, p *goquery.Selection) bool {
		text := normalizeSpace(p.Text())
		if text == "" {
			return true
		}
		result = text
		return false
	})
	return result
}

func firstButtonLink(doc *goquery.Document, base, text string) string {
	result := ""
	doc.Find("a[href]").EachWithBreak(func(_ int, link *goquery.Selection) bool {
		if normalizeSpace(link.Text()) != text {
			return true
		}
		result = resolveLink(base, link.AttrOr("href", ""))
		return result == ""
	})
	return result
}

func cardProgress(card *goquery.Selection) string {
	if badge := normalizeSpace(card.Find(".card-footer .badge").First().Text()); badge != "" {
		return badge
	}
	if value := strings.TrimSpace(card.Find(".progress-bar").First().AttrOr("aria-valuenow", "")); value != "" {
		return value + "%"
	}
	return ""
}

func extractNumericID(raw string) int {
	match := numericIDPattern.FindStringSubmatch(raw)
	if len(match) < 2 {
		return 0
	}
	id, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return id
}

func extractNthNumericID(raw string, index int) int {
	matches := numericIDPattern.FindAllStringSubmatch(raw, -1)
	if index <= 0 || len(matches) < index {
		return 0
	}
	id, err := strconv.Atoi(matches[index-1][1])
	if err != nil {
		return 0
	}
	return id
}

func extractQueryInt(rawURL, key string) int {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}
	value := parsed.Query().Get(key)
	result, _ := strconv.Atoi(value)
	return result
}

func extractAfterLabel(text, label string) string {
	text = normalizeSpace(text)
	if label == "" {
		return text
	}
	idx := strings.Index(text, label)
	if idx == -1 {
		return text
	}
	return normalizeSpace(text[idx+len(label):])
}

func splitLabelValue(text string) (string, string, bool) {
	left, right, ok := strings.Cut(text, ":")
	if !ok {
		return "", "", false
	}
	label := normalizeSpace(left)
	value := normalizeSpace(right)
	if label == "" || value == "" {
		return "", "", false
	}
	return label, value, true
}

func looksLikeExternalURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func normalizeSpace(value string) string {
	value = html.UnescapeString(value)
	return strings.Join(strings.Fields(strings.ReplaceAll(value, "\u00a0", " ")), " ")
}

func excerpt(value string, limit int) string {
	value = normalizeSpace(value)
	if len(value) <= limit || limit <= 0 {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return fmt.Sprintf("%s...", value[:limit-3])
}
