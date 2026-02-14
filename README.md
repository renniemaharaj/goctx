# GoCtx

GoCtx is a local orchestrator designed to bridge development environments with AI agents. It provides a GTK-based dashboard to manage project context and safely apply AI-generated code patches.

## Core Features

- Context Construction: Automatically gathers project files while respecting .ctxignore rules to build a system prompt.
- Clipboard Monitoring: Detects AI-generated JSON patches from the clipboard for review.
- Safety Stashing: Automatically creates a backup in .stashes before applying any changes.
- Visual Diffs: Provides a color-coded preview of changes and project structure before integration.

## Usage

1. Run the dashboard:
   go run main.go gui

2. Build context and copy to clipboard using the UI.
3. Paste the context into an AI agent.
4. Copy the AI's JSON response.
5. Review the detected patch in the GoCtx dashboard and click apply.

## Requirements

- Go 1.25+
- GTK3 development headers