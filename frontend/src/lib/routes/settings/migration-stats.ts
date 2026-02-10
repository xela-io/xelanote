export interface MigrationStats {
  total: number;
  encrypted: number;
  plaintext: number;
}

export interface MigrationStatsDeps<TNote> {
  listNotes: (options: { limit: number }) => Promise<{ notes: TNote[] }>;
  isEncrypted: (note: TNote) => boolean;
  isPlaintext: (note: TNote) => boolean;
  setStats: (stats: MigrationStats | null) => void;
  setLoading: (value: boolean) => void;
}

export async function loadMigrationStats<TNote>(deps: MigrationStatsDeps<TNote>) {
  deps.setLoading(true);
  try {
    const result = await deps.listNotes({ limit: 10000 });
    const notes = result.notes;
    const encrypted = notes.filter(deps.isEncrypted).length;
    const plaintext = notes.filter(deps.isPlaintext).length;
    deps.setStats({
      total: notes.length,
      encrypted,
      plaintext,
    });
  } catch (err) {
    console.error('Failed to load migration stats:', err);
    deps.setStats(null);
  } finally {
    deps.setLoading(false);
  }
}
