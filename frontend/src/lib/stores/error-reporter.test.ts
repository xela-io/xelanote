import { describe, expect, it, beforeEach } from 'vitest';
import {
  normalizeMessage,
  computeFingerprint,
  _resetForTesting,
} from '$lib/stores/error-reporter.svelte';

describe('error-reporter', () => {
  beforeEach(() => {
    _resetForTesting();
  });

  describe('normalizeMessage', () => {
    it('replaces standalone numbers with N', () => {
      expect(normalizeMessage('Error at line 42')).toBe('Error at line N');
    });

    it('replaces UUIDs with UUID', () => {
      expect(normalizeMessage('Note 550e8400-e29b-41d4-a716-446655440000 not found')).toBe(
        'Note UUID not found'
      );
    });

    it('replaces ISO dates with DATE', () => {
      expect(normalizeMessage('Created at 2024-01-15T10:30:00Z')).toBe('Created at DATE');
    });

    it('handles mixed content', () => {
      const input =
        'User 42 created note 550e8400-e29b-41d4-a716-446655440000 at 2024-01-15T10:30:00.123Z';
      const expected = 'User N created note UUID at DATE';
      expect(normalizeMessage(input)).toBe(expected);
    });

    it('preserves non-numeric text', () => {
      expect(normalizeMessage('TypeError: foo is not defined')).toBe(
        'TypeError: foo is not defined'
      );
    });
  });

  describe('computeFingerprint', () => {
    it('returns 16 hex characters', async () => {
      const fp = await computeFingerprint('TypeError', 'foo is not defined');
      expect(fp).toMatch(/^[0-9a-f]{16}$/);
    });

    it('returns same fingerprint for same input', async () => {
      const fp1 = await computeFingerprint('TypeError', 'foo is not defined');
      const fp2 = await computeFingerprint('TypeError', 'foo is not defined');
      expect(fp1).toBe(fp2);
    });

    it('returns different fingerprint for different input', async () => {
      const fp1 = await computeFingerprint('TypeError', 'foo is not defined');
      const fp2 = await computeFingerprint('ReferenceError', 'bar is not defined');
      expect(fp1).not.toBe(fp2);
    });

    it('normalizes numbers in message', async () => {
      const fp1 = await computeFingerprint('Error', 'Failed at line 42');
      const fp2 = await computeFingerprint('Error', 'Failed at line 99');
      expect(fp1).toBe(fp2);
    });

    it('normalizes UUIDs in message', async () => {
      const fp1 = await computeFingerprint(
        'Error',
        'Note 550e8400-e29b-41d4-a716-446655440000 not found'
      );
      const fp2 = await computeFingerprint(
        'Error',
        'Note a1b2c3d4-e5f6-7890-abcd-ef1234567890 not found'
      );
      expect(fp1).toBe(fp2);
    });
  });
});
