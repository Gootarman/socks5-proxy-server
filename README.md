# Socks5 proxy server
## Prerequisites
* Docker ([Install Docker](https://docs.docker.com/install/))

## How to run
- Copy *.env.example* file: `cp .env.example .env`
- Fill in configuration: `nano .env`. Fields:
  - **PROXY_IMAGE** - Docker image with prebuilt proxy binary (default in example: `ghcr.io/your-org/socks5-proxy-server-proxy:latest`),
  - **APP_PORT** - proxy server port (default: 54321),
  - **LOG_LEVEL** - log level (default: INFO),
  - **REQUIRE_AUTH** - if set to 1, anonymous users are not allowed.
- Pull and start application: `docker compose pull && docker compose up -d`

## Release and deploy (prebuilt proxy binary)
- Create and push a release tag:
  - `git tag v1.0.0`
  - `git push origin v1.0.0`
- GitHub Actions workflow `proxy release image` will:
  - run `go test ./...` in `proxy`,
  - run lint checks in `proxy`,
  - run `make build` in `proxy` to produce `proxy/dist/proxy`,
  - build Docker image from `proxy/Dockerfile` (using prebuilt `proxy/dist/proxy`),
  - publish image to GHCR: `ghcr.io/<owner>/socks5-proxy-server-proxy:v1.0.0`.
- On server set `PROXY_IMAGE` in `.env`, for example:
  - `PROXY_IMAGE=ghcr.io/<owner>/socks5-proxy-server-proxy:v1.0.0`
- Deploy update:
  - `docker compose pull`
  - `docker compose up -d`

## CLI commands
In all commands you need to call js-script in app docker container.  
So you need to find out container name with proxy application by running the following command:
```bash
docker compose ps
```

For example, it will be `socks5-proxy-server-proxy-1`.

In all the following commands you need to replace `socks5-proxy-server-proxy-1` with the yours container name. 

### Create user
```bash
docker exec -it socks5-proxy-server-proxy-1 sh -c 'exec node scripts/create-user.js'
```

### Delete user
```bash
docker exec -it socks5-proxy-server-proxy-1 sh -c 'exec node scripts/delete-user.js'
```

### Show users statistics
```bash
docker exec -it socks5-proxy-server-proxy-1 sh -c 'exec node scripts/users-stats.js'
```

## Telegram bot for administration
### Configuration
- Initialize bot at @botfather, get API token
- Set params in .env:
  - PUBLIC_URL - URL to server. E.g. http://proxy.domain.com:8443
  - TELEGRAM_API_TOKEN - API token from BotFather
  - TELEGRAM_WEBHOOK_URL - default: /webhook
  - TELEGRAM_USE_WEBHOOKS - 1 - use webhooks, 0 - use polling. To use webhooks you need to generate ssl certificates

- Generate self-signed SSL certificates for webhook mode:
```bash
mkdir -p ssl && openssl req -x509 -newkey rsa:4096 -sha256 -nodes -keyout ssl/key.pem -out ssl/crt.pem -days 365 -subj "/CN=<YOUR_IP_ADDRESS_OR_DOMAIN>"
```
- Create admin:
```bash
docker exec -it socks5-proxy-server-telegram_bot-1 sh -c 'exec node scripts/create-admin.js' 
```

You also can delete admin via script, if you need:
```bash
docker exec -it socks5-proxy-server-telegram_bot-1 sh -c 'exec node scripts/delete-admin.js' 
```

### Available commands
- `/users_stats` - show data usage statistics per user
- `/create_user` - create new proxy user
- `/delete_user` - delete proxy user
- `/get_users` - get list of proxy users
- `/generate_pass [length]` - generate random password with specified length (10 by default)
