# Deployment Rollback Procedure

## Overview

This document describes the rollback procedure for xelanote deployments in case of critical issues after a release. The procedure is designed to minimize downtime and restore functionality quickly.

### Automatic Rollback (Staging)

The Forgejo Actions CI/CD pipeline includes **automatic rollback** for staging deployments. If the health check fails after deploying a new version, the workflow automatically:

1. Stops and removes the failed container
2. Starts the previously running image
3. Verifies the rollback container is healthy
4. Reports the failure in the Forgejo Actions UI

This means most staging rollbacks require **no manual intervention**. See [deployment.md (CI/CD section)](deployment.md#cicd-forgejo-actions-auto-deploy-staging) for details.

The manual procedure below is needed for:
- **Hetzner Production** (no auto-deploy yet)
- **Staging** when the auto-rollback itself fails
- **Database rollback** scenarios (auto-rollback only handles the container, not the database)

## When to Rollback

Consider a rollback when:

- **Error rate > 10%**: Significant increase in application errors
- **Authentication failures**: Users unable to login or access the application
- **Upload failures**: File upload or serving functionality broken
- **Database migration errors**: Failed or incomplete migrations
- **Critical security issues**: New vulnerabilities introduced
- **Performance degradation**: Significant slowdown or resource exhaustion

## Pre-Rollback Checklist

Before executing a rollback:

1. **Verify the issue**: Confirm the problem is not environmental or temporary
2. **Check logs**: Review application logs to understand the failure
3. **Document the issue**: Note what failed, when, and any error messages
4. **Notify stakeholders**: Inform users if downtime is expected

## Rollback Procedure

### Step 1: Stop Current Container

```bash
# Homelab
ssh <STAGING_USER>@<STAGING_IP> "docker stop xelanote"

# Hetzner Production
ssh xelanote-prod "sudo docker stop xelanote"
```

### Step 2: Remove Failed Container

```bash
# Homelab
ssh <STAGING_USER>@<STAGING_IP> "docker rm xelanote"

# Hetzner Production
ssh xelanote-prod "sudo docker rm xelanote"
```

### Step 3: Restore Previous Image

**Option A: Use tagged previous version** (recommended)

```bash
# Homelab - pull specific tag
ssh <STAGING_USER>@<STAGING_IP> "docker pull xelanote:v1.2.3"

# Hetzner - pull specific tag
ssh xelanote-prod "sudo docker pull xelanote:v1.2.3"
```

**Option B: Rebuild from previous Git commit**

```bash
# Homelab
ssh <STAGING_USER>@<STAGING_IP> "cd ~/xelanote && git checkout <previous-commit-sha> && docker build -t xelanote:rollback ."

# Hetzner
ssh xelanote-prod "cd ~/xelanote && git checkout <previous-commit-sha> && sudo docker build -t xelanote:rollback ."
```

### Step 4: Restore Database (if migrations ran)

**IMPORTANT**: Only restore the database if the failed deployment ran migrations that changed the schema.

```bash
# Homelab - restore from backup
ssh <STAGING_USER>@<STAGING_IP> "cp /path/to/backup/xelanote-backup-YYYY-MM-DD.db ~/xelanote/data/xelanote.db"

# Hetzner - restore from daily backup
ssh xelanote-prod "sudo cp /root/backups/xelanote-backup-YYYY-MM-DD.db /home/deploy/xelanote-data/xelanote.db"
ssh xelanote-prod "sudo chown 100:101 /home/deploy/xelanote-data/xelanote.db"
```

### Step 5: Start Previous Version

**Homelab:**

```bash
ssh <STAGING_USER>@<STAGING_IP> 'docker run -d --name xelanote --restart unless-stopped \
  -p 8081:8080 --network nginx_default \
  -v xelanote_xelanote-data:/app/data \
  --env-file ~/.xelanote.env \
  xelanote:v1.2.3'
```

**Hetzner Production:**

```bash
ssh xelanote-prod 'sudo docker run -d --name xelanote --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v ~/xelanote-data:/app/data \
  --memory=512m \
  --cpus=1 \
  --security-opt no-new-privileges \
  --pids-limit=200 \
  --env-file ~/.xelanote.env \
  xelanote:v1.2.3'
```

### Step 6: Verify Rollback Success

Run verification checks:

```bash
# Check container is running
ssh <STAGING_USER>@<STAGING_IP> "docker ps | grep xelanote"
ssh xelanote-prod "sudo docker ps | grep xelanote"

# Check health endpoint
curl https://<STAGING_URL>/health
curl https://xelanote.com/health

# Check logs for errors
ssh <STAGING_USER>@<STAGING_IP> "docker logs --tail 50 xelanote"
ssh xelanote-prod "sudo docker logs --tail 50 xelanote"
```

**Verification Checklist:**

- [ ] Container is running
- [ ] Health endpoint returns 200 OK
- [ ] Login functionality works
- [ ] Notes can be created and viewed
- [ ] File uploads work (if applicable)
- [ ] No errors in logs
- [ ] Response times are normal

## Expected Downtime

- **Without database restore**: 2-3 minutes
- **With database restore**: 5-10 minutes (depending on DB size)

## Post-Rollback Actions

After successful rollback:

1. **Root cause analysis**: Investigate why the deployment failed
2. **Update documentation**: Document the issue and resolution
3. **Fix the issue**: Address the problem in development
4. **Test thoroughly**: Ensure the fix works before next deployment
5. **Plan re-deployment**: Schedule next deployment with additional safeguards

## Rollback Limitations

### Cannot Rollback If:

- **Database schema changed**: Forward-only migrations (data loss risk)
- **Data format changed**: Encrypted data format incompatible
- **External dependencies**: Third-party API changes not reversible

### Mitigation:

- Always backup before deployment
- Test migrations in staging first
- Use feature flags for gradual rollouts
- Monitor deployments closely for first 24 hours

## Emergency Contacts

If rollback fails or issues persist:

- **Developer**: Check GitHub issues
- **Logs location**:
  - Homelab: `docker logs xelanote`
  - Hetzner: `sudo docker logs xelanote`
- **Backup location**:
  - Homelab: Manual backups in `/path/to/backups/`
  - Hetzner: `/root/backups/` (daily at 3:00 UTC)

## Rollback History

Track rollbacks to identify patterns:

| Date | Server | Version | Reason | Downtime |
|------|--------|---------|--------|----------|
| - | - | - | - | - |

## Preventing Future Rollbacks

Best practices:

1. **Staging auto-deploy**: Every push to `forgejo main` is automatically deployed to staging with health checks and auto-rollback
2. **Gradual rollouts**: Deploy to Staging (auto) before Hetzner Production (manual)
3. **Automated tests**: Run full test suite before deployment
4. **Database backups**: Always backup before migrations
5. **Monitoring**: Set up alerts for error rates and performance
6. **Feature flags**: Use flags to enable/disable new features
7. **Canary deployments**: Roll out to small percentage of users first

## Related Documentation

- [Deployment Guide](deployment.md)
- [Security Checklist](deployment-security.md)
- [Database Migrations](../backend/internal/db/migrations/README.md)
