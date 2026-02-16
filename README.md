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

## Installation

```bash
git clone https://github.com/renniemaharaj/goctx
cd goctx
go build .
sudo mv goctx /bin/goctx
```

## Usage Workflow

1. **Launch**: Run `goctx gui` in any directory to open the dashboard.
2. **Contextualize AI**:
   - Click **Build** to generate the context.
   - Click **Copy** and paste it into an AI model's system prompt (e.g., Google AI Studio) or chat window.
   - *Note: Patch quality depends on the AI model's reasoning capabilities.*
3. **Ingest Patches**: When the AI provides a solution, simply **copy the code/JSON to your clipboard**. GoCtx automatically detects it for review.
4. **Review & Apply**: Validate the changes using the granular diff view, then click **Apply** to integrate (triggers auto-stash for dirty workspaces).

## CLI Reference

- **Stream Context**: Run `goctx` without arguments to output the project state to stdout (useful for piping).
- **Apply Pipe**: Pipe JSON directly into the tool: `cat patch.json | goctx apply`

## Requirements

- Go 1.25+
- GTK3 development headers
