# GoCtx Project Instructions

## Project Overview
Hybrid Orchestrator bridging local dev and AI. GTK Dashboard + Local API Bridge. The tool manages project context, detects AI-generated patches via clipboard, and manages stashes for version safety.

## UI Structure (LOCKED)
- **Left Sidebar**: 
    - Actions: [CURRENT CONTEXT], [COPY CONTEXT], [APPLY SELECTED PATCH], [APPLY SELECTED STASH].
    - Lists: PENDING PATCHES (from clipboard), STASHES (local history).
- **Right Dashboard**: Monospace display for Project Trees, Diff Previews (limited to first 10 files for performance), and status logs.

## Rules & Constraints
- **Core Stability**: Do NOT break the file/folder tree rendering or the diffing logic. These are critical for visualizing changes before application.
- **Context Building**: Use `.ctxignore` to exclude binary files, `.git`, `.stashes`, and large dependency folders (e.g., `node_modules`).
- **Patching & Stashes**: 
    - Patches detected from clipboard must be applied accurately to the local filesystem.
    - Every application must create a Stash in `.stashes` for rollback capability.
- **Diffing**: Use `diffmatchpatch` for accurate, color-coded visual feedback in the GTK `TextView`.
- **No Browser**: All AI interactions are manual (copy context -> paste to AI -> copy AI response -> auto-detect in GUI).

## Features
- **Clipboard Monitoring**: UI polls the clipboard for JSON matching the `ProjectOutput` schema.
- **Stash Auto-Refresh**: UI monitors the `.stashes` directory count and refreshes the list automatically if external changes (like `tidy`) occur.
- **Preview Limits**: To maintain UI responsiveness, only the first 10 modified files are rendered in the preview pane.