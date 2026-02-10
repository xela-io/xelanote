import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import * as toast from '$lib/stores/toast.svelte';

function clearToasts() {
  for (const entry of toast.getToasts()) {
    toast.removeToast(entry.id);
  }
}

describe('toast store', () => {
  beforeEach(() => {
    clearToasts();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.runOnlyPendingTimers();
    vi.useRealTimers();
    clearToasts();
  });

  it('limits the queue to the most recent 3 toasts', () => {
    toast.success('one');
    toast.success('two');
    toast.success('three');
    toast.success('four');

    expect(toast.getToasts().map((entry) => entry.message)).toEqual(['two', 'three', 'four']);
  });

  it('auto-removes toasts after their duration', () => {
    toast.success('auto-dismiss');
    expect(toast.getToasts()).toHaveLength(1);

    vi.advanceTimersByTime(3000);
    expect(toast.getToasts()).toHaveLength(0);
  });
});
