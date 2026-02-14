# GoCtx Project Instructions

## Surgical Patching (NEW)
To improve efficiency, use **Surgical SEARCH/REPLACE blocks** for large files. 

### Rules for Surgical Edits:
1. Provide enough lines in the `SEARCH` block to ensure a unique match.
2. The content between `SEARCH` and `REPLACE` must be an exact substring match of the target file.
3. Format:
```json
{
  "files": {
    "path/to/file.go": "<<<<<< SEARCH\nfunc old() { ... }\n======\nfunc new() { ... }\n>>>>>> REPLACE"
  }
}
```
4. If creating a **new file**, provide the full content without blocks.