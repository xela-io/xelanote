# Deployment Security Guide

Anleitung zur sicheren Konfiguration von XelaNote-Servern.

## Checkliste neue Server

### 1. SSH absichern
```bash
# Port ändern und Root-Login deaktivieren
sudo nano /etc/ssh/sshd_config
# Port <SSH_PORT>
# PermitRootLogin no
# PasswordAuthentication no

sudo systemctl restart sshd
```

### 2. Firewall (UFW)
```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow <SSH_PORT>/tcp comment 'SSH'
sudo ufw allow 80/tcp comment 'HTTP/Caddy'
sudo ufw allow 443/tcp comment 'HTTPS/Caddy'
sudo ufw enable

# NICHT Port 8080 öffnen - Container nur auf localhost!
```

### 3. Fail2ban
```bash
sudo apt install fail2ban
sudo systemctl enable fail2ban
```

### 4. Env-Datei für Secrets erstellen
```bash
# Secrets NIEMALS in Git oder Command-Line!
cat > ~/.xelanote.env << 'EOF'
JWT_SECRET=<64-zeichen-hex>
XELANOTE_API_KEY_SECRET=<64-zeichen-hex-anders-als-jwt-secret>
XELANOTE_DB=/app/data/xelanote.db
XELANOTE_ENV=production
CORS_ALLOWED_ORIGINS=https://deine-domain.com
TURNSTILE_SECRET_KEY=<turnstile-secret>
TURNSTILE_SITE_KEY=<turnstile-site>
EOF

chmod 600 ~/.xelanote.env
```

**Neuen JWT_SECRET generieren:**
```bash
openssl rand -hex 32
```

### 5. Caddy installieren (Reverse Proxy)
```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install caddy
```

**Caddyfile:**
```bash
sudo tee /etc/caddy/Caddyfile << 'EOF'
deine-domain.com, www.deine-domain.com {
    reverse_proxy 127.0.0.1:8080 {
        header_up Host {host}
        header_up X-Real-IP {remote_host}
    }
}
EOF

sudo systemctl reload caddy
```

### 6. Docker Container starten (mit GDPR-konformer Log-Rotation)
```bash
sudo docker run -d --name xelanote --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v ~/xelanote-data:/app/data \
  --log-driver json-file \
  --log-opt max-size=10m \
  --log-opt max-file=30 \
  --memory=512m \
  --cpus=1 \
  --security-opt no-new-privileges \
  --pids-limit=200 \
  --env-file ~/.xelanote.env \
  xelanote:latest
```

**Wichtig:**
- `-p 127.0.0.1:8080:8080` bindet NUR auf localhost!
- `--log-driver json-file` aktiviert strukturiertes Logging
- `--log-opt max-file=30` = 30 Tage Log-Retention (GDPR-konform)

**Security Logging:**
XelaNote loggt folgende Security-Events:
- Login Success/Failure (identifier, IP)
- Account Lockouts (identifier, attempts, duration)
- Password/Email Changes (user_id, IP)
- 2FA Events (user_id, IP)

Siehe [SECURITY.md](../SECURITY.md) für Details.

### 7. Cloudflare einrichten
1. Domain hinzufügen (Free Plan reicht)
2. DNS A-Record: `@` → Server-IP (Proxied ☁️)
3. DNS CNAME: `www` → `deine-domain.com` (Proxied)
4. SSL/TLS → **"Full (strict)"**

## Secrets rotieren

### JWT_SECRET ändern (invalidiert alle Sessions!)
```bash
# 1. Neuen Secret generieren
NEW_SECRET=$(openssl rand -hex 32)

# 2. Env-Datei aktualisieren
sed -i "s/JWT_SECRET=.*/JWT_SECRET=$NEW_SECRET/" ~/.xelanote.env

# 3. Container neu starten
sudo docker restart xelanote
```

### Turnstile Keys ändern
1. Neuen Key in Cloudflare Dashboard erstellen
2. `~/.xelanote.env` aktualisieren
3. `sudo docker restart xelanote`

## Troubleshooting

### Container-Logs prüfen
```bash
# Letzte 50 Zeilen
sudo docker logs xelanote --tail 50

# Alle Logs (inkl. Rotation)
sudo docker logs xelanote

# Nur Login-Events
sudo docker logs xelanote 2>&1 | grep "login_"

# Account Lockouts
sudo docker logs xelanote 2>&1 | grep "account_locked"

# 2FA Events
sudo docker logs xelanote 2>&1 | grep "2fa_"

# Logs in Echtzeit verfolgen
sudo docker logs xelanote -f
```

### Health-Check
```bash
# Lokal auf Server
curl http://localhost:8080/health

# Extern via Domain
curl https://deine-domain.com/health
```

### Firewall-Status
```bash
sudo ufw status verbose
```

### Caddy-Status
```bash
sudo systemctl status caddy
sudo journalctl -u caddy --since "1 hour ago"
```

## Backup

### Manuelles Backup
```bash
sqlite3 ~/xelanote-data/xelanote.db ".backup ~/backup-$(date +%Y%m%d).db"
```

### Automatisches Backup (Cron)
```bash
# Täglich 3:00 UTC
echo "0 3 * * * sqlite3 <DEPLOY_DATA_DIR>/xelanote.db \".backup <BACKUP_DIR>/xelanote-\$(date +\%Y\%m\%d).db\"" | sudo crontab -
```

## Security Monitoring

### Log-Analyse-Befehle

**Login Success Rate:**
```bash
TOTAL=$(sudo docker logs xelanote 2>&1 | grep -c "login_")
SUCCESS=$(sudo docker logs xelanote 2>&1 | grep -c "login_success")
echo "Success Rate: $(echo "scale=2; $SUCCESS * 100 / $TOTAL" | bc)%"
```

**Account Lockouts (letzte Stunde):**
```bash
sudo docker logs xelanote 2>&1 | grep "account_locked" | grep "$(date -u +%Y-%m-%dT%H)" | wc -l
```

**Top Failed Login Attempts:**
```bash
sudo docker logs xelanote 2>&1 | grep "login_failed" | \
  grep -oP 'identifier="[^"]*"' | sort | uniq -c | sort -rn | head -10
```

**2FA Deactivations:**
```bash
sudo docker logs xelanote 2>&1 | grep "2fa_disabled" | tail -20
```

### Alert Thresholds (Empfohlen)

**Kritisch (sofort reagieren):**
- Login Failures > 100/Stunde für single identifier → Brute-Force
- Account Lockouts > 10/Stunde → Distributed Attack
- 2FA Deactivations > 5/Stunde → Account Compromise

**Warning (überwachen):**
- Password Changes > 50/Stunde → Breach oder Reset-Attack
- Login Success Rate < 50% → Potenzielle Attacke

### Einfaches Monitoring-Script

```bash
#!/bin/bash
# /root/security-monitor.sh

# Check Login Failures (last hour)
FAILURES=$(docker logs xelanote 2>&1 | grep "login_failed" | grep "$(date -u +%Y-%m-%dT%H)" | wc -l)
if [ "$FAILURES" -gt 100 ]; then
  echo "ALERT: $FAILURES login failures in last hour!" | mail -s "XelaNote Security Alert" admin@example.com
fi

# Check Account Lockouts
LOCKOUTS=$(docker logs xelanote 2>&1 | grep "account_locked" | grep "$(date -u +%Y-%m-%dT%H)" | wc -l)
if [ "$LOCKOUTS" -gt 10 ]; then
  echo "ALERT: $LOCKOUTS account lockouts in last hour!" | mail -s "XelaNote Security Alert" admin@example.com
fi
```

**Cronjob (stündlich):**
```bash
echo "0 * * * * /root/security-monitor.sh" | sudo crontab -
```

### GDPR-Hinweise

- **Log-Retention:** 30 Tage (automatisch via `--log-opt max-file=30`)
- **Personenbezogene Daten:** user_id, identifier (username/email), IP-Adressen
- **Rechtsgrundlage:** GDPR Art. 6(1)(f) - Berechtigtes Interesse (Security Monitoring)
- **User-Rechte:** Account-Deletion löscht DB-Daten sofort, Logs bleiben bis Rotation (max 30 Tage)

**Privacy-Policy Disclosure erforderlich!**
Siehe [SECURITY.md](../SECURITY.md#gdpr-compliance--log-retention) für Template-Text.
