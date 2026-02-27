# Forgejo Actions Runner Setup

Dokumentation der selbst-gehosteten Forgejo Actions Runner auf Staging und Production.

## Architektur

```
Developer                    Forgejo (<FORGEJO_URL>)
   |                                |
   |-- git push forgejo main ------>|---> Staging-Runner (<STAGING_IP>)
   |                                |     runs-on: staging
   |                                |     Trigger: push auf main
   |                                |
   |-- git tag v1.2.3 ------------->|---> Production-Runner (Hetzner, <PRODUCTION_IP>)
   |-- git push forgejo v1.2.3 ---->|     runs-on: production
   |                                |     Trigger: Tag v* oder manuell
   |                                |
   |   (oder: manueller Trigger     |
   |    via Forgejo UI)             |
```

**Deploy-Flow:**
1. Code pushen → Staging deployed automatisch → testen
2. Wenn OK: `git tag v1.2.3 && git push forgejo v1.2.3` → Production deployed automatisch
3. Hotfix: manueller Trigger via Forgejo UI (workflow_dispatch)

**Beide Runner** laufen direkt auf dem jeweiligen Host (`:host` Label), nicht in Docker-Containern. Sie bauen und verwalten die xelanote Docker-Container.

---

## Umgebungen

| | Staging | Production |
|---|---------|------------|
| **Server** | <STAGING_IP> | <PRODUCTION_IP> (Hetzner CX22) |
| **URL** | https://<STAGING_URL> | https://xelanote.com |
| **Runner-Name** | staging-runner | production-runner |
| **Runner-Label** | `staging:host` | `production:host` |
| **Workflow** | `deploy-staging.yml` | `deploy-production.yml` |
| **Trigger** | Push auf `main` | Tag `v*` oder manuell |
| **Env-File** | `<STAGING_ENV_FILE>` | `<PROD_ENV_FILE>` |
| **Port** | 127.0.0.1:8081 → 8080 | 127.0.0.1:8080 → 8080 |
| **Reverse Proxy** | nginx-proxy-manager | Caddy |
| **Docker-Netzwerk** | `nginx_default` | keins (Caddy auf Host) |
| **Volume** | Docker Volume `xelanote_xelanote-data` | Bind Mount `<PROD_HOME>/xelanote-data` |
| **Image Retention** | 3 Images | 5 Images |
| **Pre-Deploy Backup** | nein | ja (automatisch) |

### Gemeinsame Konfiguration

| Eigenschaft | Wert |
|-------------|------|
| **Runner-Version** | v6.3.1 |
| **Binary** | `/usr/local/bin/forgejo-runner` |
| **System-User** | `forgejo-runner` |
| **Home-Verzeichnis** | `/var/lib/forgejo-runner` |
| **Systemd-Service** | `forgejo-runner.service` |
| **Forgejo-Instanz** | https://<FORGEJO_URL> |
| **Setup-Script** | `scripts/setup-forgejo-runner.sh` (fuer Staging) |

---

## Voraussetzungen

Auf beiden Servern muessen installiert sein:

| Abhaengigkeit | Grund |
|---------------|-------|
| **Docker** | Container-Build und -Verwaltung |
| **Git** | Repository-Checkout |
| **curl** | Health Checks, Download |
| **Node.js** | Benoetigt von `actions/checkout` (SHA-pinned v4) |
| **acl** (setfacl) | POSIX ACL fuer Env-File-Zugriff |

```bash
# Debian/Ubuntu
sudo apt install docker.io git curl nodejs acl
```

Zusaetzlich umgebungsspezifisch:

**Staging:**
- Docker-Netzwerk `nginx_default` (fuer nginx-proxy-manager)
- Env-File `<STAGING_ENV_FILE>` (chmod 600)

**Production:**
- Env-File `<PROD_ENV_FILE>` (chmod 600)
- Daten-Verzeichnis `<DEPLOY_DATA_DIR>/`

---

## Installation

### Automatisch (Staging)

Das Setup-Script `scripts/setup-forgejo-runner.sh` automatisiert die Installation:

```bash
sudo bash scripts/setup-forgejo-runner.sh
```

Das Script fuehrt aus:
1. Pre-Flight Checks (git, curl, docker, env-file)
2. Architektur erkennen (amd64/arm64)
3. Binary herunterladen (v6.3.1)
4. System-User `forgejo-runner` erstellen
5. User zur Docker-Gruppe hinzufuegen
6. POSIX ACL auf Env-File setzen
7. Systemd-Service erstellen und aktivieren

### Manuell (Production / beliebiger Server)

```bash
# 1. Binary herunterladen
ARCH="amd64"  # oder arm64
sudo curl -fsSL "https://code.forgejo.org/forgejo/runner/releases/download/v6.3.1/forgejo-runner-6.3.1-linux-${ARCH}" \
  -o /usr/local/bin/forgejo-runner
sudo chmod 755 /usr/local/bin/forgejo-runner

# 2. System-User erstellen
sudo useradd --system --shell /usr/sbin/nologin \
  --home-dir /var/lib/forgejo-runner --create-home forgejo-runner

# 3. Docker-Gruppe
sudo usermod -aG docker forgejo-runner

# 4. ACL auf Env-File (Pfade anpassen!)
# Staging:
sudo setfacl -m "u:forgejo-runner:r" <STAGING_ENV_FILE>
sudo setfacl -m "u:forgejo-runner:x" <STAGING_HOME>

# Production:
sudo setfacl -m "u:forgejo-runner:r" <PROD_ENV_FILE>
sudo setfacl -m "u:forgejo-runner:x" <PROD_HOME>

# 5. Systemd-Service erstellen (siehe Abschnitt weiter unten)
```

---

## Runner registrieren

### 1. Token aus Forgejo holen

1. Repository oeffnen: https://<FORGEJO_URL>/xela/xelanote
2. **Settings** > **Actions** > **Runners**
3. **Create new runner** klicken
4. Token kopieren

### 2. Registrierung ausfuehren

**Wichtig:** Muss als `forgejo-runner` User im richtigen Verzeichnis ausgefuehrt werden:

```bash
# Staging
sudo -u forgejo-runner bash -c "cd /var/lib/forgejo-runner && forgejo-runner register \
  --instance https://<FORGEJO_URL> \
  --token <TOKEN> \
  --name staging-runner \
  --labels staging:host \
  --no-interactive"

# Production
sudo -u forgejo-runner bash -c "cd /var/lib/forgejo-runner && forgejo-runner register \
  --instance https://<FORGEJO_URL> \
  --token <TOKEN> \
  --name production-runner \
  --labels production:host \
  --no-interactive"
```

**Label erklaert:**
- `staging` / `production` = Name, unter dem der Runner im Workflow referenziert wird (`runs-on: staging` / `runs-on: production`)
- `host` = Ausfuehrungsmodus: direkt auf dem Host (nicht in Docker)

### 3. Runner starten

```bash
sudo systemctl start forgejo-runner
sudo systemctl status forgejo-runner
```

### 4. Actions im Repository aktivieren

Falls noch nicht geschehen:

1. **Repository Settings** > **Features** > **Actions** Checkbox aktivieren
2. In der Forgejo `app.ini`:
   ```ini
   [actions]
   ENABLED = true
   DEFAULT_ACTIONS_URL = github
   ```

---

## Systemd-Service

### Unit-Datei

Pfad: `/etc/systemd/system/forgejo-runner.service` (identisch auf beiden Servern)

```ini
[Unit]
Description=Forgejo Actions Runner
Documentation=https://forgejo.org/docs/latest/admin/actions/
After=docker.service
Requires=docker.service

[Service]
Type=simple
User=forgejo-runner
Group=forgejo-runner
WorkingDirectory=/var/lib/forgejo-runner
ExecStart=/usr/local/bin/forgejo-runner daemon
Restart=on-failure
RestartSec=10

# Security hardening
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=/var/lib/forgejo-runner
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

### Security-Hardening

| Option | Wirkung |
|--------|---------|
| `ProtectSystem=strict` | Root-Filesystem read-only (ausser explizite Pfade) |
| `ProtectHome=read-only` | `/home` nur lesbar (fuer Env-File-Zugriff) |
| `ReadWritePaths=/var/lib/forgejo-runner` | Einziger beschreibbarer Pfad |
| `NoNewPrivileges=true` | Keine Privilege Escalation moeglich |
| `PrivateTmp=true` | Isoliertes /tmp |

### Wichtige Befehle

```bash
sudo systemctl status forgejo-runner      # Status
sudo journalctl -u forgejo-runner -f      # Logs (live)
sudo systemctl restart forgejo-runner     # Neustart
sudo systemctl stop forgejo-runner        # Stoppen
```

---

## Workflows

### Staging: `deploy-staging.yml`

**Trigger:** Push auf `main` Branch

**Pipeline:**
1. Checkout (SHA-pinned `actions/checkout@v4`)
2. Pre-flight checks (Env-File, Pflicht-Variablen, JWT-Laenge, Docker, Netzwerk)
3. Docker Build mit SHA-Tag (12-stellig) + `latest`
4. Stop/Remove alter Container (30s Grace Period)
5. Start neuer Container
6. Health Check (30 Versuche, 2s Intervall = max 60s)
7. Auto-Rollback bei Health-Check-Failure
8. Image Cleanup (3 neueste behalten)
9. Deployment Summary

### Production: `deploy-production.yml`

**Trigger:** Tag `v*` (z.B. `v1.2.3`) oder manueller Trigger (workflow_dispatch)

**Zusaetzlich zu Staging:**
- **Pre-Deploy Backup**: Automatisches Datenbank-Backup vor jedem Deploy (5 Backups Retention)
- **XELANOTE_ENV Check**: Prueft ob Env-File `XELANOTE_ENV=production` enthaelt
- **Version-Tag im Image**: Bei Tags wird der Tag-Name als Image-Tag verwendet (`xelanote:v1.2.3`), bei manuellem Trigger der Commit-SHA
- **Image Retention**: 5 statt 3 Images (konservativer)
- **Deploy-Grund im Summary**: Bei manuellem Trigger wird der Grund dokumentiert

**Verwendung:**

```bash
# Normaler Release
git tag v1.2.3
git push forgejo v1.2.3

# Hotfix (manuell in Forgejo UI)
# -> Repository -> Actions -> deploy-production -> Run workflow
# -> Reason eingeben (z.B. "hotfix: fix login bug")
```

### Container Security-Hardening (beide Umgebungen)

```
--read-only                    # Read-only Root-Filesystem
--tmpfs /tmp                   # Schreibbarer tmp-Bereich
--cap-drop ALL                 # Alle Linux Capabilities entfernt
--security-opt no-new-privileges
--memory=512m                  # RAM-Limit
--cpus=1                       # CPU-Limit
--pids-limit=200               # Fork-Bomb-Schutz
--log-driver json-file         # Strukturierte Logs
--log-opt max-size=10m         # Max 10MB pro Log-Datei
--log-opt max-file=3           # 3 Dateien rotierend
```

### Netzwerk-Konfiguration

**Staging:**
```
Container:8080 --> Host:127.0.0.1:8081 --> nginx-proxy-manager --> https://<STAGING_URL>
                                           (Docker-Netzwerk: nginx_default)
```

**Production:**
```
Container:8080 --> Host:127.0.0.1:8080 --> Caddy --> https://xelanote.com
                                           (laeuft auf Host, kein Docker-Netzwerk)
```

### Auto-Rollback (beide Umgebungen)

Falls der Health Check fehlschlaegt:
1. Neuer Container wird gestoppt und entfernt
2. Vorheriges Image wird gestartet (mit denselben Flags)
3. Health Check auf Rollback-Container
4. Deployment wird als **failed** markiert

Beim allerersten Deploy gibt es kein Rollback-Target.

---

## Env-Files

### Staging: `<STAGING_ENV_FILE>`

Owner: `<STAGING_USER>:<STAGING_USER>`, chmod 600. ACL: `u:forgejo-runner:r`.

### Production: `<PROD_ENV_FILE>`

Owner: `<PROD_USER>:<PROD_USER>`, chmod 600. ACL: `u:forgejo-runner:r`.

### Pflicht-Variablen

| Variable | Staging | Production |
|----------|---------|------------|
| `JWT_SECRET` | 64+ hex Zeichen | 64+ hex Zeichen |
| `XELANOTE_DB` | `/app/data/xelanote.db` | `/app/data/xelanote.db` |
| `XELANOTE_ENV` | `production` | `production` |
| `CORS_ALLOWED_ORIGINS` | `https://<STAGING_URL>` | `https://xelanote.com` |
| `WEBAUTHN_RP_ID` | `<STAGING_URL>` | `xelanote.com` |

### Optionale Variablen

| Variable | Beschreibung |
|----------|-------------|
| `TURNSTILE_SECRET_KEY` | Cloudflare Turnstile CAPTCHA Secret |
| `TURNSTILE_SITE_KEY` | Cloudflare Turnstile Site Key |
| `XELANOTE_DB_KEY` | SQLCipher Encryption Key |
| `XELANOTE_DB_KEY_FILE` | Pfad zu SQLCipher Key File |

### Zugriffsrechte setzen

```bash
# ACL auf Datei + Parent-Directory
sudo setfacl -m "u:forgejo-runner:r" <ENV_FILE>
sudo setfacl -m "u:forgejo-runner:x" <PARENT_DIR>

# Pruefen
getfacl <ENV_FILE>
# user:forgejo-runner:r--
```

---

## Troubleshooting

### Runner startet nicht / "unregistered runner"

```bash
# Logs pruefen
sudo journalctl -u forgejo-runner -n 50

# Haeufigste Ursache: .runner-Datei fehlt oder veraltet
sudo ls -la /var/lib/forgejo-runner/.runner

# Fix: Alte Config loeschen, neu registrieren
sudo systemctl stop forgejo-runner
sudo rm /var/lib/forgejo-runner/.runner
# Neues Token aus Forgejo UI holen, dann:
sudo -u forgejo-runner bash -c "cd /var/lib/forgejo-runner && forgejo-runner register ..."
sudo systemctl start forgejo-runner
```

### Checkout schlaegt fehl: "node: not found"

`actions/checkout@v4` benoetigt Node.js auf dem Host.

```bash
sudo apt install nodejs   # Debian/Ubuntu
```

### Health Check schlaegt fehl

```bash
# Staging
curl -v http://localhost:8081/health

# Production
curl -v http://localhost:8080/health

# Container-Logs
docker logs xelanote --tail 50

# Laeuft der Container?
docker ps -a | grep xelanote
```

### Env-File nicht lesbar

```bash
# Als Runner-User testen
sudo -u forgejo-runner cat <ENV_FILE_PATH>

# Fix: ACL auf Datei UND Parent-Directory
sudo setfacl -m "u:forgejo-runner:x" <PARENT_DIR>
sudo setfacl -m "u:forgejo-runner:r" <ENV_FILE>
```

### Docker-Netzwerk fehlt (nur Staging)

```bash
docker network create nginx_default
# Oder: nginx-proxy-manager starten (erstellt das Netzwerk automatisch)
```

### Env-File-Aenderungen greifen nicht

`docker restart` liest `--env-file` NICHT neu ein.

```bash
# FALSCH:
docker restart xelanote

# RICHTIG: Container komplett neu erstellen
docker stop xelanote && docker rm xelanote
# Neuer docker run mit --env-file (oder: naechsten Deploy abwarten)
```

---

## Wartung

### Runner-Version aktualisieren

```bash
sudo systemctl stop forgejo-runner

# Neue Binary herunterladen
sudo curl -fsSL "https://code.forgejo.org/forgejo/runner/releases/download/v<VERSION>/forgejo-runner-<VERSION>-linux-amd64" \
  -o /usr/local/bin/forgejo-runner
sudo chmod 755 /usr/local/bin/forgejo-runner

# Keine Neu-Registrierung noetig
sudo systemctl start forgejo-runner
```

### Runner-Cache bereinigen

```bash
sudo du -sh /var/lib/forgejo-runner/.cache
sudo find /var/lib/forgejo-runner/.cache -type d -mindepth 2 -maxdepth 2 -mtime +7 -exec rm -rf {} +
```

### Docker-Images bereinigen

```bash
docker images xelanote
docker image prune -f
docker builder prune --filter "until=24h" --keep-storage=2GB -f
```

### Logs pruefen

```bash
sudo journalctl -u forgejo-runner --since "1 hour ago"   # Runner
docker logs xelanote --tail 100                           # App
```

---

## Dateien-Referenz

| Datei | Beschreibung |
|-------|-------------|
| `scripts/setup-forgejo-runner.sh` | Installations-Script fuer Staging |
| `.forgejo/workflows/deploy-staging.yml` | Staging-Workflow (Push auf main) |
| `.forgejo/workflows/deploy-production.yml` | Production-Workflow (Tag v* oder manuell) |
| `/etc/systemd/system/forgejo-runner.service` | Systemd Unit (auf beiden Servern) |
| `/var/lib/forgejo-runner/.runner` | Runner-Config (nach Registrierung) |

---

## Lessons Learned

1. **Node.js ist Pflicht**: `actions/checkout` (auch die Forgejo-Variante) benoetigt Node.js auf dem Host.

2. **ACL auf Parent-Directory nicht vergessen**: `setfacl -m u:forgejo-runner:r` auf das Env-File reicht nicht - der User braucht auch Execute-Rechte (`x`) auf das Parent-Directory.

3. **`docker restart` != `docker run`**: Ein `docker restart` liest `--env-file` nicht neu ein. Bei Env-Aenderungen muss der Container komplett neu erstellt werden (stop/rm/run).

4. **`WEBAUTHN_RP_ID` in Production Mode**: Im `XELANOTE_ENV=production`-Modus muss `WEBAUTHN_RP_ID` gesetzt sein.

5. **SHA-pinned Checkout**: Die `actions/checkout` Action wird per SHA referenziert, um Supply-Chain-Angriffe zu verhindern. Bei Updates den SHA von <https://code.forgejo.org/actions/checkout> pruefen.

6. **Host-Runner statt Docker-Runner**: Die Runner laufen direkt auf dem Host, weil sie Docker-Container bauen und verwalten muessen.

7. **Registrierung im richtigen Verzeichnis**: `forgejo-runner register` muss in `/var/lib/forgejo-runner` ausgefuehrt werden (als `forgejo-runner` User), sonst scheitert das Schreiben der `.runner`-Datei.

8. **Tag-basiertes Production-Deploy**: Staging deployed auf jeden Push, Production nur auf Tags. Das gibt eine natuerliche Staging-Phase zum Testen.
