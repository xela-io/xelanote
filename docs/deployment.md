# XelaNote Deployment Guide

## Production Deployment (Docker)

### Environment

- **Target**: `<STAGING_USER>@<STAGING_IP>`
- **Port**: 8081 (external) → 8080 (internal)
- **Public URL**: https://<STAGING_URL> (via nginx-proxy-manager)
- **Internal URL**: http://<STAGING_IP>:8081
- **Database**: SQLite with persistent Docker volume
- **Repository**: https://<FORGEJO_URL>/xela/xelanote (Forgejo, primary) / https://github.com/xela-io/xelanote (GitHub, mirror)
- **CI/CD**: Forgejo Actions (Staging: auto-deploy auf Push zu `main`, Production: Tag-basiert `v*` oder manuell)
- **Network**: `nginx_default` (für nginx-proxy-manager Integration)

---

## Quick Deploy (TL;DR)

**Staging (automatisch):** Jeder Push auf `forgejo main` deployed automatisch via Forgejo Actions. Kein manueller Eingriff noetig.

```bash
# Commit & Push - Staging-Deploy startet automatisch
git add . && git commit -m "fix: description" && git push forgejo main

# Verify (nach ~1-2 Minuten)
curl https://<STAGING_URL>/health
```

**Hetzner Production (automatisch via Tag):**

```bash
# Nach erfolgreichem Staging-Test: Tag erstellen und pushen
git tag v1.2.3
git push forgejo v1.2.3

# Production-Deploy startet automatisch
# Verify (nach ~1-2 Minuten)
curl https://xelanote.com/health
```

Alternativ manueller Trigger via Forgejo UI oder [Quick Deploy auf Hetzner](#quick-deploy-auf-hetzner) als Fallback.

**Wichtig:** Secrets werden aus `~/.xelanote.env` geladen (chmod 600). Niemals Secrets in der Command-Line!

---

## CI/CD: Forgejo Actions Auto-Deploy (Staging)

Seit 2026-02-06 wird das Staging-Deployment vollautomatisch ueber Forgejo Actions ausgefuehrt.

### Uebersicht

| Eigenschaft | Wert |
|-------------|------|
| **Workflow** | `.forgejo/workflows/deploy-staging.yml` |
| **Trigger** | Push auf `main` Branch |
| **Runner** | Forgejo Actions Runner auf Staging-Server (<STAGING_IP>) |
| **Runner-Label** | `staging:host` (laeuft direkt auf dem Host, nicht in Docker) |
| **Ziel** | https://<STAGING_URL> |
| **Build-Zeit** | ~40 Sekunden (mit Docker-Cache) |
| **Downtime** | Minimal (~5-10 Sekunden waehrend Container-Wechsel) |

### Ablauf

1. **Push auf `forgejo main`** triggered den Workflow
2. **Checkout** (SHA-pinned `actions/checkout@v4`)
3. **Pre-Flight Checks:**
   - Env-File lesbar (`/home/container/.xelanote.env`)
   - Pflicht-Variablen vorhanden (`JWT_SECRET`, `XELANOTE_API_KEY_SECRET`, `CORS_ALLOWED_ORIGINS`, `XELANOTE_DB`, `XELANOTE_ENV`, `TRUSTED_PROXIES`)
   - `JWT_SECRET` und `XELANOTE_API_KEY_SECRET` mindestens 64 Zeichen
   - Docker-Daemon erreichbar
   - Docker-Netzwerk `nginx_default` existiert
   - Vorheriges Image fuer Rollback gespeichert
4. **Docker Build** mit SHA-Tag (`xelanote:<12-stelliger-sha>`) + `latest`
5. **Stop/Remove** des alten Containers (30s Grace Period)
6. **Start** des neuen Containers mit Security-Hardening
7. **Health Check** (max 30 Versuche, 2s Intervall = 60s Timeout)
8. **Auto-Rollback** auf vorheriges Image bei Health-Check-Failure
9. **Image Pruning** (dangling Images entfernen)
10. **Deployment Summary** in Forgejo UI

### Security-Hardening des Containers

Der Workflow startet den Container mit folgenden Sicherheits-Optionen:

```
--read-only              # Read-only Root-Filesystem
--tmpfs /tmp             # Schreibbarer tmp-Bereich
--cap-drop ALL           # Alle Linux Capabilities entfernt
--security-opt no-new-privileges
--memory=512m --cpus=1 --pids-limit=200
```

### Runner-Setup

Das Setup-Script `scripts/setup-forgejo-runner.sh` automatisiert die Installation:

```bash
# Auf dem Staging-Server als root ausfuehren:
sudo bash scripts/setup-forgejo-runner.sh

# Danach: Runner registrieren (Token aus Forgejo holen)
sudo -u forgejo-runner forgejo-runner register \
  --instance https://<FORGEJO_URL> \
  --token <TOKEN> \
  --name staging-runner \
  --labels staging:host \
  --no-interactive

# Runner starten
sudo systemctl start forgejo-runner
```

Das Script erstellt:
- System-User `forgejo-runner` (kein Login-Shell)
- POSIX ACL auf Env-File (nur Lese-Zugriff)
- Systemd-Service mit Security-Hardening (`ProtectSystem=strict`, `NoNewPrivileges=true`)
- Fuegt User zur Docker-Gruppe hinzu

### Wichtige Hinweise / Lessons Learned

**Node.js auf dem Runner-Host erforderlich:**
`actions/checkout` benoetigt Node.js. Ohne Node.js schlaegt der Checkout-Schritt fehl.

```bash
# Installation (Debian/Ubuntu)
sudo apt install nodejs
```

**`TRUSTED_PROXIES` in Env-File setzen:**
Im Production-Mode (`XELANOTE_ENV=production`) muss `TRUSTED_PROXIES` gesetzt sein, sonst startet der Server nicht (`log.Fatal`). Format: Komma-getrennte CIDRs.

```bash
# In ~/.xelanote.env hinzufuegen:
# Caddy/Nginx auf dem gleichen Host:
TRUSTED_PROXIES=127.0.0.1/32
# Docker-Netzwerk (z.B. nginx-proxy-manager):
# TRUSTED_PROXIES=172.18.0.0/16
```

**`WEBAUTHN_RP_ID` in Env-File setzen:**
Im Production-Mode (`XELANOTE_ENV=production`) muss `WEBAUTHN_RP_ID` gesetzt sein, sonst schlaegt WebAuthn fehl.

```bash
# In ~/.xelanote.env hinzufuegen:
WEBAUTHN_RP_ID=<STAGING_URL>
```

**Directory ACL fuer Env-File-Zugriff:**
Der Runner-User braucht Execute-Rechte auf das Parent-Directory des Env-Files:

```bash
setfacl -m u:forgejo-runner:x /home/container
```

**`docker restart` liest `--env-file` NICHT neu:**
Wenn Environment-Variablen geaendert wurden, muss der Container neu erstellt werden (stop/rm/run). Ein einfaches `docker restart` verwendet die alten Werte.

```bash
# FALSCH - env-file Aenderungen werden ignoriert:
docker restart xelanote

# RICHTIG - Container wird mit neuem env-file erstellt:
docker stop xelanote && docker rm xelanote
docker run -d --name xelanote --env-file ~/.xelanote.env ...
```

### CI/CD: Production Auto-Deploy (Hetzner)

Seit 2026-02-06 wird auch das Production-Deployment ueber Forgejo Actions ausgefuehrt.

| Eigenschaft | Wert |
|-------------|------|
| **Workflow** | `.forgejo/workflows/deploy-production.yml` |
| **Trigger** | Tag `v*` oder manueller Trigger (workflow_dispatch) |
| **Runner** | Forgejo Actions Runner auf Hetzner (<PRODUCTION_IP>) |
| **Runner-Label** | `production:host` |
| **Ziel** | https://xelanote.com |

**Normaler Release-Flow:**
```bash
# 1. Code auf Staging testen (auto-deploy via push)
git push forgejo main
curl https://<STAGING_URL>/health

# 2. Tag erstellen und pushen -> Production-Deploy startet
git tag v1.2.3
git push forgejo v1.2.3

# 3. Verify
curl https://xelanote.com/health
```

**Hotfix (manuell):** Repository -> Actions -> deploy-production -> Run workflow -> Reason eingeben.

**Unterschiede zu Staging:** Pre-Deploy Datenbank-Backup, XELANOTE_ENV-Validierung, 5 statt 3 Image-Retention, Version-Tag im Image-Name.

Vollstaendige Dokumentation: [Forgejo Runner Setup](./forgejo-runner-setup.md)

### Manueller Staging-Deploy (Fallback)

Falls der automatische Deploy nicht funktioniert oder Aenderungen direkt deployt werden sollen:

```bash
# 1. Server: Pull, Build, Deploy
ssh <STAGING_USER>@<STAGING_IP> "cd ~/xelanote && git pull && docker build -t xelanote:latest ."
ssh <STAGING_USER>@<STAGING_IP> "docker stop xelanote && docker rm xelanote"
ssh <STAGING_USER>@<STAGING_IP> 'docker run -d --name xelanote --restart unless-stopped \
  -p 8081:8080 --network nginx_default \
  -v xelanote_xelanote-data:/app/data \
  --env-file ~/.xelanote.env \
  xelanote:latest'

# 2. Verify
curl https://<STAGING_URL>/health
```

---

## Prerequisites

### 1. SSH Access
```bash
ssh <STAGING_USER>@<STAGING_IP>
```

### 2. GitHub SSH Key (One-time setup)
```bash
# Generate key
ssh-keygen -t ed25519 -C "<STAGING_USER>@<STAGING_IP>-xelanote" -f ~/.ssh/id_ed25519 -N ""

# Add GitHub to known hosts
ssh-keyscan -t ed25519 github.com >> ~/.ssh/known_hosts

# Display public key
cat ~/.ssh/id_ed25519.pub
```

Add the public key to GitHub: https://github.com/settings/keys

---

## Initial Deployment

### 1. Clone Repository
```bash
ssh <STAGING_USER>@<STAGING_IP>
git clone git@github.com:xela-io/xelanote.git
cd xelanote
```

### 2. Configure Environment
```bash
# Create .env file with secrets
cat <<EOF > .env
JWT_SECRET=$(openssl rand -hex 32)
XELANOTE_API_KEY_SECRET=$(openssl rand -hex 32)
XELANOTE_ENV=production
EOF
```

### 3. Build and Start
```bash
docker compose up -d --build
```

Hinweis: `docker-compose.yml` baut ueber das root `Dockerfile` (FTS5, ohne SQLCipher). Fuer SQLCipher bitte mit `backend/Dockerfile` aus dem Repo-Root bauen.

### 4. Verify Deployment
```bash
# Check container status
docker ps | grep xelanote

# Check logs
docker logs xelanote -f

# Test health endpoint
curl http://localhost:8081/health
```

---

## Update Deployment (Schritt für Schritt)

> **Hinweis:** Fuer Staging wird automatisch deployed (siehe [CI/CD](#cicd-forgejo-actions-auto-deploy-staging)).
> Die folgenden Schritte sind nur fuer manuelles Deployment oder Hetzner Production relevant.

### 1. Code pushen (lokal)
```bash
git add .
git commit -m "fix: Beschreibung der Änderung

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
git push forgejo main
```

**Staging:** Deploy startet automatisch via Forgejo Actions. Weiter bei Schritt 5 zum Verifizieren.

**Hetzner Production:** Weiter mit Schritt 2-5 (manuell).

### 2. Code auf Server ziehen (nur Hetzner)
```bash
ssh <PROD_SSH_ALIAS> "cd ~/xelanote && git pull"
```

### 3. Docker Image bauen (nur Hetzner)
```bash
ssh <PROD_SSH_ALIAS> "cd ~/xelanote && sudo docker build -t xelanote:latest ."
```

### 4. Container neu starten (nur Hetzner)
```bash
ssh <PROD_SSH_ALIAS> "sudo docker stop xelanote && sudo docker rm xelanote"

ssh <PROD_SSH_ALIAS> 'sudo docker run -d --name xelanote --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v ~/xelanote-data:/app/data \
  --memory=512m --cpus=1 --security-opt no-new-privileges --pids-limit=200 \
  --env-file ~/.xelanote.env \
  xelanote:latest'
```

**Hinweis:** Secrets werden aus `~/.xelanote.env` geladen. Siehe [deployment-security.md](./deployment-security.md).

### 5. Verifizieren
```bash
# Staging (automatisch deployed)
curl https://<STAGING_URL>/health

# Hetzner Production
curl https://xelanote.com/health
ssh <PROD_SSH_ALIAS> "sudo docker logs xelanote --tail 20"
```

---

## Umgebungsvariablen

| Variable | Wert | Beschreibung |
|----------|------|--------------|
| `JWT_SECRET` | (siehe Server `.env`) | Auth-Token-Signierung (64 Zeichen hex) |
| `XELANOTE_API_KEY_SECRET` | (siehe Server `.env`) | Schluessel fuer API-Key-Verschluesselung (64 Zeichen hex, muss sich von `JWT_SECRET` unterscheiden) |
| `XELANOTE_DB` | `/app/data/xelanote.db` | Datenbank-Pfad im Container |
| `XELANOTE_ENV` | `production` | Aktiviert Secure Cookies (SameSite=Strict) |
| `CORS_ALLOWED_ORIGINS` | `https://<STAGING_URL>` | Erlaubte Origins für CORS & WebSocket |
| `TRUSTED_PROXIES` | `127.0.0.1/32` | Trusted Reverse-Proxy CIDRs fuer X-Forwarded-For |

**Wichtig**: `CORS_ALLOWED_ORIGINS` MUSS gesetzt sein, sonst werden WebSocket-Verbindungen abgelehnt!

---

## Docker Run Referenz

Vollständiger Befehl für manuelles Deployment:

```bash
docker run -d --name xelanote --restart unless-stopped \
  -p 8081:8080 \
  --network nginx_default \
  -v xelanote_xelanote-data:/app/data \
  --log-driver json-file \
  --log-opt max-size=10m \
  --log-opt max-file=30 \
  --env-file ~/.xelanote.env \
  xelanote:latest
```

**Wichtig:** `~/.xelanote.env` muss folgende Variablen enthalten:
- `JWT_SECRET` (64+ Zeichen)
- `XELANOTE_API_KEY_SECRET` (64+ Zeichen, ungleich `JWT_SECRET`)
- `XELANOTE_DB=/app/data/xelanote.db`
- `XELANOTE_ENV=production`
- `CORS_ALLOWED_ORIGINS=https://your-domain.com`
- `TRUSTED_PROXIES=127.0.0.1/32` (CIDR des Reverse Proxy)

**Parameter erklärt:**
- `-d`: Detached mode (Hintergrund)
- `--restart unless-stopped`: Auto-Restart nach Reboot
- `-p 8081:8080`: Port-Mapping (extern:intern)
- `--network nginx_default`: Netzwerk für nginx-proxy-manager
- `-v xelanote_xelanote-data:/app/data`: Persistenter Datenbank-Speicher
- `--log-driver json-file`: JSON-basiertes Logging für strukturierte Logs
- `--log-opt max-size=10m`: Max 10MB pro Log-Datei
- `--log-opt max-file=30`: 30 Dateien = 30 Tage Retention (GDPR-konform)
- `-e ...`: Umgebungsvariablen

---

## Database Management

### Location
- **Inside Container**: `/app/data/xelanote.db`
- **Docker Volume**: `xelanote_xelanote-data`
- **Environment Variable**: `XELANOTE_DB=/app/data/xelanote.db`
- **Optional Encryption Key**: `XELANOTE_DB_KEY_FILE=/run/secrets/xelanote_db_key` (oder `XELANOTE_DB_KEY`)

### Encryption (SQLCipher, optional)
- Wenn `XELANOTE_DB_KEY_FILE`/`XELANOTE_DB_KEY` gesetzt ist, wird die DB verschlüsselt erstellt.
- Bestehende unverschlüsselte DBs müssen manuell migriert werden (SQLCipher CLI).

### Migration: unverschluesselte DB -> SQLCipher
**Voraussetzung**: `sqlcipher` CLI installiert.

1. Backup anlegen:
```bash
cp /app/data/xelanote.db /app/data/xelanote.db.bak
```

2. Verschluesselte DB erzeugen:
```bash
sqlcipher /app/data/xelanote.db <<'SQL'
PRAGMA key = 'NEW_SECRET_KEY';
ATTACH DATABASE '/app/data/xelanote-plain.db' AS plaintext KEY '';
SELECT sqlcipher_export('main', 'plaintext');
DETACH DATABASE plaintext;
SQL
```

3. Alte DB ersetzen:
```bash
mv /app/data/xelanote.db /app/data/xelanote.db.enc
mv /app/data/xelanote-plain.db /app/data/xelanote.db
```

4. Key setzen und Server starten:
```bash
export XELANOTE_DB_KEY='NEW_SECRET_KEY'
# oder: export XELANOTE_DB_KEY_FILE=/run/secrets/xelanote_db_key
```

**Hinweis**: Der Key darf nicht geloggt werden. Bewahre das Backup bis zum verifizierten Start auf.

### Backup Database
```bash
# Create backup inside container
ssh <STAGING_USER>@<STAGING_IP> "docker exec xelanote sqlite3 /app/data/xelanote.db '.backup /app/data/xelanote_backup_$(date +%Y%m%d_%H%M%S).db'"

# Copy backup to local machine
docker cp xelanote:/app/data/xelanote_backup_*.db ./
```

**Hinweis bei SQLCipher**: Für verschlüsselte DBs `sqlcipher` nutzen und den Key via `PRAGMA key` setzen.

### Restore Database
```bash
# Copy backup to container
docker cp ./xelanote_backup.db xelanote:/app/data/

# Stop container
docker compose down

# Replace database
docker exec xelanote cp /app/data/xelanote_backup.db /app/data/xelanote.db

# Start container
docker compose up -d
```

### Database Checks
```bash
# Integrity check
ssh <STAGING_USER>@<STAGING_IP> "docker exec xelanote sqlite3 /app/data/xelanote.db 'PRAGMA integrity_check;'"

# Check settings
ssh <STAGING_USER>@<STAGING_IP> "docker exec xelanote sqlite3 /app/data/xelanote.db 'PRAGMA journal_mode; PRAGMA foreign_keys; PRAGMA synchronous;'"

# List migrations
ssh <STAGING_USER>@<STAGING_IP> "docker exec xelanote sqlite3 /app/data/xelanote.db 'SELECT * FROM migrations ORDER BY applied_at;'"
```

---

## Troubleshooting

### Container Won't Start
```bash
# Check logs
docker logs xelanote

# Check if port is in use
ss -tlnp | grep 8081

# Remove old container and volume (CAUTION: deletes data!)
docker compose down -v
docker compose up -d --build
```

### Database Issues
```bash
# Stop container
docker compose down

# Run integrity check
docker run --rm -v xelanote_xelanote-data:/data alpine sh -c "apk add sqlite && sqlite3 /data/xelanote.db 'PRAGMA integrity_check;'"

# If corrupted, restore from backup (see above)
```

### Migration Failures
```bash
# Check applied migrations
docker exec xelanote sqlite3 /app/data/xelanote.db 'SELECT * FROM migrations;'

# If migration failed, check logs
docker logs xelanote | grep -i migration

# Manual fix: connect to DB and investigate
docker exec -it xelanote sqlite3 /app/data/xelanote.db
```

---

## Monitoring

### Health Check
```bash
# HTTP endpoint
curl http://<STAGING_IP>:8081/health

# Docker health status
docker ps --format 'table {{.Names}}\t{{.Status}}' | grep xelanote
```

### Logs
```bash
# Follow logs
docker logs xelanote -f

# Last 50 lines
docker logs xelanote --tail 50

# Errors only
docker logs xelanote 2>&1 | grep -i error
```

### Resource Usage
```bash
# Container stats
docker stats xelanote --no-stream

# Volume size
docker system df -v | grep xelanote
```

---

## Rollback

### Rollback to Previous Version
```bash
ssh <STAGING_USER>@<STAGING_IP>

cd xelanote

# Find previous commit
git log --oneline -5

# Checkout previous version
git checkout <commit-hash>

# Rebuild
docker compose up -d --build

# Verify
curl http://localhost:8081/health
```

### Return to Latest
```bash
git checkout main
git pull
docker compose up -d --build
```

---

## Security Notes

- JWT_SECRET is stored in `.env` (never commit!)
- Database is in persistent Docker volume (survives container restarts)
- No public exposure - accessed via local network only
- Consider adding Traefik/nginx reverse proxy with HTTPS for production

---

## Docker Compose Configuration

**File**: `docker-compose.yml`

```yaml
services:
  xelanote:
    build:
      context: .
      dockerfile: Dockerfile
    image: xelanote:latest
    container_name: xelanote
    ports:
      - "8081:8080"
    volumes:
      - xelanote-data:/app/data
    environment:
      - XELANOTE_DB=/app/data/xelanote.db
      - JWT_SECRET=${JWT_SECRET}
      - XELANOTE_DB_KEY_FILE=/run/secrets/xelanote_db_key
    secrets:
      - xelanote_db_key
    restart: unless-stopped
    # GDPR-compliant log rotation (30 days retention)
    logging:
      driver: json-file
      options:
        max-size: "10m"   # Max size per log file
        max-file: "30"    # 30 files = ~30 days retention
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - xelanote-net

volumes:
  xelanote-data:

secrets:
  xelanote_db_key:
    file: ./secrets/xelanote_db_key

networks:
  xelanote-net:
    driver: bridge
```

### Log Rotation (GDPR Compliance)

**Security audit logs are automatically rotated and deleted after 30 days.**

**What's logged:**
- Login attempts (identifier, IP, timestamp)
- Account lockouts (identifier, attempts, duration)
- Password/email changes (user_id, IP, timestamp)
- 2FA events (user_id, IP, timestamp)

**Why 30 days:**
- Security incident detection and forensic analysis
- GDPR Article 6(1)(f) - Legitimate interest in security monitoring
- Automatic deletion ensures minimal data retention

**View logs:**
```bash
# All logs
docker logs xelanote

# Recent login failures
docker logs xelanote 2>&1 | grep "login_failed"

# Account lockouts
docker logs xelanote 2>&1 | grep "account_locked"

# 2FA events
docker logs xelanote 2>&1 | grep "2fa_"
```

See [SECURITY.md](../SECURITY.md) for detailed security logging documentation.

---

## Build Configuration

**Multi-Stage Dockerfile:**

1. **Frontend Builder**: Node 22 Alpine for SvelteKit build
2. **Backend Builder**: Go 1.25 Alpine with CGO for SQLite/FTS5
3. **Final Image**: Alpine 3.20 with compiled binary and static assets

**Build Tags:**

| Umgebung | Tags | Hinweis |
|----------|------|---------|
| Lokal (Makefile) | `fts5 sqlite_crypt` | SQLCipher fuer DB-Encryption-at-Rest verfuegbar |
| Docker (Dockerfile) | `fts5` | SQLCipher **nicht** enthalten (opt-in, siehe Dockerfile-Kommentare) |
| CI (GitHub Actions) | `fts5 sqlite_crypt` | Wie lokal |

**Bewusste Entscheidung:** Docker verzichtet auf SQLCipher, weil die meisten Deployments DB-Encryption-at-Rest nicht benoetigen (Container-Volume + OS-Level-Encryption genuegen). Fuer SQLCipher im Docker-Image: siehe Kommentar in `Dockerfile` Zeile 17-18.

**Build Time**: ~40 seconds (with caching)

---

## Deployment History

### 2026-02-06
- **Status**: CI/CD Pipeline eingerichtet (Staging + Production)
- **Changes**:
  - Forgejo Actions Auto-Deploy fuer Staging aktiviert (Push auf `main`)
  - Forgejo Actions Auto-Deploy fuer Production aktiviert (Tag `v*` oder manuell)
  - Runner auf Staging-Server registriert (Label: `staging:host`)
  - Runner auf Hetzner Production registriert (Label: `production:host`)
  - Auto-Rollback bei Health-Check-Failure (beide Umgebungen)
  - Pre-Deploy Datenbank-Backup (nur Production)
  - Security-Hardening: read-only FS, cap-drop ALL, no-new-privileges
  - Dokumentation: `docs/forgejo-runner-setup.md`

### 2026-01-19
- **Status**: ✅ Successfully deployed
- **Commits**: `2398e8e` (Security Fixes)
- **Changes**:
  - SEC-001: WebSocket Origin-Validierung implementiert
  - SEC-004: Rate-Limit IP-Spoofing behoben
  - `CORS_ALLOWED_ORIGINS` als neue Umgebungsvariable
- **Neue Env-Var**: `CORS_ALLOWED_ORIGINS=https://<STAGING_URL>`
- **Tests**: Health Check ✓, WebSocket ✓, Auth ✓

### 2026-01-17 (Evening)
- **Status**: ✅ Successfully deployed
- **Commits**: `d399a5e` (FTS trigger fix)
- **Migrations**: 001-007, 009 applied
- **Issues Fixed**:
  - WAL mode → DELETE mode for Docker stability
  - Foreign keys explicitly enabled
  - FTS delete trigger fixed for trash permanent delete
- **Tests**: All features working (Trash, Undo/Redo, Toast, Auth)

---

## Hetzner Cloud Server (Produktion)

### Übersicht

| Eigenschaft | Wert |
|-------------|------|
| **Server** | Hetzner Cloud CX22 |
| **IP** | <PRODUCTION_IP> |
| **OS** | Ubuntu 24.04 LTS |
| **SSH** | `ssh <PROD_SSH_ALIAS>` (non-standard port, dedicated user) |
| **URL** | https://xelanote.com |
| **Domain** | xelanote.com (in Vorbereitung) |

### SSH-Zugang

```bash
# Über SSH-Config (empfohlen)
ssh <PROD_SSH_ALIAS>

# Oder manuell
ssh -i ~/.ssh/xelanote_server -p <SSH_PORT> <PROD_USER>@<PRODUCTION_IP>
```

**Lokale SSH-Config** (`~/.ssh/config`):
```
Host <PROD_SSH_ALIAS>
    HostName <PRODUCTION_IP>
    User <PROD_USER>
    Port <SSH_PORT>
    IdentityFile ~/.ssh/xelanote_server
```

---

### Server-Härtung (implementiert 2026-01-19)

#### 1. SSH-Härtung
- **Port**: Non-standard (nicht Port 22)
- **User**: Dedicated user mit sudo-Rechten
- **Root-Login**: Deaktiviert (`PermitRootLogin no`)
- **Authentifizierung**: Nur SSH-Keys (`PasswordAuthentication no`)
- **Max. Versuche**: 3 (`MaxAuthTries 3`)

**Config-Dateien:**
- `/etc/ssh/sshd_config.d/hardening.conf`
- `/etc/systemd/system/ssh.socket.d/override.conf` (Port <SSH_PORT>)

#### 2. Firewall (UFW)

```bash
sudo ufw status
```

| Port | Protokoll | Beschreibung |
|------|-----------|--------------|
| <SSH_PORT> | TCP | SSH |
| 80 | TCP | HTTP (Caddy) |
| 443 | TCP | HTTPS (Caddy) |
| 8080 | TCP | xelanote direkt (temporär) |

**Wichtig:** Auch Hetzner Cloud Firewall muss dieselben Ports freigeben!

#### 3. Fail2ban

- **Aktiviert für**: SSH (<SSH_PORT>)
- **Max. Versuche**: 3
- **Ban-Dauer**: 24 Stunden
- **Config**: `/etc/fail2ban/jail.local`

```bash
# Status prüfen
sudo fail2ban-client status sshd

# Gebannte IPs anzeigen
sudo fail2ban-client status sshd | grep "Banned IP"
```

#### 4. Automatische Sicherheitsupdates

- **Paket**: unattended-upgrades
- **Nur Security-Updates**: Ja
- **Auto-Reboot**: Nein
- **Config**: `/etc/apt/apt.conf.d/51custom-unattended-upgrades`

```bash
# Status prüfen
sudo systemctl status unattended-upgrades
```

#### 5. Docker-Härtung

**Daemon-Config** (`/etc/docker/daemon.json`):
```json
{
  "live-restore": true,
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

**Container Security-Optionen:**
- `--memory=512m` - RAM-Limit
- `--cpus=1` - CPU-Limit
- `--security-opt no-new-privileges` - Privilege Escalation verhindern
- `--pids-limit=200` - Fork-Bomb-Schutz

#### 6. Backup-Strategie

**Backup-Skript**: `/root/backup-xelanote.sh`
- **Frequenz**: Täglich um 3:00 UTC (Cronjob)
- **Speicherort**: `<BACKUP_DIR>/`
- **Retention**: Letzte 7 Backups
- **Methode**: SQLite Online-Backup (ohne Downtime)

```bash
# Manuelles Backup
sudo /root/backup-xelanote.sh

# Backups anzeigen
sudo ls -la <BACKUP_DIR>/
```

#### 7. Health-Monitoring

**Health-Check-Skript**: `/root/healthcheck.sh`
- **Frequenz**: Alle 5 Minuten (Cronjob)
- **Aktion bei Failure**: Auto-Restart des Containers
- **Log**: `/var/log/xelanote-health.log`

```bash
# Manueller Check
sudo /root/healthcheck.sh

# Log anzeigen
sudo tail -20 /var/log/xelanote-health.log
```

---

### Quick Deploy auf Hetzner

```bash
# 1. Push (Hetzner pullt von forgejo)
git push forgejo main

# 2. Server: Pull & Build
ssh <PROD_SSH_ALIAS> "cd ~/xelanote && git pull && sudo docker build -t xelanote:latest ."

# 3. Container neu starten
ssh <PROD_SSH_ALIAS> "sudo docker stop xelanote && sudo docker rm xelanote"
ssh <PROD_SSH_ALIAS> 'sudo docker run -d --name xelanote --restart unless-stopped \
  -p 8080:8080 \
  -v ~/xelanote-data:/app/data \
  --memory=512m \
  --cpus=1 \
  --security-opt no-new-privileges \
  --pids-limit=200 \
  --env-file ~/.xelanote.env \
  xelanote:latest'

# 4. Verify
curl https://xelanote.com/health
```

---

### HTTPS mit Caddy (vorbereitet)

Caddy ist installiert und wartet auf Domain-Aktivierung.

**Status prüfen:**
```bash
ssh <PROD_SSH_ALIAS> "sudo systemctl status caddy"
```

**Caddyfile** (`/etc/caddy/Caddyfile`):
```
# Temporär: HTTP
:80 {
    reverse_proxy localhost:8080
}

# Nach Domain-Aktivierung einkommentieren:
# xelanote.com, www.xelanote.com {
#     reverse_proxy localhost:8080
#     encode zstd gzip
#     header {
#         Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
#         X-Content-Type-Options "nosniff"
#         X-Frame-Options "DENY"
#         -Server
#     }
# }
```

#### Domain-Aktivierung (wenn bereit)

1. **Cloudflare DNS konfigurieren:**
   - A-Record: `@` → `<PRODUCTION_IP>` (Proxied via Cloudflare)
   - CNAME: `www` → `xelanote.com` (Proxied)
   - SSL/TLS Mode: "Full (strict)"

2. **Caddyfile aktivieren:**
   ```bash
   ssh <PROD_SSH_ALIAS>
   sudo nano /etc/caddy/Caddyfile
   # Domain-Block einkommentieren, :80 Block auskommentieren
   sudo systemctl reload caddy
   ```

3. **Container auf localhost umstellen:**
   ```bash
   sudo docker stop xelanote && sudo docker rm xelanote
   sudo docker run -d --name xelanote --restart unless-stopped \
     -p 127.0.0.1:8080:8080 \
     -v ~/xelanote-data:/app/data \
     --log-driver json-file \
     --log-opt max-size=10m \
     --log-opt max-file=30 \
     --memory=512m --cpus=1 --security-opt no-new-privileges --pids-limit=200 \
     --env-file ~/.xelanote.env \
     xelanote:latest
   ```

4. **Hetzner Firewall:** Port 8080 entfernen (nur noch 80/443 nötig)

---

### Performance-Benchmark (2026-01-19)

| Metrik | Ergebnis |
|--------|----------|
| Health-Endpoint (10 concurrent) | 1,722 req/s |
| Health-Endpoint (50 concurrent) | 2,295 req/s |
| API /notes (10 concurrent) | 2,078 req/s |
| Durchschnittliche Latenz | 4-22 ms |
| Fehlerrate | 0% |
| RAM-Nutzung unter Last | 7.3 MB / 512 MB |

**Fazit:** Server ist für xelanote deutlich überdimensioniert.

---

### Wartungsaufgaben

#### Kernel-Update (pending)
```bash
ssh <PROD_SSH_ALIAS> "sudo reboot"
# Warten, dann reconnect
ssh <PROD_SSH_ALIAS> "uname -r"
```

#### Logs rotieren
Automatisch via Docker und logrotate.

#### Backups prüfen
```bash
ssh <PROD_SSH_ALIAS> "sudo ls -la <BACKUP_DIR>/"
```

#### Fail2ban-Bans prüfen
```bash
ssh <PROD_SSH_ALIAS> "sudo fail2ban-client status sshd"
```
