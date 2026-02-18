import { beforeEach, describe, expect, it, vi } from 'vitest';

const request = vi.fn();

vi.mock('./client', () => ({
  request,
}));

describe('features api', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
  });

  it('getFeature requests the correct endpoint', async () => {
    request.mockResolvedValue({ enabled: true });
    const { getFeature } = await import('./features');

    const result = await getFeature('canvas');

    expect(request).toHaveBeenCalledWith('/features/canvas');
    expect(result).toEqual({ enabled: true });
  });

  it('setFeature sends PUT request with enabled payload', async () => {
    request.mockResolvedValue({ success: true });
    const { setFeature } = await import('./features');

    await setFeature('canvas', true);

    expect(request).toHaveBeenCalledWith('/features/canvas', {
      method: 'PUT',
      body: JSON.stringify({ enabled: true, settings: undefined }),
    });
  });

  it('setFeature sends settings when provided', async () => {
    request.mockResolvedValue({ success: true });
    const { setFeature } = await import('./features');

    await setFeature('recipe', true, { locale: 'de-DE' });

    expect(request).toHaveBeenCalledWith('/features/recipe', {
      method: 'PUT',
      body: JSON.stringify({ enabled: true, settings: { locale: 'de-DE' } }),
    });
  });

  it('listFeatures requests the list endpoint', async () => {
    request.mockResolvedValue([]);
    const { listFeatures } = await import('./features');

    const result = await listFeatures();

    expect(request).toHaveBeenCalledWith('/features');
    expect(result).toEqual([]);
  });
});
