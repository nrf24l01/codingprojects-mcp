package main

import (
	"context"
	"os"
	"testing"
)

func TestAuthenticatedFetches(t *testing.T) {
	email := os.Getenv("CODINGPROJECTS_EMAIL")
	password := os.Getenv("CODINGPROJECTS_PASSWORD")
	if email == "" || password == "" {
		t.Skip("set CODINGPROJECTS_EMAIL and CODINGPROJECTS_PASSWORD for smoke test")
	}

	client, err := newSiteClient(serverConfig{
		BaseURL:  getenvDefault("CODINGPROJECTS_BASE_URL", "https://codingprojects.ru"),
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("newSiteClient: %v", err)
	}

	ctx := context.Background()

	profile, err := client.GetProfile(ctx, getProfileInput{})
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if profile.Name == "" {
		t.Fatal("GetProfile returned empty name")
	}

	courses, err := client.ListCourses(ctx, listCoursesInput{})
	if err != nil {
		t.Fatalf("ListCourses: %v", err)
	}
	if len(courses.Courses) == 0 {
		t.Fatal("ListCourses returned no courses")
	}

	course, err := client.GetCourse(ctx, getCourseInput{CourseID: 153})
	if err != nil {
		t.Fatalf("GetCourse: %v", err)
	}
	if len(course.Parts) == 0 {
		t.Fatal("GetCourse returned no parts")
	}

	chapterCourse, err := client.GetCourse(ctx, getCourseInput{CourseID: 157})
	if err != nil {
		t.Fatalf("GetCourse for chapters: %v", err)
	}
	if len(chapterCourse.Chapters) == 0 {
		t.Fatal("GetCourse returned no chapters for chapter test")
	}

	chapter, err := client.GetChapter(ctx, getChapterInput{CourseID: 157, StepID: chapterCourse.Chapters[0].ID})
	if err != nil {
		t.Fatalf("GetChapter: %v", err)
	}
	if chapter.ID == 0 {
		t.Fatal("GetChapter returned no chapter id")
	}
	if len(chapter.Parts) == 0 {
		t.Fatal("GetChapter returned no chapter parts")
	}

	part, err := client.GetPart(ctx, getPartInput{CourseID: 153, StepID: 2334})
	if err != nil {
		t.Fatalf("GetPart: %v", err)
	}
	if len(part.Tasks) == 0 {
		t.Fatal("GetPart returned no tasks")
	}

	task, err := client.GetTask(ctx, getTaskInput{CourseID: 153, StepID: 2334, TaskID: 1356})
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.ID != 1356 {
		t.Fatalf("unexpected task id %d", task.ID)
	}
	if task.SubmissionMode == "" {
		t.Fatal("GetTask returned empty submission mode")
	}

	projects, err := client.ListProjects(ctx, listProjectsInput{})
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects.Projects) == 0 {
		t.Fatal("ListProjects returned no projects")
	}

	page, err := client.FetchPage(ctx, fetchPageInput{Path: "/insider/projects", MaxLinks: 10, TextLimit: 500})
	if err != nil {
		t.Fatalf("FetchPage: %v", err)
	}
	if page.Title == "" {
		t.Fatal("FetchPage returned empty title")
	}
}

func TestPasteTaskPageFetch(t *testing.T) {
	email := os.Getenv("CODINGPROJECTS_EMAIL")
	password := os.Getenv("CODINGPROJECTS_PASSWORD")
	if email == "" || password == "" {
		t.Skip("set CODINGPROJECTS_EMAIL and CODINGPROJECTS_PASSWORD for smoke test")
	}

	client, err := newSiteClient(serverConfig{
		BaseURL:  getenvDefault("CODINGPROJECTS_BASE_URL", "https://codingprojects.ru"),
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("newSiteClient: %v", err)
	}

	pastePage, err := client.fetchPasteTaskPage(context.Background(), "https://paste.geekclass.ru/?task_id=2297&course_id=153")
	if err != nil {
		t.Fatalf("fetchPasteTaskPage: %v", err)
	}
	if pastePage.NotFound {
		t.Fatal("fetchPasteTaskPage unexpectedly returned not found")
	}
	if !pastePage.HasForm {
		t.Fatal("fetchPasteTaskPage returned no form")
	}
	if len(pastePage.AcceptedFileTypes) == 0 {
		t.Fatal("fetchPasteTaskPage returned no accepted file types")
	}
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
