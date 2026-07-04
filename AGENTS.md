# Repository Guidelines

## Project Structure & Module Organization

PFChat is split into a Go backend, Vue frontend, and Dockerized MySQL setup. Backend code lives in `backend/` as one `package main`; handlers are grouped by topic in files such as `handler.go`, `grammar_handler.go`, and `session_handler.go`, while schema structs live in `model.go`. Frontend code lives in `frontend/src/` with `views/`, `components/`, `stores/`, `router/`, `api/`, `i18n/`, and `utils/`. Database and container setup is in `docker/`; runtime logs and PID files are written under `logs/`.

## Build, Test, and Development Commands

- `cd backend && go run .`: run the API server on port 8080; GORM migrations run at startup.
- `cd backend && go build -o pfchat .`: build the backend binary.
- `cd backend && go mod tidy`: clean up Go dependencies.
- `cd frontend && npm install`: install frontend dependencies. Both npm and pnpm lockfiles are present; avoid updating both unless intended.
- `cd frontend && npm run dev`: start Vite on port 3000.
- `cd frontend && npm run build`: produce the frontend bundle in `frontend/dist/`.
- `cd docker && docker compose up -d`: start MySQL.
- `./start.sh` and `./stop.sh`: Linux/macOS helper scripts for the full stack; they use `sudo` and are not PowerShell-friendly.

## Coding Style & Naming Conventions

Format Go with `gofmt`; keep backend additions in the flat `backend/` package unless a larger refactor is requested. Vue files use PascalCase, such as `Chat.vue` and `LocaleSwitcher.vue`. JavaScript modules use camelCase exports and keep API calls centralized through `frontend/src/api/index.js`. Preserve country-code and error-type constants across backend and frontend when changing pragmatic-failure behavior.

## Testing Guidelines

No automated test suite is configured: there are no `*_test.go` files and no frontend test script. For backend changes, add focused Go tests when introducing testable logic, then run `go test ./...` from `backend/`. For frontend changes, at minimum run `npm run build` and manually verify affected routes.

## Commit & Pull Request Guidelines

Recent commits use short Chinese summaries and sometimes a scope-style prefix. Keep commits concise and behavior-focused. Pull requests should include a clear description, local verification commands, linked issues when available, and screenshots for UI changes.

## Security & Configuration Tips

Secrets are currently hard-coded in places such as `backend/llm.go` and `backend/auth.go`; do not rotate or expose them casually. Prefer moving new configuration to environment variables. The frontend WebSocket URL may be deployment-specific, so verify it when switching between local and hosted environments.
