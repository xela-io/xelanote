import fs from 'node:fs/promises';
import path from 'node:path';

import type { BrowserContext, Page } from '@playwright/test';

import { test } from '../fixtures/auth.fixture';
import { spoofedClientIP } from './helpers/auth';

async function ensureDir(dir: string) {
  await fs.mkdir(dir, { recursive: true });
}

type UiVariant = {
  locale: 'de' | 'en';
  theme: 'gruvbox-light' | 'gruvbox-dark';
  suffix: string;
};

const DEFAULT_VARIANT: UiVariant = {
  locale: 'en',
  theme: 'gruvbox-light',
  suffix: 'en-gruvbox-light',
};

const REVIEW_VARIANTS: UiVariant[] = [
  DEFAULT_VARIANT,
  { locale: 'de', theme: 'gruvbox-light', suffix: 'de-gruvbox-light' },
  { locale: 'de', theme: 'gruvbox-dark', suffix: 'de-gruvbox-dark' },
];

// Route-specific selectors to wait for before taking screenshot
const ROUTE_READY_HINTS: Record<string, { selector: string; label: string }> = {
  '/recipes': { selector: '[role="tab"], .ui-list-item', label: 'recipe tabs or list items' },
  '/journal': {
    selector: '.journal-heatmap, .journal-entries, [data-journal-loaded]',
    label: 'journal content',
  },
  '/graph': { selector: 'canvas', label: 'graph canvas' },
};

async function waitForAppReady(page: Page) {
  await page.waitForLoadState('load');
  await page.waitForTimeout(800);

  // Wait until the splash/loading screen is gone (best-effort)
  await page
    .waitForFunction(
      () => {
        const text = document.body?.innerText ?? '';
        return !text.includes('Laden...') && !text.includes('Loading...');
      },
      { timeout: 10000 }
    )
    .catch(() => {});
}

async function applyUiVariant(page: Page, variant: UiVariant) {
  await page.addInitScript(({ locale, theme }) => {
    window.localStorage.setItem('locale', locale);
    window.localStorage.setItem('xelanote-theme', theme);
  }, variant);
}

async function captureOnNewPage(
  context: BrowserContext,
  route: string,
  filePath: string,
  viewport: { width: number; height: number },
  variant: UiVariant = DEFAULT_VARIANT
): Promise<string | null> {
  const page = await context.newPage();
  try {
    await page.setViewportSize(viewport);
    await applyUiVariant(page, variant);
    await page.goto(route, { waitUntil: 'domcontentloaded' });
    await waitForAppReady(page);

    // Route-specific waits
    const hint = Object.entries(ROUTE_READY_HINTS).find(([r]) => route.startsWith(r));
    if (hint) {
      await page
        .waitForSelector(hint[1].selector, { timeout: 15000 })
        .catch((err) =>
          console.warn(`[Screenshot] ${route}: ${hint[1].label} not found: ${err.message}`)
        );
      // Extra settle time for dynamic content (graph rendering, heatmap etc.)
      await page.waitForTimeout(500);
    }

    await page.evaluate(() => window.scrollTo(0, 0));

    try {
      await page.screenshot({ path: filePath, animations: 'disabled' });
    } catch {
      await page.locator('body').screenshot({ path: filePath, animations: 'disabled' });
    }
    return null;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return `${route}: ${message}`;
  } finally {
    await page.close().catch(() => {});
  }
}

// ---------------------------------------------------------------------------
// Recipe tab screenshots (Phase 3)
// ---------------------------------------------------------------------------
async function captureRecipeTabs(
  context: BrowserContext,
  recipeId: string,
  outDir: string,
  viewport: { width: number; height: number },
  variant: UiVariant = DEFAULT_VARIANT
): Promise<string[]> {
  const page = await context.newPage();
  const failures: string[] = [];
  try {
    await page.setViewportSize(viewport);
    await applyUiVariant(page, variant);
    await page.goto(`/note/${recipeId}`, { waitUntil: 'domcontentloaded' });
    await waitForAppReady(page);

    // Wait for recipe tab slider
    await page.waitForSelector('[role="tablist"]', { timeout: 10000 });

    const tabs = ['ingredients', 'instructions', 'preview'] as const;
    for (const tab of tabs) {
      const tabButtons = page.locator('[role="tab"]');
      const tabIndex = tabs.indexOf(tab);
      await tabButtons.nth(tabIndex).click();
      await page.waitForTimeout(500); // Tab animation settle

      const filename = `recipe-${tab}--${variant.suffix}.png`;
      try {
        await page.screenshot({ path: path.join(outDir, filename), animations: 'disabled' });
      } catch {
        failures.push(`recipe-${tab}: screenshot failed`);
      }
    }
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error);
    failures.push(`recipe-tabs: ${msg}`);
  } finally {
    await page.close().catch(() => {});
  }
  return failures;
}

// ---------------------------------------------------------------------------
// API helpers for test data seeding
// ---------------------------------------------------------------------------
function getApiHeaders(csrfToken: string | undefined): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Forwarded-For': spoofedClientIP(),
  };
  if (csrfToken) headers['X-CSRF-Token'] = csrfToken;
  return headers;
}

interface SeedResult {
  testRecipeId: string;
}

async function seedTestData(
  page: Page,
  baseURL: string,
  csrfToken: string | undefined
): Promise<SeedResult> {
  const headers = getApiHeaders(csrfToken);

  // --- Enable recipe + journal features ---
  await page.request.put(`${baseURL}/api/features/recipe`, {
    headers,
    data: { enabled: true },
  });
  await page.request.put(`${baseURL}/api/features/journal`, {
    headers,
    data: { enabled: true },
  });

  // --- Graph notes (4 interconnected via wikilinks) ---
  const graphNotes = [
    {
      title: 'Project Planning',
      content:
        '# Project Planning\n\nKey links: [[Architecture Overview]] and [[API Design]].\n\n## Timeline\n- Phase 1: Core features\n- Phase 2: Encryption\n- Phase 3: Collaboration',
    },
    {
      title: 'Architecture Overview',
      content:
        '# Architecture Overview\n\nSee also [[Project Planning]] and [[Database Schema]].\n\n## Stack\n- Go backend with Chi router\n- SvelteKit frontend\n- SQLite database',
    },
    {
      title: 'API Design',
      content:
        '# API Design\n\nRelated: [[Project Planning]] and [[Architecture Overview]].\n\n## Endpoints\n- `GET /api/notes` - List notes\n- `POST /api/notes` - Create note\n- `PUT /api/notes/:id` - Update note',
    },
    {
      title: 'Database Schema',
      content:
        '# Database Schema\n\nReferenced by [[Architecture Overview]] and [[API Design]].\n\n## Tables\n- `notes` - Main content\n- `users` - Authentication\n- `note_links` - Wikilink graph',
    },
  ];

  for (const note of graphNotes) {
    await page.request.post(`${baseURL}/api/notes`, {
      headers,
      data: { title: note.title, content: note.content, folder_path: '/' },
    });
  }

  // --- Recipes (3) ---
  const recipeDefs = [
    {
      title: 'Spaghetti Carbonara',
      content:
        '## Instructions\n\n1. Cook spaghetti in salted boiling water until al dente\n2. Fry guanciale until crispy\n3. Mix eggs with pecorino and pepper\n4. Toss hot pasta with guanciale, then add egg mixture off heat\n5. Serve immediately with extra pecorino',
      servings: 4,
      prep_time_minutes: 15,
      cook_time_minutes: 20,
      difficulty: 'easy',
      ingredients: [
        { name: 'Spaghetti', amount: 400, unit: 'g', group_name: 'Pasta', display_order: 1 },
        { name: 'Guanciale', amount: 200, unit: 'g', group_name: 'Sauce', display_order: 2 },
        { name: 'Egg yolks', amount: 4, unit: 'pcs', group_name: 'Sauce', display_order: 3 },
        {
          name: 'Pecorino Romano',
          amount: 100,
          unit: 'g',
          group_name: 'Sauce',
          display_order: 4,
        },
        {
          name: 'Black pepper',
          amount: 2,
          unit: 'tsp',
          group_name: 'Seasoning',
          display_order: 5,
        },
      ],
    },
    {
      title: 'Chicken Tikka Masala',
      content:
        '## Instructions\n\n1. Marinate chicken in yogurt and spices for 2 hours\n2. Grill or broil marinated chicken pieces\n3. Prepare masala sauce with tomatoes, cream and spices\n4. Combine chicken with sauce and simmer\n5. Garnish with cilantro and serve with naan',
      servings: 6,
      prep_time_minutes: 30,
      cook_time_minutes: 45,
      difficulty: 'medium',
      ingredients: [
        {
          name: 'Chicken thighs',
          amount: 800,
          unit: 'g',
          group_name: 'Chicken',
          display_order: 1,
        },
        { name: 'Yogurt', amount: 200, unit: 'ml', group_name: 'Marinade', display_order: 2 },
        {
          name: 'Garam masala',
          amount: 2,
          unit: 'tbsp',
          group_name: 'Spices',
          display_order: 3,
        },
        {
          name: 'Tomato passata',
          amount: 400,
          unit: 'ml',
          group_name: 'Sauce',
          display_order: 4,
        },
        {
          name: 'Heavy cream',
          amount: 200,
          unit: 'ml',
          group_name: 'Sauce',
          display_order: 5,
        },
      ],
    },
    {
      title: 'Chocolate Lava Cake',
      content:
        '## Instructions\n\n1. Melt dark chocolate and butter together\n2. Whisk eggs and sugar until pale and fluffy\n3. Fold chocolate mixture into eggs, add flour\n4. Pour into greased ramekins\n5. Bake at 220°C for 12 minutes until edges are set but center is soft',
      servings: 2,
      prep_time_minutes: 20,
      cook_time_minutes: 12,
      difficulty: 'hard',
      ingredients: [
        {
          name: 'Dark chocolate (70%)',
          amount: 100,
          unit: 'g',
          group_name: 'Chocolate',
          display_order: 1,
        },
        { name: 'Butter', amount: 50, unit: 'g', group_name: 'Chocolate', display_order: 2 },
        { name: 'Eggs', amount: 2, unit: 'pcs', group_name: 'Batter', display_order: 3 },
        { name: 'Sugar', amount: 50, unit: 'g', group_name: 'Batter', display_order: 4 },
        { name: 'Flour', amount: 20, unit: 'g', group_name: 'Batter', display_order: 5 },
      ],
    },
  ];

  let testRecipeId = '';

  for (const recipe of recipeDefs) {
    const noteResp = await page.request.post(`${baseURL}/api/notes`, {
      headers,
      data: {
        title: recipe.title,
        content: recipe.content,
        folder_path: '/',
        note_type: 'recipe',
      },
    });

    if (!noteResp.ok()) {
      console.warn(`[Seed] Failed to create recipe "${recipe.title}": ${noteResp.status()}`);
      continue;
    }

    const noteData = await noteResp.json();
    const recipeId = noteData.id;

    // Store first recipe ID for tab screenshots
    if (!testRecipeId) testRecipeId = recipeId;

    // Set metadata (empty expected_updated_at = upsert without lock check)
    await page.request.put(`${baseURL}/api/recipes/${recipeId}/metadata`, {
      headers,
      data: {
        servings: recipe.servings,
        prep_time_minutes: recipe.prep_time_minutes,
        cook_time_minutes: recipe.cook_time_minutes,
        difficulty: recipe.difficulty,
        expected_updated_at: '',
      },
    });

    // Set ingredients
    await page.request.put(`${baseURL}/api/recipes/${recipeId}/ingredients`, {
      headers,
      data: {
        ingredients: recipe.ingredients,
        expected_updated_at: '',
      },
    });
  }

  // --- Journal entries (3 recent dates) ---
  const journalEntries = [
    {
      title: 'Productive Monday',
      content:
        '# Productive Monday\n\nFinished the API refactoring and wrote comprehensive tests.\n\n## Highlights\n- Completed recipe metadata API\n- Fixed encryption edge case\n- Reviewed 3 pull requests',
      journal_date: '2026-02-24',
    },
    {
      title: 'Weekend Review',
      content:
        '# Weekend Review\n\nRelaxing weekend with some reading and light coding.\n\n## Notes\n- Read "Designing Data-Intensive Applications"\n- Sketched out graph visualization ideas',
      journal_date: '2026-02-23',
    },
    {
      title: 'Project Kickoff',
      content:
        '# Project Kickoff\n\nStarted planning the new collaboration features.\n\n## Action Items\n- [ ] Design sharing permissions model\n- [ ] Prototype real-time sync\n- [x] Set up development environment',
      journal_date: '2026-02-22',
    },
  ];

  for (const entry of journalEntries) {
    await page.request.post(`${baseURL}/api/notes`, {
      headers,
      data: {
        title: entry.title,
        content: entry.content,
        folder_path: '/',
        note_type: 'journal',
        journal_date: entry.journal_date,
      },
    });
  }

  return { testRecipeId };
}

// ---------------------------------------------------------------------------
// Encryption unlock (Phase 2)
// ---------------------------------------------------------------------------
async function setupEncryptionForScreenshots(
  page: Page,
  baseURL: string,
  password: string
): Promise<void> {
  // Fetch user data with encryption salt
  const meResponse = await page.request.get(`${baseURL}/api/auth/me`);
  if (!meResponse.ok()) {
    console.warn('[Encryption] Failed to fetch /api/auth/me:', meResponse.status());
    return;
  }
  const userData = await meResponse.json();
  const saltBase64: string | undefined = userData.encryption_salt;
  const userId: number | undefined = userData.id;

  if (!saltBase64 || !userId) {
    console.warn('[Encryption] Missing encryption_salt or user id from /api/auth/me');
    return;
  }

  // Use window helper exposed in DEV mode (guaranteed same module instance)
  // The helper is set up via addInitScript to be available before app renders
  try {
    await page.evaluate(
      async ({ pw, uid, salt }) => {
        // Wait for the app to expose the test helper (set in +layout.svelte DEV block)
        let attempts = 0;
        while (!(window as any).__testSetupEncryption && attempts < 50) {
          await new Promise((r) => setTimeout(r, 100));
          attempts++;
        }
        const setupFn = (window as any).__testSetupEncryption;
        if (!setupFn) throw new Error('__testSetupEncryption not found on window');

        const binary = atob(salt);
        const saltArr = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) saltArr[i] = binary.charCodeAt(i);
        await setupFn(pw, uid, saltArr, 'balanced');
      },
      { pw: password, uid: userId, salt: saltBase64 }
    );
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    console.warn(`[Encryption] setupEncryption failed: ${msg}`);
  }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------
test.describe('UI consistency screenshots', () => {
  test('capture desktop core pages', async ({ authenticatedContext }, testInfo) => {
    const { page, testNoteId, baseURL, credentials } = authenticatedContext;
    const context = page.context();
    const outDir = path.join(testInfo.config.rootDir, 'test-results', 'ui-review', 'desktop');
    await ensureDir(outDir);
    const failures: string[] = [];
    const viewport = { width: 1440, height: 900 };

    // Seed test data (graph notes, recipes, journal entries)
    const csrfCookies = await context.cookies(baseURL);
    const csrfToken = csrfCookies.find((c) => c.name === 'csrf_token')?.value;
    const { testRecipeId } = await seedTestData(page, baseURL, csrfToken);

    // Setup encryption so journal page shows content instead of lock screen
    await setupEncryptionForScreenshots(page, baseURL, credentials.password);

    for (const [route, name] of [
      ['/', `home-dashboard--${DEFAULT_VARIANT.suffix}.png`],
      [`/note/${testNoteId}`, `note-editor--${DEFAULT_VARIANT.suffix}.png`],
      ['/recipes', `recipes--${DEFAULT_VARIANT.suffix}.png`],
      ['/journal', `journal--${DEFAULT_VARIANT.suffix}.png`],
      ['/graph', `graph--${DEFAULT_VARIANT.suffix}.png`],
      ['/settings', `settings--${DEFAULT_VARIANT.suffix}.png`],
      ['/settings/encryption', `settings-encryption--${DEFAULT_VARIANT.suffix}.png`],
      ['/settings/migration', `settings-migration--${DEFAULT_VARIANT.suffix}.png`],
    ] as const) {
      const error = await captureOnNewPage(
        context,
        route,
        path.join(outDir, name),
        viewport,
        DEFAULT_VARIANT
      );
      if (error) failures.push(error);
    }

    // Recipe tab screenshots (ingredients, instructions, preview)
    if (testRecipeId) {
      const recipeTabFailures = await captureRecipeTabs(
        context,
        testRecipeId,
        outDir,
        viewport,
        DEFAULT_VARIANT
      );
      failures.push(...recipeTabFailures);
    }

    // Theme/locale matrix on stable pages for design review
    for (const variant of REVIEW_VARIANTS) {
      for (const [route, prefix] of [
        ['/', 'home-dashboard'],
        ['/settings', 'settings'],
        ['/recipes', 'recipes'],
      ] as const) {
        const name = `${prefix}--${variant.suffix}.png`;
        const error = await captureOnNewPage(
          context,
          route,
          path.join(outDir, name),
          viewport,
          variant
        );
        if (error) failures.push(error);
      }
    }

    if (failures.length) {
      await fs.writeFile(
        path.join(outDir, '_capture-failures.txt'),
        failures.join('\n') + '\n',
        'utf8'
      );
    } else {
      await fs.unlink(path.join(outDir, '_capture-failures.txt')).catch(() => {});
    }
  });

  test('capture mobile core pages', async ({ authenticatedContext }, testInfo) => {
    const { page, testNoteId, baseURL, credentials } = authenticatedContext;
    const context = page.context();
    const outDir = path.join(testInfo.config.rootDir, 'test-results', 'ui-review', 'mobile');
    await ensureDir(outDir);
    const failures: string[] = [];
    const viewport = { width: 393, height: 852 };

    // Seed test data (graph notes, recipes, journal entries)
    const csrfCookies = await context.cookies(baseURL);
    const csrfToken = csrfCookies.find((c) => c.name === 'csrf_token')?.value;
    const { testRecipeId } = await seedTestData(page, baseURL, csrfToken);

    // Setup encryption so journal page shows content instead of lock screen
    await setupEncryptionForScreenshots(page, baseURL, credentials.password);

    for (const [route, name] of [
      ['/', `home-dashboard-mobile--${DEFAULT_VARIANT.suffix}.png`],
      [`/note/${testNoteId}`, `note-editor-mobile--${DEFAULT_VARIANT.suffix}.png`],
      ['/recipes', `recipes-mobile--${DEFAULT_VARIANT.suffix}.png`],
      ['/journal', `journal-mobile--${DEFAULT_VARIANT.suffix}.png`],
      ['/settings', `settings-mobile--${DEFAULT_VARIANT.suffix}.png`],
    ] as const) {
      const error = await captureOnNewPage(
        context,
        route,
        path.join(outDir, name),
        viewport,
        DEFAULT_VARIANT
      );
      if (error) failures.push(error);
    }

    // Recipe tab screenshots (mobile)
    if (testRecipeId) {
      const recipeTabFailures = await captureRecipeTabs(
        context,
        testRecipeId,
        outDir,
        viewport,
        DEFAULT_VARIANT
      );
      failures.push(...recipeTabFailures);
    }

    for (const variant of REVIEW_VARIANTS) {
      for (const [route, prefix] of [
        ['/', 'home-dashboard-mobile'],
        ['/settings', 'settings-mobile'],
      ] as const) {
        const name = `${prefix}--${variant.suffix}.png`;
        const error = await captureOnNewPage(
          context,
          route,
          path.join(outDir, name),
          viewport,
          variant
        );
        if (error) failures.push(error);
      }
    }

    if (failures.length) {
      await fs.writeFile(
        path.join(outDir, '_capture-failures.txt'),
        failures.join('\n') + '\n',
        'utf8'
      );
    } else {
      await fs.unlink(path.join(outDir, '_capture-failures.txt')).catch(() => {});
    }
  });
});
