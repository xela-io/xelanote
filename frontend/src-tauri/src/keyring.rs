use aes_gcm::{
    aead::{Aead, KeyInit},
    Aes256Gcm, Nonce,
};
use keyring::Entry;
use rand::RngCore;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs;
use std::path::PathBuf;

/// Auth tokens for a server connection.
#[derive(Serialize, Deserialize, Clone)]
pub struct AuthTokens {
    pub access_token: String,
    pub refresh_token: String,
    pub user_id: Option<i64>,
}

/// Sanitize server URL for use as keyring/filename key.
fn sanitize_server_key(server_url: &str) -> String {
    server_url
        .replace("https://", "")
        .replace("http://", "")
        .replace('/', "_")
        .replace(':', "_")
}

/// Get fallback encrypted file path for a server.
fn fallback_path(server_url: &str) -> PathBuf {
    let key = sanitize_server_key(server_url);
    dirs::data_local_dir()
        .unwrap_or_else(|| PathBuf::from("."))
        .join("xelanote")
        .join(format!("{}.tokens.enc", key))
}

/// Derive encryption key from machine-specific data.
/// This ensures tokens cannot be decrypted on a different machine.
fn derive_fallback_key() -> [u8; 32] {
    // Use machine-id + username as key derivation source
    let machine_id = fs::read_to_string("/etc/machine-id")
        .unwrap_or_else(|_| "fallback-id".to_string());
    let username = std::env::var("USER").unwrap_or_else(|_| "user".to_string());

    let mut hasher = Sha256::new();
    hasher.update(machine_id.trim().as_bytes());
    hasher.update(username.as_bytes());
    hasher.update(b"xelanote-token-encryption");
    hasher.finalize().into()
}

/// Store auth tokens for a server.
/// Tries OS keyring first, falls back to AES-256-GCM encrypted file.
pub fn store_tokens(server_url: &str, tokens: &AuthTokens) -> Result<(), String> {
    let key = sanitize_server_key(server_url);
    let json = serde_json::to_string(tokens).map_err(|e| e.to_string())?;

    // Try OS keyring first
    if let Ok(entry) = Entry::new("xelanote", &format!("tokens_{}", key)) {
        if entry.set_password(&json).is_ok() {
            return Ok(());
        }
    }

    // Fallback: AES-256-GCM encrypted file
    let path = fallback_path(server_url);
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).map_err(|e| e.to_string())?;
    }

    let encryption_key = derive_fallback_key();
    let cipher =
        Aes256Gcm::new_from_slice(&encryption_key).map_err(|e| format!("Cipher error: {}", e))?;

    // Generate random 12-byte nonce
    let mut nonce_bytes = [0u8; 12];
    rand::thread_rng().fill_bytes(&mut nonce_bytes);
    let nonce = Nonce::from_slice(&nonce_bytes);

    let ciphertext = cipher
        .encrypt(nonce, json.as_bytes())
        .map_err(|e| format!("Encryption error: {}", e))?;

    // Store: nonce (12 bytes) || ciphertext
    let mut encrypted = nonce_bytes.to_vec();
    encrypted.extend(ciphertext);

    fs::write(&path, &encrypted).map_err(|e| e.to_string())?;
    Ok(())
}

/// Load auth tokens for a server.
/// Tries OS keyring first, falls back to decrypting file.
pub fn load_tokens(server_url: &str) -> Result<Option<AuthTokens>, String> {
    let key = sanitize_server_key(server_url);

    // Try OS keyring first
    if let Ok(entry) = Entry::new("xelanote", &format!("tokens_{}", key)) {
        if let Ok(json) = entry.get_password() {
            if let Ok(tokens) = serde_json::from_str(&json) {
                return Ok(Some(tokens));
            }
        }
    }

    // Fallback: decrypt file
    let path = fallback_path(server_url);
    if !path.exists() {
        return Ok(None);
    }

    let encrypted = fs::read(&path).map_err(|e| e.to_string())?;
    if encrypted.len() < 13 {
        // 12 byte nonce + at least 1 byte
        return Err("Invalid encrypted file".to_string());
    }

    let encryption_key = derive_fallback_key();
    let cipher =
        Aes256Gcm::new_from_slice(&encryption_key).map_err(|e| format!("Cipher error: {}", e))?;

    let nonce = Nonce::from_slice(&encrypted[..12]);
    let plaintext = cipher
        .decrypt(nonce, &encrypted[12..])
        .map_err(|_| "Decryption failed - machine may have changed".to_string())?;

    let json = String::from_utf8(plaintext).map_err(|e| e.to_string())?;
    let tokens = serde_json::from_str(&json).map_err(|e| e.to_string())?;
    Ok(Some(tokens))
}

/// Delete auth tokens for a server.
/// Clears from both keyring and fallback file.
pub fn delete_tokens(server_url: &str) -> Result<(), String> {
    let key = sanitize_server_key(server_url);

    // Try OS keyring
    if let Ok(entry) = Entry::new("xelanote", &format!("tokens_{}", key)) {
        let _ = entry.delete_credential();
    }

    // Also delete fallback file
    let path = fallback_path(server_url);
    if path.exists() {
        fs::remove_file(&path).map_err(|e| e.to_string())?;
    }

    Ok(())
}
