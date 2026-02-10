/**
 * KEK (Key Encryption Key) Manager
 *
 * Manages in-memory storage of the encryption key.
 * Note: V8's garbage collector may retain copies of the key in memory.
 * For higher security requirements, use Tauri with Rust's zeroize crate.
 */

class KekManager {
  private kek: Buffer | null = null;

  /**
   * Store KEK in memory.
   * Any existing KEK is cleared first (best effort).
   *
   * @param kek - Key Encryption Key as Uint8Array
   */
  store(kek: Uint8Array): void {
    // Clear existing key (best effort)
    if (this.kek) {
      this.kek.fill(0);
    }

    // Store new key
    this.kek = Buffer.from(kek);
    console.log('[KekManager] KEK stored in memory');
  }

  /**
   * Get KEK from memory.
   *
   * @returns KEK as Uint8Array or null if not stored
   */
  get(): Uint8Array | null {
    if (!this.kek) {
      return null;
    }
    // Return a copy to prevent external modification
    return new Uint8Array(this.kek);
  }

  /**
   * Lock (clear) KEK from memory.
   * Uses fill(0) as best-effort secure clearing.
   */
  lock(): void {
    if (this.kek) {
      this.kek.fill(0);
      this.kek = null;
      console.log('[KekManager] KEK cleared from memory');
    }
  }

  /**
   * Check if KEK is locked (not in memory).
   *
   * @returns true if KEK is not stored
   */
  isLocked(): boolean {
    return this.kek === null;
  }
}

// Singleton instance
export const kekManager = new KekManager();
