<p align="center">
  <!-- TODO: Replace with a real banner image -->
  <img src="docs/images/banner-placeholder.svg" alt="xelanote banner" width="100%" />
</p>

<p align="center">
  <!-- TODO: Replace with a high-res logo if available -->
  <img src="frontend/static/icon-192.png" alt="xelanote logo" width="96" height="96" />
</p>

<h1 align="center">xelanote</h1>

<p align="center">
  <strong>Self-hosted, encrypted note-taking with Wikilinks, Backlinks, and Full-Text Search.</strong>
  <br />
  <em>Your knowledge, your server, your rules.</em>
</p>

<p align="center">
  <a href="https://github.com/xela-io/xelanote/actions/workflows/ci.yml"><img src="https://github.com/xela-io/xelanote/actions/workflows/ci.yml/badge.svg" alt="CI Status" /></a>
  <a href="https://github.com/xela-io/xelanote/actions/workflows/quality.yml"><img src="https://github.com/xela-io/xelanote/actions/workflows/quality.yml/badge.svg" alt="Quality Status" /></a>
  <a href="https://github.com/xela-io/xelanote/releases"><img src="https://img.shields.io/github/v/release/xela-io/xelanote?include_prereleases&label=release" alt="Release" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT" /></a>
  <a href="https://hub.docker.com/r/xelaio/xelanote"><img src="https://img.shields.io/badge/docker-ready-2496ED?logo=docker&logoColor=white" alt="Docker" /></a>
</p>

<p align="center">
  <a href="#features">Features</a>&nbsp;&nbsp;&bull;&nbsp;&nbsp;
  <a href="#installation">Installation</a>&nbsp;&nbsp;&bull;&nbsp;&nbsp;
  <a href="#usage">Usage</a>&nbsp;&nbsp;&bull;&nbsp;&nbsp;
  <a href="#configuration">Configuration</a>&nbsp;&nbsp;&bull;&nbsp;&nbsp;
  <a href="#contributing">Contributing</a>&nbsp;&nbsp;&bull;&nbsp;&nbsp;
  <a href="#license">License</a>
</p>

---

## Project Description

xelanote is a privacy-first, self-hosted note app that brings modern knowledge management features
(Wikilinks, backlinks, graph view, offline mode) together with optional end-to-end encryption. It
runs as a single Go binary with an embedded SvelteKit frontend and stores everything in a single
SQLite database file.

---

## Features

- Wikilinks and backlinks with automatic reference tracking
- Full-text search (server-side FTS5 and optional client-side search for encrypted notes)
- Folder hierarchy, tags, and version history
- Optional end-to-end encryption (AES-256-GCM)
- Two-factor authentication (TOTP and WebAuthn)
- Offline-first sync with conflict handling
- Responsive UI with multiple themes
- Docker-first deployment and single-binary builds

---

## Installation

**Prerequisites**

- Go 1.24+
- Node.js 20+
- GCC (for SQLite CGO)

**Local Development**

```bash
# Clone
git clone https://github.com/xela-io/xelanote.git
cd xelanote

# Install dependencies
make init

# Set required secret (min. 64 characters)
export JWT_SECRET="$(openssl rand -hex 32)"
```

---

## Usage

**Run locally**

```bash
# Terminal 1: Start backend (port 8080)
make run-backend

# Terminal 2: Start frontend dev server (port 5173)
make run-frontend
```

Open `http://localhost:5173` and create your first account.

**Docker (recommended)**

```bash
cat > .env.local << 'ENVEOF'
JWT_SECRET=your-secret-here-min-64-chars-use-openssl-rand-hex-32
XELANOTE_ENV=production
CORS_ALLOWED_ORIGINS=https://notes.example.com
ENVEOF

docker compose --env-file .env.local up -d --build
```

Open `http://localhost:8080` and create your first account.

**API example**

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"your-password"}'
```

Full API documentation: `docs/api.md`.

---

## Configuration

**Required**

| Variable | Description |
| --- | --- |
| `JWT_SECRET` | Min. 64 characters. Generate: `openssl rand -hex 32` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated origins for production (e.g. `https://notes.example.com`) |

**Optional**

| Variable | Default | Description |
| --- | --- | --- |
| `XELANOTE_DB` | `./data/xelanote.db` | Path to SQLite database |
| `XELANOTE_ENV` | `development` | Set to `production` for secure cookies and hardened defaults |
| `XELANOTE_DB_KEY` | — | SQLCipher encryption key for database-at-rest encryption |
| `XELANOTE_DB_KEY_FILE` | — | Path to file containing the SQLCipher key |

---

## Frontend API Structure

Frontend API calls live in `frontend/src/lib/api/`. The entry point is `frontend/src/lib/api.ts`,
which re-exports the module APIs. Use `import * as api from '$lib/api'` at call sites.
Modul-Snapshot: siehe `REFACTORING_REPORT.md`.
| `TURNSTILE_SECRET_KEY` | — | Cloudflare Turnstile CAPTCHA secret |
| `TURNSTILE_SITE_KEY` | — | Cloudflare Turnstile CAPTCHA site key |
| `PPROF_ENABLED` | `false` | Enable Go pprof profiling endpoint |

Full list: `docs/environment-variables.md`.

---

## Screenshots

<!-- TODO: Replace with real screenshots -->

<p align="center">
  <img src="docs/images/screenshot-editor-placeholder.svg" alt="Editor screenshot" width="800" />
</p>

<p align="center">
  <img src="docs/images/screenshot-graph-placeholder.svg" alt="Graph screenshot" width="800" />
</p>

---

## Contributing

Contributions are welcome. Please see `CONTRIBUTING.md` for workflow, style, and test guidelines.

---

## License

This project is licensed under the MIT License. See `LICENSE` for details.
