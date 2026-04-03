# Roadmap

## Phase 1 — Core Platform (done)

- [x] Auth (JWT + bcrypt)
- [x] Project / Environment / Resource CRUD
- [x] Resource lifecycle (start / stop / logs)
- [x] Port conflict detection
- [x] Environment variables with encryption
- [x] Docker runtime integration

## Phase 2 — Build Pipeline (done)

- [x] BuildKit integration
- [x] Git-based resource creation
- [x] Build job lifecycle management
- [x] Real-time build log streaming (WebSocket)
- [x] GitHub source connections (OAuth App + PAT)
- [x] Branch listing and repo browser

## Phase 3 — Developer Experience (in progress)

- [x] MySQL logical backup / restore with runner-based execution
- [x] MariaDB logical backup / restore with runner-based execution
- [x] PostgreSQL logical backup / restore with runner-based execution
- [x] MongoDB logical backup / restore with runner-based execution
- [x] CI/CD with GitHub Actions (Docker Hub image builds)
- [ ] Environment fork with resource cloning
- [ ] Resource templates (one-click Postgres, Redis, etc.)
- [ ] Container terminal improvements (resize, history)
- [x] Resource health checks and auto-restart (via CLI daemon)
- [x] CLI service management (list, status, restart, stop, start, logs)
- [x] Pluggable orchestrator driver interface (compose driver)
- [x] Daemon auto-install as system service (launchd/systemd)
- [ ] Deployment history and rollback

## Phase 4 — CLI Distribution & Multi-Node

- [ ] GoReleaser for CLI binary releases (GitHub Releases, Homebrew)
- [ ] K3s orchestrator driver
- [ ] Docker Swarm orchestrator driver
- [ ] Nomad orchestrator driver
- [ ] Multi-node: `tango join` for worker nodes
- [ ] Node management and scheduling

## Phase 5 — Collaboration & Ops

- [ ] Multi-user project access control
- [ ] Resource metrics (CPU, memory, network)
- [ ] Scheduled resource start/stop
- [ ] Webhook notifications on build/deploy events
- [ ] Audit log for resource and build actions
