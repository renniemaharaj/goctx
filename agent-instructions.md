# GoCtx Project Instructions

## Surgical Patching Protocol
To improve efficiency and reliability, use **Surgical SEARCH/REPLACE blocks** for updates to existing files. 

### Rules for Surgical Edits:
1. **Uniqueness vs. Brevity**: Provide enough context lines (3-5) to ensure a unique match, but avoid excessive vertical context. This optimizes the granular diff view for the human reviewer.
2. **Exact Match**: The content in the `SEARCH` block must be an exact character-for-character match (including indentation) of the target file.
3. **Formatting**: Always wrap the markers in the JSON value. Use `\n` for newlines.
\nfunc new() { ... }\n======\nfunc new() { ... }\n>>>>>> REPLACE"
  }
}
```

4. **New Files**: If creating a brand new file (or performing a total recovery), provide the full content as a string without SEARCH/REPLACE markers.

---

## AI Agent Persona
You are an expert Go developer. When modifying the `internal/ui` package, ensure GTK widgets are handled safely and idle-callbacks (`glib.IdleAdd`) are used for UI updates from goroutines.

## Project Context
GoCtx is a high-integrity orchestrator for surgical code patches. It utilizes a "Stash-Apply-Rollback" pattern: every operation is backed by a native Git stash with automated recovery logic to ensure workspace consistency. The GUI serves as the primary dashboard for context rendering and patch validation.