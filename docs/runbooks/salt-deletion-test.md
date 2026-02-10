# Salt-Deletion Simulation Test

**Purpose:** Verify that the Encryption Salt Overwrite Prevention Fix (389f7b2) works correctly.

**WARNING:** After step 3, login is NOT possible until the salt is restored!

## Prerequisites

- Database backup created
- Salt value documented for recovery

## Test Steps

```bash
# 1. Check that encrypted notes exist
ssh SERVER 'docker exec xelanote sqlite3 /app/data/xelanote.db "SELECT COUNT(*) FROM notes WHERE user_id = 1 AND encrypted_content IS NOT NULL AND length(encrypted_content) > 0;"'
# Expected: > 0 notes

# 2. Save current salt
ssh SERVER 'docker exec xelanote sqlite3 /app/data/xelanote.db "SELECT hex(encryption_salt) FROM users WHERE id = 1;"'
# Expected: 32 hex characters (e.g. 2B92BF65913CEDF189896FFA2D0338AC)
# IMPORTANT: Copy and securely store this value!

# 3. Simulate salt loss (data corruption)
ssh SERVER 'docker exec xelanote sqlite3 /app/data/xelanote.db "UPDATE users SET encryption_salt = NULL WHERE id = 1;"'

# 4. Verify salt is NULL
ssh SERVER 'docker exec xelanote sqlite3 /app/data/xelanote.db "SELECT encryption_salt FROM users WHERE id = 1;"'
# Expected: No output (NULL)

# 5. Logout completely (clear session cookies)
# Browser: Application -> Cookies -> delete all xelanote cookies
# OR: Use incognito window

# 6. Attempt login
# Go to: https://SERVER-URL/login
# Try to log in
#
# EXPECTED RESULT:
# - Login fails
# - Error message: "encryption salt missing but encrypted notes exist - contact administrator for data recovery"
#
# ERROR IF:
# - Login succeeds -> Fix does NOT work
# - New salt generated -> DATA LOSS POSSIBLE

# 7. Check server logs
ssh SERVER "docker logs xelanote 2>&1 | grep -A2 'CRITICAL.*salt'"
# Expected:
# {"level":"ERROR","msg":"CRITICAL: User has encrypted notes but encryption salt is missing - REFUSING to generate new salt to prevent data loss","user_id":1}

# 8. Restore salt (with value from step 2)
ssh SERVER "docker exec xelanote sqlite3 /app/data/xelanote.db \"UPDATE users SET encryption_salt = x'2B92BF65913CEDF189896FFA2D0338AC' WHERE id = 1;\""
# Replace '2B92BF65913CEDF189896FFA2D0338AC' with your saved value!

# 9. Verify recovery
ssh SERVER 'docker exec xelanote sqlite3 /app/data/xelanote.db "SELECT hex(encryption_salt) FROM users WHERE id = 1;"'
# Expected: Your original salt

# 10. Retry login
# Go to: https://SERVER-URL/login
# Login should now work
# Notes should be readable
```

## Test Results

### Homelab (2026-01-23)

- Salt successfully set to NULL
- Salt successfully restored
- Login test not fully completed (active session cookie prevented getOrGenerateUserSalt() call)
- For complete test: Fully logout (delete all cookies) before login attempt

### Hetzner (2026-01-25)

- Deployment successful
- Salt Overwrite Prevention Fix verified

## References

- SALT_BUG_FIX.md - Full documentation
- Commit 389f7b2 - Fix implementation
- backend/internal/api/auth.go:167-204 - getOrGenerateUserSalt() with check
- backend/internal/service/notes.go:971-989 - UserHasEncryptedNotes()
