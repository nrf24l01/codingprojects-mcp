package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "0.2.0"

type serverConfig struct {
	BaseURL  string
	Email    string
	Password string
}

type fetchPageInput struct {
	Path      string `json:"path" jsonschema:"Relative path like /insider/projects or an absolute codingprojects.ru URL"`
	MaxLinks  int    `json:"max_links,omitempty" jsonschema:"Maximum number of links to include in the response"`
	TextLimit int    `json:"text_limit,omitempty" jsonschema:"Maximum number of characters in the text excerpt"`
}

type getProfileInput struct {
	Path string `json:"path,omitempty" jsonschema:"Optional profile path, defaults to /insider/profile"`
}

type listCoursesInput struct {
	Path string `json:"path,omitempty" jsonschema:"Optional courses path, defaults to /insider/courses"`
}

type getCourseInput struct {
	CourseID int    `json:"course_id,omitempty" jsonschema:"Course numeric id, for example 158"`
	Path     string `json:"path,omitempty" jsonschema:"Optional full course path like /insider/courses/158"`
}

type listProjectsInput struct {
	Path string `json:"path,omitempty" jsonschema:"Optional projects path, defaults to /insider/projects"`
}

type getProjectInput struct {
	ProjectID int    `json:"project_id,omitempty" jsonschema:"Project numeric id, for example 316"`
	Path      string `json:"path,omitempty" jsonschema:"Optional full project path like /insider/projects/316"`
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	client, err := newSiteClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "codingprojects",
		Version: version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fetch_page",
		Description: "Fetch an authenticated codingprojects.ru page and return headings, links, and a text excerpt.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, pageSnapshot, error) {
		parsed := parseFetchPageInput(input)
		out, err := client.FetchPage(ctx, parsed)
		return nil, out, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_profile",
		Description: "Fetch the current user's profile page and extract key fields.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, profilePage, error) {
		out, err := client.GetProfile(ctx, parseGetProfileInput(input))
		return nil, out, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_courses",
		Description: "Fetch the authenticated courses page and extract enrolled courses.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, coursesPage, error) {
		out, err := client.ListCourses(ctx, parseListCoursesInput(input))
		return nil, out, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_course",
		Description: "Fetch a course page with chapters, parts, and grade links.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, courseDetailPage, error) {
		out, err := client.GetCourse(ctx, parseGetCourseInput(input))
		return nil, out, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_course_chapter",
		Description: "Fetch one chapter as a grouped step range from a step page sidebar.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, map[string]any, error) {
		out, err := client.GetChapter(ctx, parseGetChapterInput(input))
		if err != nil {
			return nil, nil, err
		}
		return nil, toMap(out), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_course_part",
		Description: "Fetch one course part or step with theory and all tasks on the page.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, coursePartPage, error) {
		out, err := client.GetPart(ctx, parseGetPartInput(input))
		return nil, out, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_course_task",
		Description: "Fetch one specific course task, including its latest submission result when present.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, courseTask, error) {
		out, err := client.GetTask(ctx, parseGetTaskInput(input))
		return nil, out, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "submit_manual_task",
		Description: "Submit a text or link answer for a manually checked task and return the immediate response plus latest result when available.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, taskSubmissionResult, error) {
		out, err := client.SubmitManualTask(ctx, parseSubmitManualTaskInput(input))
		return nil, out, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "submit_ai_task",
		Description: "Submit a notebook file for an AI-checked task through paste.geekclass.ru and return the observed result.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, taskSubmissionResult, error) {
		out, err := client.SubmitAITask(ctx, parseSubmitAITaskInput(input))
		return nil, out, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_projects",
		Description: "Fetch the authenticated projects page and extract available projects.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, projectsPage, error) {
		out, err := client.ListProjects(ctx, parseListProjectsInput(input))
		return nil, out, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_project",
		Description: "Fetch a project page and extract its main details.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, projectDetailPage, error) {
		out, err := client.GetProject(ctx, parseGetProjectInput(input))
		return nil, out, err
	})

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func loadConfig() (serverConfig, error) {
	baseURL := strings.TrimSpace(os.Getenv("CODINGPROJECTS_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://codingprojects.ru"
	}

	email := strings.TrimSpace(os.Getenv("CODINGPROJECTS_EMAIL"))
	password := strings.TrimSpace(os.Getenv("CODINGPROJECTS_PASSWORD"))

	if email == "" || password == "" {
		return serverConfig{}, fmt.Errorf("missing credentials: set CODINGPROJECTS_EMAIL and CODINGPROJECTS_PASSWORD")
	}

	return serverConfig{
		BaseURL:  baseURL,
		Email:    email,
		Password: password,
	}, nil
}

func toMap(value any) map[string]any {
	encoded, err := json.Marshal(value)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(encoded, &out); err != nil {
		return map[string]any{}
	}
	return out
}
