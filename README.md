````markdown
# GoCtx: High-Integrity Context & Patch Orchestrator

GoCtx is a local orchestrator designed to bridge development environments with AI agents. It provides a GTK-based dashboard and a flexible CLI to manage project context and safely apply AI-generated code patches using a strict **Stash-Apply-Verify-Commit** workflow.

## Core Features

- **Context Construction**: Gathers project state while respecting `.ctxignore`. Supports selective inclusion via the GUI tree-view.
  - **Smart Context**: LSP-aware resolution that automatically includes dependencies of selected files and pulls in files with build errors.
  - **Token Budget**: Adjustable slider to manage context size limits.
- **Surgical Patching**: Uses `SEARCH/REPLACE` blocks to modify specific lines. This preserves file integrity, minimizes token overhead, and avoids the "lazy AI" habit of omitting code.
- **Native Dialect Support**: Accepts raw text patches (non-JSON) directly from the clipboard, reducing JSON escaping errors from AI agents.
- **Verification Engine**: Integrated **Build & Test runners**. Automatically executes your project's validation scripts before finalizing a patch to ensure the AI didn't introduce regressions.
- **High-Integrity Workflow**: Implements a **Stash-Apply-Verify** pattern. Every operation is backed by a native Git stash; if a patch breaks the build or tests, GoCtx automatically stashes the failing changes to keep your workspace stable.
- **Clipboard Monitoring**: Background watcher that instantly detects and ingests AI-generated patches (JSON or Native Dialect) from your clipboard for review.
- **Visual Diffs**: Granular, color-coded diffing that highlights changes _inside_ surgical blocks, allowing for instant human verification of logic tweaks.

## Installation

```bash
git clone https://github.com/renniemaharaj/goctx
cd goctx
go build .
sudo mv goctx /bin/goctx
```
````

## Usage Workflow

1. **Launch**: Run `goctx gui` in your project directory.
2. **Contextualize AI**:
   - Use the file tree to select relevant files.
   - Click **Build** to generate the context.
   - Click **Copy** and paste it into your AI chat (e.g., Google AI Studio, ChatGPT).
3. **Ingest Patches**: When the AI provides a solution, copy the code block (JSON or Native Dialect). GoCtx detects it automatically.
4. **Review & Apply**:
   - Inspect the granular diff in the main panel.
   - Click **Apply**. GoCtx will run your configured Build/Test scripts.
   - If verification fails, you will be prompted to either discard the changes (returning to a clean state) or keep them to fix manually.

## Configuration (`goctx.json` or `ctx.json`)

Define verification scripts in your project root to enable automated safety checks:

```json
{
  "scripts": {
    "build": "go build ./...",
    "test": "go test ./internal/patch/..."
  }
}
```

## CLI Reference

- **Stream Context**: Run `goctx` without arguments to output the project state to stdout (useful for piping).
- **Apply Pipe**: Pipe JSON directly into the tool: `cat patch.json | goctx apply`

## Future Ideas & Roadmap

- **Patch Chaining**: Apply multiple queued patches in a single atomic transaction.
- **Undo/Redo**: Deep integration with Git Reflog for one-click restoration of previous workspace states.
- **Remote Sync**: Optional secure endpoint to receive patches directly from hosted AI agents.
- **Enhanced Diffing**: Move from line-based to AST-aware diffing for Go code.

## Requirements

- Go 1.25+
- GTK3 development headers (e.g., `libgtk-3-dev` on Ubuntu/Debian)

```

```
