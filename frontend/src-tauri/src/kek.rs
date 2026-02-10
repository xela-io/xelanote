use std::sync::Mutex;
use zeroize::Zeroize;

/// KEK (Key Encryption Key) manager for secure in-memory storage.
/// Provides defense-in-depth: XSS would need to call invoke() to access KEK.
/// The TypeScript auto-lock timer handles timing; Rust only provides secure storage.
pub struct KekManager {
    kek: Mutex<Option<Vec<u8>>>,
}

impl KekManager {
    pub fn new() -> Self {
        Self {
            kek: Mutex::new(None),
        }
    }

    /// Store KEK in secure memory.
    pub fn store_kek(&self, kek: Vec<u8>) -> Result<(), String> {
        let mut guard = self.kek.lock().map_err(|_| "KEK mutex poisoned")?;
        *guard = Some(kek);
        Ok(())
    }

    /// Get KEK from memory (clone to prevent moving out).
    pub fn get_kek(&self) -> Result<Option<Vec<u8>>, String> {
        let guard = self.kek.lock().map_err(|_| "KEK mutex poisoned")?;
        Ok(guard.clone())
    }

    /// Lock encryption by securely clearing KEK from memory.
    pub fn lock(&self) -> Result<(), String> {
        let mut guard = self.kek.lock().map_err(|_| "KEK mutex poisoned")?;
        if let Some(mut kek) = guard.take() {
            kek.zeroize();
        }
        Ok(())
    }

    /// Check if KEK is locked (not in memory).
    pub fn is_locked(&self) -> Result<bool, String> {
        let guard = self.kek.lock().map_err(|_| "KEK mutex poisoned")?;
        Ok(guard.is_none())
    }
}

impl Default for KekManager {
    fn default() -> Self {
        Self::new()
    }
}
