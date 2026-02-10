# Security Policy

## Reporting Security Vulnerabilities

If you discover a security vulnerability in XelaNote, please report it to:

**Email:** anor.londoe@pm.me

Please **do not** create public GitHub issues for security vulnerabilities.

---

## Security Features

### Authentication & Authorization

- **Password Hashing:** bcrypt with cost factor 12
- **JWT Tokens:** 15-minute access tokens, 7-day refresh tokens (HttpOnly cookies)
- **2FA Support:** TOTP-based two-factor authentication with backup codes
- **Account Lockout:** Exponential backoff after 5 failed login attempts
- **Password Recovery:** Secure recovery key mechanism with constant-time validation
- **Session Management:** Secure refresh token rotation with hash storage

### Timing Attack Prevention

- **Constant-Time Login:** Dummy bcrypt comparison for non-existent users (SEC-H01)
- **Generic Error Messages:** No user enumeration via registration errors (SEC-H02)
- **Timing-Safe Comparisons:** All authentication flows use constant-time operations

### Data Protection

- **E2E Encryption:** Client-side encryption for notes (AES-256-GCM)
- **Database Encryption:** Optional SQLCipher support for at-rest encryption
- **CORS Protection:** Strict origin validation in production
- **CSRF Protection:** Double-submit cookie pattern for state-changing operations
- **XSS Prevention:** Content Security Policy headers, input sanitization
- **ETag Hashing:** Version numbers obscured via SHA256 to prevent information disclosure (SEC-L02)

### Security Logging & Audit Trail

**All security-relevant events are logged with structured metadata for forensic analysis.**

#### Logged Security Events

| Event | Log Level | Metadata | Purpose |
|-------|-----------|----------|---------|
| `login_success` | INFO | identifier, remote_ip | Successful authentication tracking |
| `login_failed` | WARN | identifier, remote_ip, reason | Failed login attempts detection |
| `account_lockout` | WARN | identifier, failed_attempts, lockout_duration | Brute-force attack detection |
| `password_changed` | INFO | user_id, remote_ip | Password change audit |
| `email_changed` | INFO | user_id, new_email, remote_ip | Email change audit |
| `2fa_enabled` | INFO | user_id, method, remote_ip | 2FA activation tracking |
| `2fa_disabled` | WARN | user_id, remote_ip | 2FA deactivation tracking (security-sensitive) |
| `backup_codes_regenerated` | INFO | user_id, remote_ip | Backup code regeneration audit |

#### Log Format

All security events use structured logging (slog) with consistent fields:

```go
logger.Info("user_logged_in",
    slog.String("event", "login_success"),
    slog.String("identifier", "user@example.com"),
    slog.String("remote_ip", "1.2.3.4"))
```

**Metadata Fields:**
- `event` - Machine-readable event type (required)
- `user_id` - User ID for authenticated events (when available)
- `identifier` - Username/email for unauthenticated events
- `remote_ip` - Client IP address (from `X-Forwarded-For` or `X-Real-IP` headers, fallback to socket IP)
- `timestamp` - Automatically added by slog

---

## GDPR Compliance & Log Retention

### Personal Data in Logs

Security audit logs contain the following personal data:
- **User Identifiers:** user_id, username, email
- **Network Data:** IP addresses (remote_ip)
- **Behavioral Data:** Login attempts, password changes, 2FA events

### Log Retention Policy

**Retention Period:** 30 days

- Logs are automatically rotated and deleted after 30 days
- No manual deletion required (automatic via Docker log driver)
- Retention period balances security needs with privacy obligations

### Log Rotation Configuration

**Docker Logging (JSON File Driver):**

```yaml
# docker-compose.yml
services:
  xelanote:
    logging:
      driver: json-file
      options:
        max-size: "10m"     # Max size per log file
        max-file: "30"      # 30 files = ~30 days retention (1 file/day)
```

**Docker Run:**

```bash
docker run -d --name xelanote \
  --log-driver json-file \
  --log-opt max-size=10m \
  --log-opt max-file=30 \
  xelanote:latest
```

### User Rights (GDPR Article 17 - Right to Erasure)

**Account Deletion:**
- When a user deletes their account, all user data (notes, preferences, credentials) is immediately deleted from the database
- Security audit logs are **retained until automatic expiration** (max 30 days after account deletion)
- Rationale: Security logs are required for incident response and forensic analysis (GDPR Article 17(3)(e) - Legal compliance)

**Log Anonymization:**
- After account deletion, logs remain in their original form until automatic rotation
- The 30-day retention period is disclosed in the Privacy Policy
- Users are informed about log retention during account deletion

### Privacy Policy Disclosure

**Required Section in Privacy Policy:**

```markdown
## Security Audit Logs

We retain security audit logs for **30 days** for security monitoring and incident response purposes.

**Data Logged:**
- Login attempts (username/email, IP address, timestamp)
- Account lockouts (username/email, timestamp)
- Password and email changes (user ID, IP address, timestamp)
- Two-factor authentication events (user ID, IP address, timestamp)

**Retention:**
- Logs are automatically deleted after 30 days
- Upon account deletion, logs are retained until automatic expiration (maximum 30 days)

**Legal Basis:** GDPR Article 6(1)(f) - Legitimate interest in security monitoring
```

---

## Monitoring & Alerting

### Recommended Alert Thresholds

**Account Security:**
- Login failures > 100/hour for single identifier → Investigate potential brute-force attack
- Account lockouts > 10/hour → Possible distributed attack or credential stuffing
- 2FA deactivations > 5/hour → Investigate potential account compromise

**Password Changes:**
- Password changes > 50/hour → Possible breach or mass password reset attack

**Log Analysis Commands:**

```bash
# Count login failures in last hour
docker logs xelanote 2>&1 | grep "login_failed" | grep "$(date -u +%Y-%m-%dT%H)" | wc -l

# List recent account lockouts
docker logs xelanote 2>&1 | grep "account_locked" | tail -20

# Monitor 2FA deactivations
docker logs xelanote 2>&1 | grep "2fa_disabled"

# Check login success rate
TOTAL=$(docker logs xelanote 2>&1 | grep -c "login_")
SUCCESS=$(docker logs xelanote 2>&1 | grep -c "login_success")
echo "Success Rate: $(echo "scale=2; $SUCCESS * 100 / $TOTAL" | bc)%"
```

---

## Vulnerability Management

### Recent Security Fixes

**2026-01-28: Penetration Testing Quick Wins (SEC-H01, SEC-H02, SEC-M02, SEC-L02)**

- **SEC-H01 (HIGH):** Constant-time login to prevent timing attacks
  - Impact: User enumeration via timing analysis prevented
  - Timing difference: < 3ms (target: < 100ms)

- **SEC-H02 (HIGH):** Generic error messages to prevent user enumeration
  - Impact: Registration endpoint no longer leaks user existence
  - Error message: "unable to complete registration" (no specifics)

- **SEC-M02 (MEDIUM):** Security event logging for audit trail
  - Impact: Security incidents now detectable and traceable
  - 9 security events logged with structured metadata

- **SEC-L02 (LOW):** ETag hashing to prevent version disclosure
  - Impact: Note version numbers no longer exposed in ETags
  - Format: SHA256[:8] (16 hex chars) instead of raw integer

**Fixed Vulnerabilities:**

- **SEC-L04 (LOW):** Cookie `SameSite=Strict` + Signed URLs ✅ FIXED (2026-01-28)
  - Status: Implemented
  - Solution: Signed URLs (HMAC-SHA256) for uploads + SameSite=Strict cookies
  - Impact: Full CSRF protection without breaking image rendering
  - See: `docs/signed-urls.md` for technical details

### Security Hardening Checklist

**Production Deployment:**

- [x] JWT_SECRET ≥ 64 characters (enforced at startup)
- [x] CORS_ALLOWED_ORIGINS explicitly set (no wildcards)
- [x] HTTPS/TLS enabled (via Caddy/nginx)
- [x] Database backups configured (daily @ 3:00 UTC)
- [x] Log rotation enabled (30 days retention)
- [x] Security headers configured (CSP, HSTS, X-Frame-Options)
- [ ] Rate limiting enabled (optional, via nginx/Caddy)
- [ ] Intrusion detection system (optional, via fail2ban)
- [ ] Security monitoring dashboard (optional, via Grafana/Loki)

**Environment Variables (Production):**

```bash
# Required (Server will NOT start without these)
JWT_SECRET=<64+ char random string>
CORS_ALLOWED_ORIGINS=https://xelanote.com,https://www.xelanote.com

# Recommended
XELANOTE_ENV=production
XELANOTE_DB=/app/data/xelanote.db

# Optional (E2E Encryption)
XELANOTE_DB_KEY_FILE=/run/secrets/xelanote_db_key

# Optional (CAPTCHA - both required if enabled)
TURNSTILE_SECRET_KEY=<cloudflare_secret>
TURNSTILE_SITE_KEY=<cloudflare_site_key>
```

---

## Security Best Practices

### For Administrators

1. **JWT Secret:** Generate with `openssl rand -hex 32` (64 hex chars = 256 bits)
2. **Database Backups:** Test restore procedure regularly
3. **Log Monitoring:** Review security logs weekly for anomalies
4. **Dependency Updates:** Keep Go dependencies up-to-date (`go get -u`)
5. **TLS Certificates:** Use Let's Encrypt or similar for automatic renewal

### For Developers

1. **Input Validation:** Validate and sanitize all user input
2. **SQL Injection:** Use parameterized queries (never string concatenation)
3. **XSS Prevention:** Escape output, use Content Security Policy
4. **Secrets Management:** Never commit secrets to Git (use env files with chmod 600)
5. **Error Handling:** Log detailed errors server-side, return generic errors to clients

### For Users

1. **Strong Passwords:** Use a password manager, ≥12 characters
2. **Enable 2FA:** TOTP-based two-factor authentication strongly recommended
3. **Backup Codes:** Store backup codes securely (offline, encrypted)
4. **Recovery Key:** Generate and store recovery key in case of password loss
5. **Regular Backups:** Export notes regularly (especially if self-hosting)

---

## Contact

**Security Issues:** anor.londoe@pm.me
**General Support:** https://github.com/xela-io/xelanote/issues
**Documentation:** https://github.com/xela-io/xelanote/tree/main/docs

---

**Last Updated:** 2026-01-28
**Version:** 1.0
