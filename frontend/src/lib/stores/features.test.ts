import { beforeEach, describe, expect, it, vi } from 'vitest';

const getFeature = vi.fn();
const setFeature = vi.fn();

vi.mock('$lib/api', () => ({
  getFeature,
  setFeature,
}));

vi.mock('$lib/config', () => ({
  getApiBaseUrl: vi.fn(() => 'http://localhost:8080/api'),
}));

describe('features store', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
  });

  describe('journal feature', () => {
    it('loads journal feature and sets state', async () => {
      getFeature.mockResolvedValue({ enabled: true });
      const features = await import('$lib/stores/features.svelte');

      await features.loadJournalFeature();

      expect(getFeature).toHaveBeenCalledWith('journal');
      expect(features.getJournalFeatureEnabled()).toBe(true);
      expect(features.getJournalFeatureLoaded()).toBe(true);
      expect(features.getJournalFeatureLoading()).toBe(false);
    });

    it('logs load errors and keeps loaded=true with enabled=false', async () => {
      const err = new Error('journal-load-failed');
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      getFeature.mockRejectedValue(err);
      const features = await import('$lib/stores/features.svelte');

      await features.loadJournalFeature();

      expect(features.getJournalFeatureEnabled()).toBe(false);
      expect(features.getJournalFeatureLoaded()).toBe(true);
      expect(features.getJournalFeatureLoading()).toBe(false);
      expect(consoleError).toHaveBeenCalledWith('Failed to load journal feature:', err);
      consoleError.mockRestore();
    });

    it('toggles journal feature', async () => {
      setFeature.mockResolvedValue({ success: true });
      const features = await import('$lib/stores/features.svelte');

      await features.toggleJournalFeature(true);

      expect(setFeature).toHaveBeenCalledWith('journal', true);
      expect(features.getJournalFeatureEnabled()).toBe(true);
      expect(features.getJournalFeatureLoading()).toBe(false);
    });

    it('resets journal feature state', async () => {
      getFeature.mockResolvedValue({ enabled: true });
      const features = await import('$lib/stores/features.svelte');
      await features.loadJournalFeature();

      features.resetJournalFeature();

      expect(features.getJournalFeatureEnabled()).toBe(false);
      expect(features.getJournalFeatureLoaded()).toBe(false);
      expect(features.getJournalFeatureLoading()).toBe(false);
    });
  });

  describe('recipe feature', () => {
    it('loads recipe feature and sets state', async () => {
      getFeature.mockResolvedValue({ enabled: true });
      const features = await import('$lib/stores/features.svelte');

      await features.loadRecipeFeature();

      expect(getFeature).toHaveBeenCalledWith('recipe');
      expect(features.getRecipeFeatureEnabled()).toBe(true);
      expect(features.getRecipeFeatureLoaded()).toBe(true);
      expect(features.getRecipeFeatureLoading()).toBe(false);
    });

    it('logs toggle errors and resets loading before rethrow', async () => {
      const err = new Error('recipe-toggle-failed');
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      setFeature.mockRejectedValue(err);
      const features = await import('$lib/stores/features.svelte');

      await expect(features.toggleRecipeFeature(true)).rejects.toThrow('recipe-toggle-failed');

      expect(setFeature).toHaveBeenCalledWith('recipe', true);
      expect(features.getRecipeFeatureEnabled()).toBe(false);
      expect(features.getRecipeFeatureLoading()).toBe(false);
      expect(consoleError).toHaveBeenCalledWith('Failed to toggle recipe feature:', err);
      consoleError.mockRestore();
    });

    it('resets recipe feature state', async () => {
      getFeature.mockResolvedValue({ enabled: true });
      const features = await import('$lib/stores/features.svelte');
      await features.loadRecipeFeature();

      features.resetRecipeFeature();

      expect(features.getRecipeFeatureEnabled()).toBe(false);
      expect(features.getRecipeFeatureLoaded()).toBe(false);
      expect(features.getRecipeFeatureLoading()).toBe(false);
    });
  });

  describe('canvas feature', () => {
    it('loads canvas feature and updates loaded/enabled state', async () => {
      getFeature.mockResolvedValue({ enabled: true });
      const features = await import('$lib/stores/features.svelte');

      expect(features.getCanvasFeatureEnabled()).toBe(false);
      expect(features.getCanvasFeatureLoaded()).toBe(false);
      expect(features.getCanvasFeatureLoading()).toBe(false);

      await features.loadCanvasFeature();

      expect(getFeature).toHaveBeenCalledWith('canvas');
      expect(features.getCanvasFeatureEnabled()).toBe(true);
      expect(features.getCanvasFeatureLoaded()).toBe(true);
      expect(features.getCanvasFeatureLoading()).toBe(false);
    });

    it('marks canvas feature as loaded=true and disabled on load failure', async () => {
      const err = new Error('canvas-load-failed');
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      getFeature.mockRejectedValue(err);
      const features = await import('$lib/stores/features.svelte');

      await features.loadCanvasFeature();

      expect(getFeature).toHaveBeenCalledWith('canvas');
      expect(features.getCanvasFeatureEnabled()).toBe(false);
      expect(features.getCanvasFeatureLoaded()).toBe(true);
      expect(features.getCanvasFeatureLoading()).toBe(false);
      expect(consoleError).toHaveBeenCalledWith('Failed to load canvas feature:', err);
      consoleError.mockRestore();
    });

    it('does not execute a second concurrent canvas load while loading', async () => {
      let releaseLoad: () => void = () => undefined;
      const loadGate = new Promise<void>((resolve) => {
        releaseLoad = resolve;
      });
      getFeature.mockImplementation(async () => {
        await loadGate;
        return { enabled: true };
      });
      const features = await import('$lib/stores/features.svelte');

      const firstLoad = features.loadCanvasFeature();
      expect(features.getCanvasFeatureLoading()).toBe(true);

      await features.loadCanvasFeature();
      expect(getFeature).toHaveBeenCalledTimes(1);

      releaseLoad();
      await firstLoad;

      expect(features.getCanvasFeatureEnabled()).toBe(true);
      expect(features.getCanvasFeatureLoaded()).toBe(true);
      expect(features.getCanvasFeatureLoading()).toBe(false);
    });

    it('toggles canvas feature and updates state', async () => {
      setFeature.mockResolvedValue({ success: true });
      const features = await import('$lib/stores/features.svelte');

      await features.toggleCanvasFeature(true);

      expect(setFeature).toHaveBeenCalledWith('canvas', true);
      expect(features.getCanvasFeatureEnabled()).toBe(true);
      expect(features.getCanvasFeatureLoading()).toBe(false);
    });

    it('rethrows canvas toggle errors and resets loading', async () => {
      const err = new Error('canvas-toggle-failed');
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      setFeature.mockRejectedValue(err);
      const features = await import('$lib/stores/features.svelte');

      await expect(features.toggleCanvasFeature(true)).rejects.toThrow('canvas-toggle-failed');

      expect(setFeature).toHaveBeenCalledWith('canvas', true);
      expect(features.getCanvasFeatureEnabled()).toBe(false);
      expect(features.getCanvasFeatureLoading()).toBe(false);
      expect(consoleError).toHaveBeenCalledWith('Failed to toggle canvas feature:', err);
      consoleError.mockRestore();
    });

    it('resets canvas feature state', async () => {
      getFeature.mockResolvedValue({ enabled: true });
      const features = await import('$lib/stores/features.svelte');

      await features.loadCanvasFeature();
      expect(features.getCanvasFeatureEnabled()).toBe(true);
      expect(features.getCanvasFeatureLoaded()).toBe(true);

      features.resetCanvasFeature();

      expect(features.getCanvasFeatureEnabled()).toBe(false);
      expect(features.getCanvasFeatureLoaded()).toBe(false);
      expect(features.getCanvasFeatureLoading()).toBe(false);
    });
  });
});
