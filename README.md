# codingprojects-mcp

Go MCP server for `codingprojects.ru`.

It logs in with credentials from environment variables, keeps an authenticated session, and exposes tools for:

- listing courses
- reading course chapters
- reading course parts or steps
- reading individual tasks
- submitting manual-review tasks
- submitting AI-review tasks through `paste.geekclass.ru`
- reading the latest submission result shown on the course page

## Environment

Required:

- `CODINGPROJECTS_EMAIL`
- `CODINGPROJECTS_PASSWORD`

Optional:

- `CODINGPROJECTS_BASE_URL` default: `https://codingprojects.ru`

Example:

```bash
export CODINGPROJECTS_EMAIL="user@example.com"
export CODINGPROJECTS_PASSWORD="secret"
export CODINGPROJECTS_BASE_URL="https://codingprojects.ru"
```

## Tools

- `fetch_page`
- `get_profile`
- `list_courses`
- `get_course`
- `get_course_chapter`
- `get_course_part`
- `get_course_task`
- `submit_manual_task`
- `submit_ai_task`
- `list_projects`
- `get_project`

## Submission notes

- Manual-review tasks submit text to `/insider/courses/{course_id}/tasks/{task_id}/solution`
- AI-review tasks are submitted through `https://codingprojects.ru/insider/jwt?redirect_url=...` and then uploaded to `paste.geekclass.ru`
- For the tested AI task `2297` at `https://codingprojects.ru/insider/courses/153/steps/3626`, the paste service currently returns `"Задача не найдена в базе."`, so the server reports that as an error instead of pretending submission succeeded
- Latest task results are parsed from the course step page when they are visible there

## Installation

Download a binary from the GitHub `Latest` release that matches your OS and CPU.

Examples:

- Linux x86_64: `codingprojects-mcp-linux-amd64.tar.gz`
- Linux ARM64: `codingprojects-mcp-linux-arm64.tar.gz`
- macOS Apple Silicon: `codingprojects-mcp-darwin-arm64.tar.gz`
- macOS Intel: `codingprojects-mcp-darwin-amd64.tar.gz`
- Windows x86_64: `codingprojects-mcp-windows-amd64.zip`

Extract it and move the binary somewhere in your `PATH`, for example:

```bash
tar -xzf codingprojects-mcp-linux-amd64.tar.gz
chmod +x codingprojects-mcp
mv codingprojects-mcp ~/.local/bin/
```

You can also verify the download with the matching `.sha256` file from the release:

```bash
sha256sum -c codingprojects-mcp-linux-amd64.tar.gz.sha256
```

## Run

```bash
go run .
```

## Build

```bash
go build -o codingprojects-mcp .
```

## Test

```bash
go test ./...
```

## Opencode setup

Create or edit `~/.config/opencode/opencode.jsonc`:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "codingprojects-mcp": {
      "enabled": true,
      "type": "local",
      "command": ["/home/your-user/.local/bin/codingprojects-mcp"],
      "environment": {
        "CODINGPROJECTS_EMAIL": "user@example.com",
        "CODINGPROJECTS_PASSWORD": "secret",
        "CODINGPROJECTS_BASE_URL": "https://codingprojects.ru"
      }
    }
  }
}
```

If the binary is already in your `PATH`, you can use just:

```jsonc
"command": ["codingprojects-mcp"]
```

Example opencode config:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "codingprojects-mcp": {
      "enabled": true,
      "type": "local",
      "command": ["codingprojects-mcp"],
      "environment": {
        "CODINGPROJECTS_EMAIL": "user@example.com",
        "CODINGPROJECTS_PASSWORD": "secret"
      }
    }
  }
}
```
