#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod commands;
mod kek;
mod keyring;

use kek::KekManager;

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(KekManager::new())
        .invoke_handler(tauri::generate_handler![
            commands::store_auth_tokens,
            commands::load_auth_tokens,
            commands::delete_auth_tokens,
            commands::store_kek,
            commands::get_kek,
            commands::lock_kek,
            commands::is_kek_locked,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
