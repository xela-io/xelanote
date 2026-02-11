import { expect, test, type Page } from '@playwright/test';
import { registerAndLogin } from './helpers/auth';

interface Folder {
  id: number;
  name: string;
  path: string;
  parent_id?: number | null;
  color?: string | null;
}

async function requestWithCsrf(
  page: Page,
  method: 'GET' | 'POST' | 'PUT' | 'DELETE',
  path: string,
  body?: Record<string, unknown>
): Promise<{ status: number; payload: unknown }> {
  return page.evaluate(
    async ({ method, path, body }) => {
      const headers: Record<string, string> = {};
      if (body) {
        headers['Content-Type'] = 'application/json';
      }
      const csrfMatch = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]+)/);
      if (csrfMatch?.[1]) {
        headers['X-CSRF-Token'] = decodeURIComponent(csrfMatch[1]);
      }

      const response = await fetch(path, {
        method,
        credentials: 'include',
        headers,
        body: body ? JSON.stringify(body) : undefined,
      });

      let payload: unknown = null;
      try {
        payload = await response.json();
      } catch {
        payload = null;
      }
      return { status: response.status, payload };
    },
    { method, path, body }
  );
}

async function listFolders(page: Page): Promise<Folder[]> {
  const response = await requestWithCsrf(page, 'GET', '/api/folders');
  expect(response.status).toBe(200);
  const json = response.payload as Folder[] | { folders?: Folder[] | null };
  if (Array.isArray(json)) {
    return json;
  }
  if (json && json.folders === null) {
    return [];
  }
  if (json && Array.isArray(json.folders)) {
    return json.folders;
  }
  throw new Error(`unexpected /api/folders payload: ${JSON.stringify(json)}`);
}

async function createFolder(page: Page, path: string): Promise<Folder> {
  const response = await requestWithCsrf(page, 'POST', '/api/folders', { path });
  expect(response.status).toBe(201);
  return response.payload as Folder;
}

async function moveFolder(page: Page, id: number, newParentPath: string): Promise<void> {
  const response = await requestWithCsrf(page, 'PUT', `/api/folders/${id}/move`, {
    new_parent_path: newParentPath,
  });
  expect(response.status).toBe(200);
}

async function renameFolder(page: Page, id: number, newName: string): Promise<void> {
  const response = await requestWithCsrf(page, 'PUT', `/api/folders/${id}/rename`, {
    new_name: newName,
  });
  expect(response.status).toBe(200);
}

async function setFolderColor(page: Page, id: number, color: string | null): Promise<void> {
  const response = await requestWithCsrf(page, 'PUT', `/api/folders/${id}/color`, { color });
  expect(response.status).toBe(200);
}

function findByPath(folders: Folder[], path: string): Folder | undefined {
  return folders.find((folder) => folder.path === path);
}

test.describe('Folders API - Virtual Root (Migration 025)', () => {
  test('no hardcoded root folder with id=1 exists in API response', async ({ page }) => {
    await registerAndLogin(page);
    const folders = await listFolders(page);

    for (const folder of folders) {
      expect(folder.path).not.toBe('/');
      expect(folder.name).not.toBe('Root');
    }
  });

  test('top-level folder created via API has parent_id=null', async ({ page }) => {
    await registerAndLogin(page);
    const folderName = `APITest-${Date.now()}`;

    const folder = await createFolder(page, `/${folderName}`);
    expect(folder.path).toBe(`/${folderName}`);
    expect(folder.name).toBe(folderName);
    expect(folder.parent_id == null).toBeTruthy();
  });

  test('nested folder has correct parent_id pointing to parent folder', async ({ page }) => {
    await registerAndLogin(page);
    const parentName = `Parent-${Date.now()}`;
    const childName = `Child-${Date.now()}`;

    const parent = await createFolder(page, `/${parentName}`);
    const child = await createFolder(page, `/${parentName}/${childName}`);

    expect(parent.parent_id == null).toBeTruthy();
    expect(child.parent_id).toBe(parent.id);
    expect(child.path).toBe(`/${parentName}/${childName}`);
  });

  test('moving folder to root level sets parent_id=null', async ({ page }) => {
    await registerAndLogin(page);
    const parentName = `MoveParent-${Date.now()}`;
    const childName = `MoveChild-${Date.now()}`;

    const parent = await createFolder(page, `/${parentName}`);
    const child = await createFolder(page, `/${parentName}/${childName}`);
    expect(child.parent_id).toBe(parent.id);

    await moveFolder(page, child.id, '/');
    const folders = await listFolders(page);
    const moved = findByPath(folders, `/${childName}`);

    expect(moved).toBeDefined();
    expect(moved?.parent_id == null).toBeTruthy();
  });

  test('moving folder from root into another folder sets parent_id', async ({ page }) => {
    await registerAndLogin(page);
    const folder1Name = `Folder1-${Date.now()}`;
    const folder2Name = `Folder2-${Date.now()}`;

    const folder1 = await createFolder(page, `/${folder1Name}`);
    const folder2 = await createFolder(page, `/${folder2Name}`);
    expect(folder1.parent_id == null).toBeTruthy();
    expect(folder2.parent_id == null).toBeTruthy();

    await moveFolder(page, folder2.id, `/${folder1Name}`);
    const folders = await listFolders(page);
    const moved = findByPath(folders, `/${folder1Name}/${folder2Name}`);

    expect(moved).toBeDefined();
    expect(moved?.parent_id).toBe(folder1.id);
  });
});

test.describe('Folders API - CRUD Operations', () => {
  test('creates folder successfully', async ({ page }) => {
    await registerAndLogin(page);
    const folderName = `CreateTest-${Date.now()}`;

    const folder = await createFolder(page, `/${folderName}`);
    expect(folder.name).toBe(folderName);
    expect(folder.path).toBe(`/${folderName}`);
  });

  test('deletes folder successfully', async ({ page }) => {
    await registerAndLogin(page);
    const folderName = `DeleteTest-${Date.now()}`;

    const folder = await createFolder(page, `/${folderName}`);
    const deleteResponse = await requestWithCsrf(page, 'DELETE', `/api/folders/${folder.id}`);
    expect(deleteResponse.status).toBe(204);

    const folders = await listFolders(page);
    expect(folders.some((item) => item.id === folder.id)).toBeFalsy();
  });

  test('renames folder successfully', async ({ page }) => {
    await registerAndLogin(page);
    const oldName = `OldName-${Date.now()}`;
    const newName = `NewName-${Date.now()}`;

    const folder = await createFolder(page, `/${oldName}`);
    await renameFolder(page, folder.id, newName);
    const folders = await listFolders(page);
    const renamed = findByPath(folders, `/${newName}`);

    expect(renamed).toBeDefined();
    expect(renamed?.id).toBe(folder.id);
    expect(findByPath(folders, `/${oldName}`)).toBeUndefined();
  });

  test('sets and clears folder color', async ({ page }) => {
    await registerAndLogin(page);
    const folderName = `ColorTest-${Date.now()}`;
    const color = '#FF5733';

    const folder = await createFolder(page, `/${folderName}`);
    await setFolderColor(page, folder.id, color);

    let folders = await listFolders(page);
    let updated = folders.find((item) => item.id === folder.id);
    expect(updated?.color).toBe(color);

    await setFolderColor(page, folder.id, null);
    folders = await listFolders(page);
    updated = folders.find((item) => item.id === folder.id);
    expect(updated?.color ?? null).toBeNull();
  });
});

test.describe('Folders API - User Isolation', () => {
  test('users cannot see each other folders via API', async ({ browser }) => {
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const page1 = await context1.newPage();
    const page2 = await context2.newPage();

    try {
      await registerAndLogin(page1, { forceNewUser: true });
      await registerAndLogin(page2, { forceNewUser: true });

      const folder1Name = `User1Folder-${Date.now()}`;
      const folder2Name = `User2Folder-${Date.now() + 1}`;

      await createFolder(page1, `/${folder1Name}`);
      await createFolder(page2, `/${folder2Name}`);

      const user1Folders = await listFolders(page1);
      const user2Folders = await listFolders(page2);

      expect(user1Folders.map((folder) => folder.name)).toContain(folder1Name);
      expect(user1Folders.map((folder) => folder.name)).not.toContain(folder2Name);

      expect(user2Folders.map((folder) => folder.name)).toContain(folder2Name);
      expect(user2Folders.map((folder) => folder.name)).not.toContain(folder1Name);
    } finally {
      await context1.close();
      await context2.close();
    }
  });
});
