# GoCtx Project Instructions

## Surgical Patching Protocol

To improve efficiency and clarity, use **Surgical SEARCH/REPLACE blocks** for updates to existing files. This is the only patch format supported—simple, direct, and human-readable.

### Rules for Surgical Edits:

1. **Uniqueness vs. Brevity**: Provide enough context lines (3-5) to ensure a unique match, but avoid excessive vertical context. This optimizes the granular diff view for the human reviewer.
2. **Exact Match**: The content in the `SEARCH` block must be an exact character-for-character match (including indentation) of the target file.
3. **Native Dialect Format**: Use the raw format for clipboard transfer. This is direct and avoids any escaping issues.

   ```text
   "internal/pkg/file.go":
   <<<<<< SEARCH
   func Old() {
       return
   }
   ======
   func New() {
       return "updated"
   }
   >>>>>> REPLACE
   ```

4. **New Files**: If creating a brand new file, provide the full content as a text block without SEARCH/REPLACE markers.

5. **Single File, Single Purpose**: Each patch should target one file with one logical change. If multiple files or multiple independent changes are needed, create separate patches for each.

---

## AI Agent Persona

You are an expert Go developer. When modifying the `internal/ui` package, ensure GTK widgets are handled safely and idle-callbacks (`glib.IdleAdd`) are used for UI updates from goroutines.

## Project Context

GoCtx is a high-integrity orchestrator for surgical code patches. It utilizes a "Stash-Apply-Rollback" pattern: every operation is backed by a native Git stash with automated recovery logic to ensure workspace consistency. The GUI serves as the primary dashboard for context rendering and patch validation.
