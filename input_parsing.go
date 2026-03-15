package main

import (
	"fmt"
	"strconv"
	"strings"
)

func parseFetchPageInput(input map[string]any) fetchPageInput {
	return fetchPageInput{
		Path:      stringValue(input, "path", "url"),
		MaxLinks:  intValue(input, "max_links", "maxLinks"),
		TextLimit: intValue(input, "text_limit", "textLimit"),
	}
}

func parseGetProfileInput(input map[string]any) getProfileInput {
	return getProfileInput{Path: stringValue(input, "path", "profile_path")}
}

func parseListCoursesInput(input map[string]any) listCoursesInput {
	return listCoursesInput{Path: stringValue(input, "path")}
}

func parseGetCourseInput(input map[string]any) getCourseInput {
	return getCourseInput{
		CourseID: intValue(input, "course_id", "id"),
		Path:     stringValue(input, "path", "course_path", "url"),
	}
}

func parseGetChapterInput(input map[string]any) getChapterInput {
	stepID := intValue(input, "step_id", "chapter_id", "id")
	return getChapterInput{
		CourseID:  intValue(input, "course_id"),
		StepID:    stepID,
		ChapterID: stepID,
		Path:      stringValue(input, "path", "chapter_path", "course_path", "url"),
	}
}

func parseGetPartInput(input map[string]any) getPartInput {
	stepID := intValue(input, "step_id", "part_id", "id")
	return getPartInput{
		CourseID: intValue(input, "course_id"),
		StepID:   stepID,
		Path:     stringValue(input, "path", "part_path", "url"),
	}
}

func parseGetTaskInput(input map[string]any) getTaskInput {
	stepID := intValue(input, "step_id", "part_id")
	return getTaskInput{
		CourseID: intValue(input, "course_id"),
		StepID:   stepID,
		TaskID:   intValue(input, "task_id", "id"),
		Path:     stringValue(input, "path", "part_path", "url"),
	}
}

func parseSubmitManualTaskInput(input map[string]any) submitManualTaskInput {
	return submitManualTaskInput{
		CourseID: intValue(input, "course_id"),
		TaskID:   intValue(input, "task_id", "id"),
		Text:     stringValue(input, "text", "answer", "solution"),
	}
}

func parseSubmitAITaskInput(input map[string]any) submitAITaskInput {
	return submitAITaskInput{
		CourseID: intValue(input, "course_id"),
		TaskID:   intValue(input, "task_id", "id"),
		FilePath: stringValue(input, "file_path", "path", "upload_path"),
		PasteURL: stringValue(input, "paste_url", "url"),
		StepID:   intValue(input, "step_id", "part_id"),
		PartPath: stringValue(input, "part_path", "chapter_path"),
	}
}

func parseListProjectsInput(input map[string]any) listProjectsInput {
	return listProjectsInput{Path: stringValue(input, "path")}
}

func parseGetProjectInput(input map[string]any) getProjectInput {
	return getProjectInput{
		ProjectID: intValue(input, "project_id", "id"),
		Path:      stringValue(input, "path", "url"),
	}
}

func stringValue(input map[string]any, keys ...string) string {
	for _, key := range keys {
		if raw, ok := input[key]; ok {
			switch value := raw.(type) {
			case string:
				return strings.TrimSpace(value)
			case fmt.Stringer:
				return strings.TrimSpace(value.String())
			default:
				text := strings.TrimSpace(fmt.Sprint(value))
				if text != "" && text != "<nil>" {
					return text
				}
			}
		}
	}
	return ""
}

func intValue(input map[string]any, keys ...string) int {
	for _, key := range keys {
		if raw, ok := input[key]; ok {
			switch value := raw.(type) {
			case int:
				return value
			case int64:
				return int(value)
			case float64:
				return int(value)
			case string:
				parsed, err := strconv.Atoi(strings.TrimSpace(value))
				if err == nil {
					return parsed
				}
			}
		}
	}
	return 0
}
