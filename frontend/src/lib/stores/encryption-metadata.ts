import type { EncryptedPayload } from '$lib/crypto/e2e';

export function parseEncryptionMetadata(
  raw: string | null | undefined
): EncryptedPayload['metadata'] {
  if (!raw) {
    throw new Error('Missing encryption metadata');
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    throw new Error('Invalid encryption metadata JSON');
  }

  if (!parsed || typeof parsed !== 'object') {
    throw new Error('Invalid encryption metadata payload');
  }

  const metadata = parsed as {
    version?: unknown;
    algorithm?: unknown;
    kdf?: unknown;
    kdf_strength?: unknown;
    nonce_bytes?: unknown;
    wrapped_dek?: unknown;
  };

  if (
    (metadata.version !== 2 && metadata.version !== 3) ||
    metadata.algorithm !== 'XChaCha20-Poly1305' ||
    metadata.kdf !== 'Argon2id' ||
    metadata.kdf_strength !== 'interactive' ||
    metadata.nonce_bytes !== 24 ||
    typeof metadata.wrapped_dek !== 'string'
  ) {
    throw new Error('Invalid encryption metadata fields');
  }

  return {
    version: metadata.version,
    algorithm: 'XChaCha20-Poly1305',
    kdf: 'Argon2id',
    kdf_strength: 'interactive',
    nonce_bytes: 24,
    wrapped_dek: metadata.wrapped_dek,
  };
}
