export interface EncryptedAttachmentMigrationAuditCounters {
  detectedNotes: number;
  detectedLinks: number;
  persistedNotes: number;
  persistedLinks: number;
  failedNotes: number;
  failedLinks: number;
}

const counters: EncryptedAttachmentMigrationAuditCounters = {
  detectedNotes: 0,
  detectedLinks: 0,
  persistedNotes: 0,
  persistedLinks: 0,
  failedNotes: 0,
  failedLinks: 0,
};

export function recordEncryptedAttachmentMigrationDetected(migratedLinks: number): void {
  counters.detectedNotes += 1;
  counters.detectedLinks += migratedLinks;
  console.info('[ENCRYPTION] Legacy encrypted attachment metadata migration detected', {
    migrated_links: migratedLinks,
  });
}

export function recordEncryptedAttachmentMigrationPersisted(migratedLinks: number): void {
  counters.persistedNotes += 1;
  counters.persistedLinks += migratedLinks;
  console.info('[ENCRYPTION] Legacy encrypted attachment metadata migration persisted', {
    migrated_links: migratedLinks,
  });
}

export function recordEncryptedAttachmentMigrationFailed(
  migratedLinks: number,
  reason: 'conflict' | 'update_failed'
): void {
  counters.failedNotes += 1;
  counters.failedLinks += migratedLinks;
  console.warn('[ENCRYPTION] Legacy encrypted attachment metadata migration persist failed', {
    migrated_links: migratedLinks,
    reason,
  });
}

export function getEncryptedAttachmentMigrationAuditCounters(): EncryptedAttachmentMigrationAuditCounters {
  return { ...counters };
}

export function _resetEncryptedAttachmentMigrationAuditForTests(): void {
  counters.detectedNotes = 0;
  counters.detectedLinks = 0;
  counters.persistedNotes = 0;
  counters.persistedLinks = 0;
  counters.failedNotes = 0;
  counters.failedLinks = 0;
}
