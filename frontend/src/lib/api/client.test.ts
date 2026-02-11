import { beforeEach, describe, expect, it, vi } from 'vitest';

const enqueueOperation = vi.fn();
const getQueueCount = vi.fn(async () => 0);

vi.mock('$lib/config', () => ({
  getApiBaseUrl: vi.fn(() => 'http://api'),
  isDesktop: vi.fn(() => true),
}));

vi.mock('$lib/offline/offline-queue', () => ({
  enqueueOperation,
  getQueueCount,
}));

describe('api client', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    document.cookie = '';
    vi.stubGlobal('fetch', vi.fn());
  });

  it('adds auth, csrf, and desktop headers and returns JSON', async () => {
    document.cookie = 'csrf_token=abc';

    const { initApiAuth, request } = await import('$lib/api/client');
    let accessToken = 'token-1';
    initApiAuth(
      () => accessToken,
      () => null,
      (nextAccess) => {
        accessToken = nextAccess;
      },
      () => undefined
    );

    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true }),
    } as Response);

    const result = await request<{ ok: boolean }>('/notes', {
      method: 'POST',
      body: JSON.stringify({ title: 'Hello' }),
    });

    expect(result.ok).toBe(true);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [, options] = fetchMock.mock.calls[0] as [string, RequestInit];
    const headers = new Headers(options.headers);
    expect(headers.get('Authorization')).toBe('Bearer token-1');
    expect(headers.get('X-CSRF-Token')).toBe('abc');
    expect(headers.get('X-Client-Type')).toBe('desktop');
    expect(headers.get('Content-Type')).toBe('application/json');
  });

  it('returns undefined for 204 responses', async () => {
    const { request } = await import('$lib/api/client');
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue({
      ok: true,
      status: 204,
      json: async () => ({}),
    } as Response);

    const result = await request('/notes', { method: 'DELETE' });
    expect(result).toBeUndefined();
  });

  it('queues offline note creation when offline and allowed', async () => {
    const navigatorBackup = globalThis.navigator;
    Object.defineProperty(globalThis, 'navigator', {
      value: { onLine: false },
      configurable: true,
    });

    const randomUUID = vi.fn().mockReturnValueOnce('temp-id').mockReturnValueOnce('op-id');
    const cryptoBackup = globalThis.crypto;
    Object.defineProperty(globalThis, 'crypto', {
      value: { randomUUID },
      configurable: true,
    });

    const onEnqueue = vi.fn();
    getQueueCount.mockResolvedValueOnce(3);

    const { request, setOnOfflineEnqueue } = await import('$lib/api/client');
    setOnOfflineEnqueue(onEnqueue);

    const created = await request<{ id: string; folder_path: string; content_encrypted?: boolean }>(
      '/notes',
      {
        method: 'POST',
        body: JSON.stringify({
          folder_path: '/work',
          encrypted_content: 'cipher',
          encryption_metadata: JSON.stringify({ version: 2 }),
        }),
        _offlineAllowed: true,
      }
    );

    expect(created.id).toBe('temp_temp-id');
    expect(created.folder_path).toBe('/work');
    expect(created.content_encrypted).toBe(true);
    expect(enqueueOperation).toHaveBeenCalledTimes(1);
    expect(onEnqueue).toHaveBeenCalledWith(3);

    Object.defineProperty(globalThis, 'navigator', {
      value: navigatorBackup,
      configurable: true,
    });
    Object.defineProperty(globalThis, 'crypto', {
      value: cryptoBackup,
      configurable: true,
    });
  });

  it('refreshes on 401 and retries with new token', async () => {
    const { initApiAuth, request } = await import('$lib/api/client');
    let accessToken = 'old-token';
    let refreshToken = 'refresh-token';
    initApiAuth(
      () => accessToken,
      () => refreshToken,
      (nextAccess, nextRefresh) => {
        accessToken = nextAccess;
        refreshToken = nextRefresh;
      },
      () => undefined
    );

    let first = true;
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockImplementation(async (url, options) => {
      if (typeof url === 'string' && url.endsWith('/auth/refresh')) {
        return {
          ok: true,
          status: 200,
          json: async () => ({ access_token: 'new-token', refresh_token: 'r2' }),
        } as Response;
      }

      if (first) {
        first = false;
        return { ok: false, status: 401 } as Response;
      }

      const headers = new Headers((options as RequestInit)?.headers);
      expect(headers.get('Authorization')).toBe('Bearer new-token');
      return {
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      } as Response;
    });

    const result = await request<{ ok: boolean }>('/notes', { method: 'GET' });
    expect(result.ok).toBe(true);
  });

  it('throws when offline and offline mutations are not allowed', async () => {
    const navigatorBackup = globalThis.navigator;
    Object.defineProperty(globalThis, 'navigator', {
      value: { onLine: false },
      configurable: true,
    });

    const { ApiError, request } = await import('$lib/api/client');

    await expect(
      request('/notes', {
        method: 'POST',
        body: JSON.stringify({ title: 'Nope' }),
      })
    ).rejects.toBeInstanceOf(ApiError);

    Object.defineProperty(globalThis, 'navigator', {
      value: navigatorBackup,
      configurable: true,
    });
  });
});
