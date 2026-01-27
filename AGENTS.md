# AGENTS.md

## Project overview
- This repo contains multiple services: `go-proxy` (Go SOCKS5 proxy), `proxy` (Node.js app), and `telegram-bot` (Node.js bot).
- `docker-compose.yml` wires services together with Redis.

## Service capabilities
### proxy (Node.js SOCKS5 server)
- Starts a SOCKS5 proxy with optional authentication when `REQUIRE_AUTH=1`.
- Validates credentials against Redis (`REDIS.AUTH_USER_KEY`), using bcrypt to compare passwords.
- Tracks per-user data usage in Redis on each proxied data event.
- Updates last-auth timestamps for users and logs auth/proxy events.
- Runs a monthly job to clear data-usage stats.

### telegram-bot (Node.js admin bot)
- Telegram bot for proxy administration; supports polling or webhook mode.
- Optional outbound SOCKS5 for Telegram API via `PROXY_SOCKS5_*` env vars.
- Admin-only commands: list users, create user, delete user, and show usage stats.
- Guides admins through multi-step flows to create/delete users with validation.
- Generates passwords on demand and sends connection details (host/port/credentials).

## Common commands
- Go proxy tests: `cd go-proxy && go test ./...`
- Build Go binary for Docker (expected at `go-proxy/dist/proxy`): `cd go-proxy && make build` (or your local build flow)
- Docker compose: `docker compose up --build`

## Conventions
- Prefer small, focused changes; keep Go code idiomatic and Node code consistent with existing style.
- Do not add dependencies unless needed; update `go.mod`/`go.sum` or `package.json`/lockfiles accordingly.
- Keep configuration in env vars; see `.env` and `.env.example`.

## Notes for Docker
- `go-proxy/Dockerfile` expects a prebuilt binary at `go-proxy/dist/proxy`.
- Redis is required; services should point to `REDIS_HOST=redis`.
