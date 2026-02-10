import { expect, test } from '@playwright/test';

// Helper function to register and login a new user (copied from notes.spec.ts)
async function registerAndLogin(page: import('@playwright/test').Page) {
  const username = `testuser${Date.now()}`;
  const email = `testuser-${Date.now()}@example.com`;
  const password = 'password123';

  // Register - use exact same pattern as notes.spec.ts
  await page.goto('/register', { waitUntil: 'networkidle' });
  await page.waitForSelector('input[name="username"]', { state: 'visible' });

  await page.fill('input[name="username"]', username);
  await page.fill('input[name="email"]', email);
  await page.fill('input[name="password"]', password);
  await page.fill('input[name="confirmPassword"]', password);

  // Wait for form to be interactive and submit
  await page.waitForTimeout(500);

  // Click the submit button directly
  await page.click('button[type="submit"]');

  // Wait for redirect to login page
  await page.waitForURL(/\/login$/, { timeout: 15000 });
  await expect(page).toHaveURL(/\/login$/);

  // Login
  await page.goto('/login');
  await page.fill('input[name="username_or_email"]', email);
  await page.fill('input[name="password"]', password);
  await page.click('button[type="submit"]');
  await expect(page).toHaveURL(/\/$/, { timeout: 10000 });

  // Wait for encryption to initialize (libsodium + KEK derivation)
  await page.waitForTimeout(2000);

  return { username, email, password };
}

test.describe('Folders API - Virtual Root (Migration 025)', () => {
  test('no hardcoded root folder with id=1 exists in API response', async ({ page }) => {
    await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    // Check API response for folders
    const foldersResponse = await page.evaluate(async () => {
      const response = await fetch('/api/folders', {
        credentials: 'include',
      });
      if (!response.ok) {
        return { error: response.status };
      }
      return response.json();
    });

    // Should return array (possibly empty for new user)
    expect(Array.isArray(foldersResponse)).toBe(true);

    // Verify no folder has path="/" or name="Root"
    const folders = foldersResponse as Array<{ path: string; name: string; id: number }>;
    for (const folder of folders) {
      expect(folder.path).not.toBe('/');
      expect(folder.name).not.toBe('Root');
    }
  });

  test('top-level folder created via API has parent_id=null', async ({ page }) => {
    await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    const folderName = `APITest${Date.now()}`;

    // Create folder via API
    const createResponse = await page.evaluate(async (name) => {
      const response = await fetch('/api/folders', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: `/${name}`, parent_path: '/' }),
      });
      if (!response.ok) {
        return { error: response.status, text: await response.text() };
      }
      return response.json();
    }, folderName);

    // Verify response is a folder object
    expect(createResponse).toHaveProperty('id');
    expect(createResponse).toHaveProperty('path');
    expect(createResponse).toHaveProperty('parent_id');

    // Verify parent_id is null (virtual root)
    const folder = createResponse as { parent_id: number | null; path: string; name: string };
    expect(folder.parent_id).toBeNull();
    expect(folder.path).toBe(`/${folderName}`);
    expect(folder.name).toBe(folderName);
  });

  test('nested folder has correct parent_id pointing to parent folder', async ({ page }) => {
    await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    const parentName = `Parent${Date.now()}`;
    const childName = `Child${Date.now()}`;

    // Create parent folder via API
    const parentResponse = await page.evaluate(async (name) => {
      const response = await fetch('/api/folders', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: `/${name}`, parent_path: '/' }),
      });
      return response.json();
    }, parentName);

    const parent = parentResponse as { id: number; parent_id: number | null };
    expect(parent.parent_id).toBeNull(); // Top-level = virtual root

    // Create child folder
    const childResponse = await page.evaluate(
      async ({ parentPath, childName }) => {
        const response = await fetch('/api/folders', {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            path: `${parentPath}/${childName}`,
            parent_path: parentPath,
          }),
        });
        return response.json();
      },
      { parentPath: `/${parentName}`, childName }
    );

    const child = childResponse as { id: number; parent_id: number | null; path: string };
    expect(child.parent_id).toBe(parent.id); // Should point to parent's ID
    expect(child.path).toBe(`/${parentName}/${childName}`);
  });

  test('moving folder to root level sets parent_id=null', async ({ page }) => {
    await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    const parentName = `MoveParent${Date.now()}`;
    const childName = `MoveChild${Date.now()}`;

    // Create parent folder
    const parentResponse = await page.evaluate(async (name) => {
      const response = await fetch('/api/folders', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: `/${name}`, parent_path: '/' }),
      });
      return response.json();
    }, parentName);

    const parent = parentResponse as { id: number };

    // Create nested folder
    const childResponse = await page.evaluate(
      async ({ parentPath, childName }) => {
        const response = await fetch('/api/folders', {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            path: `${parentPath}/${childName}`,
            parent_path: parentPath,
          }),
        });
        return response.json();
      },
      { parentPath: `/${parentName}`, childName }
    );

    const child = childResponse as { id: number; parent_id: number | null };
    expect(child.parent_id).toBe(parent.id); // Initially nested

    // Move child to root level
    const moveResponse = await page.evaluate(async (childId) => {
      const response = await fetch(`/api/folders/${childId}/move`, {
        method: 'PUT',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ new_parent_path: '/' }),
      });
      return response.json();
    }, child.id);

    const movedChild = moveResponse as { parent_id: number | null; path: string };
    expect(movedChild.parent_id).toBeNull(); // Now at root level
    expect(movedChild.path).toBe(`/${childName}`);
  });

  test('moving folder from root into another folder sets parent_id', async ({ page }) => {
    await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    const folder1Name = `Folder1_${Date.now()}`;
    const folder2Name = `Folder2_${Date.now()}`;

    // Create two top-level folders
    const folder1Response = await page.evaluate(async (name) => {
      const response = await fetch('/api/folders', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: `/${name}`, parent_path: '/' }),
      });
      return response.json();
    }, folder1Name);

    const folder2Response = await page.evaluate(async (name) => {
      const response = await fetch('/api/folders', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: `/${name}`, parent_path: '/' }),
      });
      return response.json();
    }, folder2Name);

    const folder1 = folder1Response as { id: number; parent_id: number | null };
    const folder2 = folder2Response as { id: number; parent_id: number | null };

    // Both should be at root level
    expect(folder1.parent_id).toBeNull();
    expect(folder2.parent_id).toBeNull();

    // Move folder2 into folder1
    const moveResponse = await page.evaluate(
      async ({ folderId, targetPath }) => {
        const response = await fetch(`/api/folders/${folderId}/move`, {
          method: 'PUT',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ new_parent_path: targetPath }),
        });
        return response.json();
      },
      { folderId: folder2.id, targetPath: `/${folder1Name}` }
    );

    const moved = moveResponse as { parent_id: number | null; path: string };
    expect(moved.parent_id).toBe(folder1.id);
    expect(moved.path).toBe(`/${folder1Name}/${folder2Name}`);
  });
});

test.describe('Folders API - CRUD Operations', () => {
  test('creates folder successfully', async ({ page }) => {
    await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    const folderName = `CreateTest${Date.now()}`;

    const response = await page.evaluate(async (name) => {
      const res = await fetch('/api/folders', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: `/${name}`, parent_path: '/' }),
      });
      return { status: res.status, data: await res.json() };
    }, folderName);

    expect(response.status).toBe(201);
    expect(response.data.name).toBe(folderName);
    expect(response.data.path).toBe(`/${folderName}`);
  });

  test('deletes folder successfully', async ({ page }) => {
    await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    const folderName = `DeleteTest${Date.now()}`;

    // Create folder
    const createResponse = await page.evaluate(async (name) => {
      const res = await fetch('/api/folders', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: `/${name}`, parent_path: '/' }),
      });
      return res.json();
    }, folderName);

    const folder = createResponse as { id: number };

    // Delete folder
    const deleteResponse = await page.evaluate(async (folderId) => {
      const res = await fetch(`/api/folders/${folderId}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      return { status: res.status };
    }, folder.id);

    expect(deleteResponse.status).toBe(204);

    // Verify folder is gone
    const listResponse = await page.evaluate(async () => {
      const res = await fetch('/api/folders', { credentials: 'include' });
      return res.json();
    });

    const folders = listResponse as Array<{ id: number }>;
    expect(folders.find((f) => f.id === folder.id)).toBeUndefined();
  });

  test('renames folder successfully', async ({ page }) => {
    await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    const oldName = `OldName${Date.now()}`;
    const newName = `NewName${Date.now()}`;

    // Create folder
    const createResponse = await page.evaluate(async (name) => {
      const res = await fetch('/api/folders', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: `/${name}`, parent_path: '/' }),
      });
      return res.json();
    }, oldName);

    const folder = createResponse as { id: number };

    // Rename folder
    const renameResponse = await page.evaluate(
      async ({ folderId, newName }) => {
        const res = await fetch(`/api/folders/${folderId}/rename`, {
          method: 'PUT',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ new_name: newName }),
        });
        return { status: res.status, data: await res.json() };
      },
      { folderId: folder.id, newName }
    );

    expect(renameResponse.status).toBe(200);
    expect(renameResponse.data.name).toBe(newName);
    expect(renameResponse.data.path).toBe(`/${newName}`);
  });

  test('sets and clears folder color', async ({ page }) => {
    await registerAndLogin(page);
    await page.waitForLoadState('networkidle');

    const folderName = `ColorTest${Date.now()}`;
    const color = '#FF5733';

    // Create folder
    const createResponse = await page.evaluate(async (name) => {
      const res = await fetch('/api/folders', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: `/${name}`, parent_path: '/' }),
      });
      return res.json();
    }, folderName);

    const folder = createResponse as { id: number; color: string | null };
    expect(folder.color).toBeUndefined(); // No color initially

    // Set color
    const setColorResponse = await page.evaluate(
      async ({ folderId, color }) => {
        const res = await fetch(`/api/folders/${folderId}/color`, {
          method: 'PUT',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ color }),
        });
        return { status: res.status, data: await res.json() };
      },
      { folderId: folder.id, color }
    );

    expect(setColorResponse.status).toBe(200);
    expect(setColorResponse.data.color).toBe(color);

    // Clear color
    const clearColorResponse = await page.evaluate(async (folderId) => {
      const res = await fetch(`/api/folders/${folderId}/color`, {
        method: 'PUT',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ color: null }),
      });
      return { status: res.status, data: await res.json() };
    }, folder.id);

    expect(clearColorResponse.status).toBe(200);
    expect(clearColorResponse.data.color).toBeNull();
  });
});

test.describe('Folders API - User Isolation', () => {
  test('users cannot see each other folders via API', async ({ browser }) => {
    // Create two separate browser contexts (two users)
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const page1 = await context1.newPage();
    const page2 = await context2.newPage();

    try {
      // Register and login user 1
      await registerAndLogin(page1);
      await page1.waitForLoadState('networkidle');

      // Register and login user 2
      await registerAndLogin(page2);
      await page2.waitForLoadState('networkidle');

      const folder1Name = `User1Folder${Date.now()}`;
      const folder2Name = `User2Folder${Date.now() + 1}`;

      // Create folder for user 1
      await page1.evaluate(async (name) => {
        await fetch('/api/folders', {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path: `/${name}`, parent_path: '/' }),
        });
      }, folder1Name);

      // Create folder for user 2
      await page2.evaluate(async (name) => {
        await fetch('/api/folders', {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path: `/${name}`, parent_path: '/' }),
        });
      }, folder2Name);

      // Get folders for user 1
      const user1Folders = (await page1.evaluate(async () => {
        const res = await fetch('/api/folders', { credentials: 'include' });
        return res.json();
      })) as Array<{ name: string }>;

      // Get folders for user 2
      const user2Folders = (await page2.evaluate(async () => {
        const res = await fetch('/api/folders', { credentials: 'include' });
        return res.json();
      })) as Array<{ name: string }>;

      // User 1 should only see their folder
      expect(user1Folders.map((f) => f.name)).toContain(folder1Name);
      expect(user1Folders.map((f) => f.name)).not.toContain(folder2Name);

      // User 2 should only see their folder
      expect(user2Folders.map((f) => f.name)).toContain(folder2Name);
      expect(user2Folders.map((f) => f.name)).not.toContain(folder1Name);
    } finally {
      await context1.close();
      await context2.close();
    }
  });
});
