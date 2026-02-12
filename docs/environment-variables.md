# Environment Variables Reference

Complete reference for all environment variables used by xelanote.

## Required

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | HMAC-SHA256 signing key for JWT tokens. **Min. 64 characters.** Generate with `openssl rand -hex 32`. Server refuses to start without this. |
| `CORS_ALLOWED_ORIGINS` | Comma-separated list of allowed origins (e.g., `https://notes.example.com`). **Required when `XELANOTE_ENV=production`** (server refuses to start). In development, allows all origins with a warning. |

## Optional - Application

| Variable | Default | Description |
|----------|---------|-------------|
| `XELANOTE_ENV` | `development` | Set to `production` for secure cookies (HttpOnly, Secure, SameSite=Strict). |
| `XELANOTE_DB` | `./data/xelanote.db` | Path to SQLite database file. Can also be set via `-db` CLI flag. |
| `XELANOTE_DB_KEY` | — | SQLCipher encryption key for database-at-rest encryption. |
| `XELANOTE_DB_KEY_FILE` | — | Path to file containing the SQLCipher key (alternative to `XELANOTE_DB_KEY`). |
| `XELANOTE_API_KEY_SECRET` | value of `JWT_SECRET` | Secret for encrypting stored API keys. Falls back to `JWT_SECRET` if not set. |
| `TRUSTED_PROXIES` | `127.0.0.1/32,::1/128` | Comma-separated list of trusted proxy CIDRs for `X-Forwarded-For` parsing. **Required when `XELANOTE_ENV=production`**. |
| `XELANOTE_BOOTSTRAP_TOKEN` | — | One-time bootstrap token for first admin creation when registration is disabled. Send as `bootstrap_token` in `POST /api/auth/register`. |

## Optional - WebAuthn/FIDO2

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBAUTHN_RP_ID` | `localhost` (dev only) | Required in production. Relying Party ID for WebAuthn. |
| `WEBAUTHN_RP_ORIGINS` | derived | Comma-separated allowed origins for WebAuthn. Falls back to `CORS_ALLOWED_ORIGINS`, then `http://localhost:5173,http://localhost:8080`. |

## Optional - CAPTCHA

| Variable | Default | Description |
|----------|---------|-------------|
| `TURNSTILE_SECRET_KEY` | — | Cloudflare Turnstile server-side secret key. CAPTCHA disabled if not set. |
| `TURNSTILE_SITE_KEY` | — | Cloudflare Turnstile client-side site key. |

## Optional - AI/LLM

| Variable | Default | Description |
|----------|---------|-------------|
| `GEMINI_MODEL` | `gemini-2.5-flash` | Override the default Gemini model for LLM features. |

> **Note:** Claude and Gemini API keys are configured per-user in Settings, not via environment variables.

## Optional - Error Reporting (Forgejo)

| Variable | Default | Description |
|----------|---------|-------------|
| `FORGEJO_URL` | — | Forgejo base URL for error reporting issues. |
| `FORGEJO_REPO` | — | Forgejo repo in `owner/name` format for error reporting issues. |
| `FORGEJO_API_TOKEN` | — | Forgejo API token for creating issues/comments. |

## Optional - Development/Debugging

| Variable | Default | Description |
|----------|---------|-------------|
| `PPROF_ENABLED` | `false` | Set to `true` to enable Go pprof profiling on `127.0.0.1:6060`. Localhost only, use SSH tunnel for remote access. |

## CLI Flags

The server binary also accepts command-line flags:

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:8080` | HTTP listen address and port. |
| `-db` | — | Database path (overrides `XELANOTE_DB`). |

## Frontend Build-Time Variables

The SvelteKit frontend uses Vite's `import.meta.env.DEV` to detect development mode. In development, API requests are proxied to `http://localhost:8080`. In production, the frontend is served by the Go backend and uses relative paths (`/api`).

No custom `VITE_*` environment variables are required.

## Example .env File

```bash
# Required
JWT_SECRET=your-64-char-secret-here-generate-with-openssl-rand-hex-32-which-gives-64-chars

# Production
XELANOTE_ENV=production
CORS_ALLOWED_ORIGINS=https://notes.example.com

# Optional: Database encryption
# XELANOTE_DB_KEY=your-database-encryption-key
# XELANOTE_DB=/custom/path/to/xelanote.db

# Optional: CAPTCHA
# TURNSTILE_SECRET_KEY=0x...
# TURNSTILE_SITE_KEY=0x...

# Optional: Reverse proxy
# TRUSTED_PROXIES=172.17.0.1/32,10.0.0.1/32

# Optional: First admin bootstrap when registration is disabled
# XELANOTE_BOOTSTRAP_TOKEN=long-random-secret

# Optional: Error reporting to Forgejo (see docs/error-reporting.md)
# FORGEJO_URL=https://git.example.com
# FORGEJO_REPO=owner/reponame
# FORGEJO_API_TOKEN=gta_your_token_here
```
