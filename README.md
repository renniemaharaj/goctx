# GoCtx: High-Integrity Context & Patch Orchestrator

GoCtx is a local orchestrator designed to bridge development environments with AI agents. It provides a GTK-based dashboard and a flexible CLI to manage project context and safely apply AI-generated code patches using a strict **Stash-Apply-Commit** workflow.

## Core Features

- **Context Construction**: Automatically gathers project files while respecting `.ctxignore` rules.
- **Surgical Patching**: Supports `SEARCH/REPLACE` blocks for large files, reducing token usage and improving reliability.
- **Hybrid Ingestion**: Monitors the clipboard for two distinct formats:
  1. **Standard JSON**: Full `ProjectOutput` schema with metadata.
  2. **Raw Surgical Fallback**: Detects `SEARCH/REPLACE` blocks directly. Requires a `FILE: path/to/file` header for target identification.
- **Patch Management**: Queues detected patches in the sidebar with a **Surgical Delete (Trash Icon)** to prune the queue without affecting the workspace.
- **Granular Visual Diffs**: Provides a color-coded, semantic diff of changes *within* the surgical block, allowing instant verification of variable renames or logic tweaks.

## CLI Interface

GoCtx supports standard streams for seamless integration between tools:

- **Piping Context**: Send project context directly to other applications (e.g., VS Code):
  ```bash
  go run main.go | code -
  ```
- **Piping for Application**: Pipe a JSON context or patch directly into the apply command:
  ```bash
  cat patch.json | go run main.go apply
  ```

## Usage Protocol

1. **Run Dashboard**: `go run main.go gui`
2. **Build Context**: Generate project state via the UI and copy to the AI agent.
3. **Sanitize Queue**: Review pending patches in the dashboard. Use the delete button to discard non-viable suggestions.
4. **Apply & Finalize**: Apply verified patches (triggering an auto-stash) and use the **Commit** button to clear the dirty state.

Turn your clipboard into data pipelines

## Requirements

- Go 1.25+
- GTK3 development headers
