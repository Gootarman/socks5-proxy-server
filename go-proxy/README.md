## go-proxy

Re-written proxy apps from Node.js to Go.

Motivation for rewriting in Go: when I deployed the proxy on a low-resource VPS that was also running other applications, I noticed that the previous implementation was consuming a disproportionate amount of disk space and RAM. This is especially critical on minimal VPS setups, so as an experiment I decided to try a minimal implementation in Go — and it resulted in a significant improvement in resource efficiency.

What's needed to be implemented:
- [x] socks5 proxy
- [x] proxy authentication
- [ ] last login user date saving
- [ ] data usage stats
- [ ] CLI commands for creating and deleting users and getting users stats
- [ ] Telegram bot with commands
- [ ] Telegram bot CLI commands for creating and deleting admin user

Additional features:
- [ ] In-memory cache for user authentication
- [ ] Prometheus metrics export
- [ ] Publishing proxy app as a separate image to Docker Hub
- [ ] Add different levels tests
- [ ] Linting and testing via Github Actions
