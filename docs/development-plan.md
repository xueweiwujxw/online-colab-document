# Development Plan

## Plan 1: Fix ONLYOFFICE Download Failed

Status: completed

Goal:

- Fix `Download failed` when opening Word / Excel files in ONLYOFFICE.

Scope:

- Keep ONLYOFFICE as the Word / Excel editor.
- Replace the MinIO presigned document URL in ONLYOFFICE config with a backend-controlled internal download URL.
- Keep permission checks centralized before generating the ONLYOFFICE config.
- Use a short-lived signed download ticket for ONLYOFFICE document fetches.

Acceptance:

- ONLYOFFICE config returns a document URL served by backend, not MinIO.
- ONLYOFFICE container can fetch the document through the backend URL.
- Existing backend tests pass.
- Frontend build still passes after any related type or API changes.

Notes:

- Switching to another Word / Excel editor is out of current repo constraints unless `AGENTS.md` / `TASK.md` are changed first.
