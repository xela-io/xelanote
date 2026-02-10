use crate::kek::KekManager;
use crate::keyring::{self, AuthTokens};
use tauri::State;

// ===== Token Management Commands =====

/// Store auth tokens for a server in the OS keyring (or encrypted fallback).
#[tauri::command]
pub async fn store_auth_tokens(server_url: String, tokens: AuthTokens) -> Result<(), String> {
    keyring::store_tokens(&server_url, &tokens)
}

/// Load auth tokens for a server from the OS keyring (or encrypted fallback).
#[tauri::command]
pub async fn load_auth_tokens(server_url: String) -> Result<Option<AuthTokens>, String> {
    keyring::load_tokens(&server_url)
}

/// Delete auth tokens for a server from the OS keyring and encrypted fallback.
#[tauri::command]
pub async fn delete_auth_tokens(server_url: String) -> Result<(), String> {
    keyring::delete_tokens(&server_url)
}

// ===== KEK Management Commands =====

/// Store KEK (Key Encryption Key) in secure Rust memory.
#[tauri::command]
pub async fn store_kek(kek_manager: State<'_, KekManager>, kek: Vec<u8>) -> Result<(), String> {
    kek_manager.store_kek(kek)
}

/// Get KEK from Rust memory.
#[tauri::command]
pub async fn get_kek(kek_manager: State<'_, KekManager>) -> Result<Option<Vec<u8>>, String> {
    kek_manager.get_kek()
}

/// Lock KEK by securely clearing it from Rust memory.
#[tauri::command]
pub async fn lock_kek(kek_manager: State<'_, KekManager>) -> Result<(), String> {
    kek_manager.lock()
}

/// Check if KEK is locked (not in Rust memory).
#[tauri::command]
pub async fn is_kek_locked(kek_manager: State<'_, KekManager>) -> Result<bool, String> {
    kek_manager.is_locked()
}
