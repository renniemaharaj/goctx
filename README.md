# GoCtx

GoCtx is a local orchestrator designed to bridge development environments with AI agents. It provides a GTK-based dashboard and a flexible CLI to manage project context and safely apply AI-generated code patches.

## Core Features

- **Context Construction**: Automatically gathers project files while respecting `.ctxignore` rules.
- **Surgical Patching**: Supports `SEARCH/REPLACE` blocks for large files, reducing token usage and improving reliability.
- **Clipboard Monitoring**: Detects AI-generated JSON patches from the clipboard for instant review.
- **Safety Stashing**: Automatically creates a backup in `.stashes` before applying any changes.
- **Visual Diffs**: Provides a color-coded preview of changes and validates surgical matches before integration.

## CLI Interface

GoCtx supports standard streams for seamless integration between tools:

- **Piping Context**: Send project context directly to other applications (e.g., VS Code):
  ```bash
  go run main.go | code -
  ```
- **Piping for Application**: Pipe a JSON context or patch directly into the apply command:
  ```bash
  cat patch.json | go run main.go apply
  # Or piping from another tool's output
  custom-generator-tool | go run main.go apply
  ```

## Stash Management

- **Navigation**: Move through your history of changes using the command line:
  ```bash
  go run main.go back
  go run main.go forward
  ```
- **Tidy**: Clean up the stash history and reset the `.stashes` directory:
  ```bash
  go run main.go tidy
  ```

## Usage

1. Run the dashboard:

   ```bash
   go run main.go gui
   ```

2. Build context and copy to clipboard using the UI.
3. Paste the context into an AI agent.
4. Copy the AI's JSON response.
5. Review the detected patch in the GoCtx dashboard and click apply.

## Requirements

- Go 1.25+
- GTK3 development headers
