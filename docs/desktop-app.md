# Desktop App

Die xelanote Desktop-App bietet eine native Anwendung mit verbesserter Sicherheit durch OS-Keyring-Integration und In-Memory-Verschlüsselung.

**Verfügbare Builds:**
- **Electron** (empfohlen) - Stabil, produktionsbereit
- **Tauri** (experimentell) - Kleinere Binary, aber noch in Entwicklung

---

## Electron Desktop App

## Quick Start

```bash
cd frontend

# Browser-Entwicklung gegen lokales Backend (Standard)
npm run dev:local

# Development (lokales Backend auf :8080)
npm run electron:dev

# Development gegen Production-Server (empfohlen für Login/CAPTCHA-Tests)
npm run electron:dev:prod

# Production Build (Linux)
npm run electron:build:linux

# Output:
# - release/xelanote-0.1.0-x86_64.AppImage
# - release/xelanote-frontend_0.1.0_amd64.deb
```

## Installation

### Arch-basiert (CachyOS, Manjaro, EndeavourOS)

```bash
# AppImage installieren
mkdir -p ~/.local/bin
cp release/xelanote-0.1.0-x86_64.AppImage ~/.local/bin/xelanote
chmod +x ~/.local/bin/xelanote

# Desktop-Eintrag erstellen
mkdir -p ~/.local/share/applications
echo '[Desktop Entry]
Name=xelanote
Comment=Personal note-taking with wikilinks
Exec=xelanote --no-sandbox
Terminal=false
Type=Application
Categories=Office;TextEditor;Utility;
Keywords=notes;markdown;wikilinks;' > ~/.local/share/applications/xelanote.desktop
```

### Debian-basiert (Ubuntu, Mint, Pop!_OS)

```bash
sudo dpkg -i release/xelanote-frontend_0.1.0_amd64.deb
```

## Starten

- **Anwendungsmenü**: Nach "xelanote" suchen
- **Terminal**: `xelanote --no-sandbox`

## Features

- Frameless Window mit custom TitleBar
- Sichere Token-Speicherung im System-Keychain
- CAPTCHA-Bypass für Desktop-Clients
- Automatischer Login beim App-Start
- WebAssembly Crypto (libsodium)

## Architektur

```
frontend/
├── src-electron/
│   ├── main.ts              # Electron Main Process
│   │                        # - Custom app:// Protocol
│   │                        # - CORS Bypass für API
│   │                        # - CSP Configuration
│   ├── preload.ts           # Context Bridge API
│   ├── modules/
│   │   ├── ipc-handlers.ts  # IPC Handler Registration
│   │   ├── secure-storage.ts # Token Storage (Keychain)
│   │   ├── kek-manager.ts   # Encryption Key Management
│   │   └── window-manager.ts # Window State Management
│   └── windows/
│       └── main-window.ts   # Main Window Factory
├── electron-builder.yml     # Build Configuration
└── electron.vite.config.ts  # Vite Config for Electron
```

## Technische Details

### Custom Protocol (`app://`)

In Production werden statische Dateien über ein custom `app://` Protocol geladen,
da das `file://` Protocol absolute Pfade wie `/_app/...` nicht korrekt auflöst.

```typescript
// main.ts
protocol.handle('app', (request) => {
  const filePath = new URL(request.url).pathname;
  return net.fetch(pathToFileURL(join(buildPath, filePath)).toString());
});
```

### CORS Bypass

Desktop-Apps können nicht die üblichen CORS-Header senden, da `app://` keine
gültige HTTP-Origin ist. Daher werden Origin-Header im Main Process entfernt:

```typescript
session.defaultSession.webRequest.onBeforeSendHeaders(
  { urls: ['https://xelanote.com/*'] },
  (details, callback) => {
    delete details.requestHeaders['Origin'];
    callback({ requestHeaders: details.requestHeaders });
  }
);
```

Für lokale Electron-Entwicklung gegen Production wird stattdessen ein
Same-Origin-Proxy verwendet (`VITE_API_BASE_URL=/api`,
`VITE_API_PROXY_TARGET=https://xelanote.com`). Das vermeidet CORS-Probleme im
Renderer bei aktiviertem `webSecurity`.

### CAPTCHA (Pflicht)

CAPTCHA bleibt für Login und Registrierung auch in Desktop-Apps verpflichtend.
Es gibt keinen Desktop-Bypass mehr.

## Known Issues

- `--no-sandbox` Flag erforderlich auf einigen Linux-Systemen
- GPU-Beschleunigung deaktiviert für Kompatibilität
- `Port 5173 is already in use` beim Start:
  - Prozess finden: `lsof -i :5173`
  - Prozess beenden: `kill <PID>`
  - Alternative: `pkill -f "vite dev --port 5173"`

## TODO

- [ ] App-Icon erstellen und einbinden
- [ ] Windows Build testen
- [ ] macOS Build testen
- [ ] Auto-Update implementieren

---

## Tauri Desktop App (Experimentell)

## Inhaltsverzeichnis

1. [Überblick](#überblick)
2. [Architektur](#architektur)
3. [Installation](#installation)
4. [Entwicklung](#entwicklung)
5. [Build für Production](#build-für-production)
6. [Konfiguration](#konfiguration)
7. [Sicherheitsfunktionen](#sicherheitsfunktionen)
8. [Troubleshooting](#troubleshooting)

## Überblick

Die Desktop-App basiert auf Tauri 2 und kombiniert die SvelteKit-Oberfläche mit einem Rust-Backend für sichere Token- und Schlüsselverwaltung.

### Hauptfunktionen

- **Multi-Server-Unterstützung**: Verbindung zu xelanote.com oder Self-Hosted Instanzen
- **OS Keyring Integration**: Sichere Token-Speicherung über das Betriebssystem
- **In-Memory KEK Storage**: Verschlüsselungsschlüssel nur im Rust-Memory
- **Encrypted Fallback**: AES-256-GCM-verschlüsselte Dateien, wenn Keyring nicht verfügbar
- **Native Performance**: Schnellere Startzeiten und geringerer Ressourcenverbrauch

### Bundle-Formate

- **Linux**: `.deb` (Debian/Ubuntu) und `.AppImage` (Universal)
- **Weitere Plattformen**: Windows und macOS werden noch nicht unterstützt

## Architektur

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (SvelteKit)                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │   config.ts  │  │   api.ts     │  │   Settings UI    │  │
│  │  IS_TAURI    │  │ dynamic URL  │  │  Connection tab  │  │
│  └──────┬───────┘  └──────┬───────┘  └────────┬─────────┘  │
│         │                 │                    │            │
│         └─────────────────┴────────────────────┘            │
│                           │                                 │
│                    Tauri IPC (invoke)                       │
│                           │                                 │
├───────────────────────────┼─────────────────────────────────┤
│                    Rust Backend                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  main.rs                                            │   │
│  │  - Tauri entry point                                │   │
│  │  - KekManager state                                 │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                 │
│  ┌────────────┬──────────┴──────────┬────────────────┐     │
│  │ commands.rs│     kek.rs          │   keyring.rs   │     │
│  │            │                     │                │     │
│  │ IPC cmds:  │ KEK Manager:        │ Token Storage: │     │
│  │ - store_*  │ - Mutex<Vec<u8>>    │ - OS keyring   │     │
│  │ - load_*   │ - zeroize on lock   │ - AES fallback │     │
│  │ - delete_* │ - XSS protection    │ - per-server   │     │
│  │ - lock_*   │                     │                │     │
│  └────────────┴─────────────────────┴────────────────┘     │
│                           │                                 │
│  ┌────────────────────────┼──────────────────────────┐     │
│  │         OS Integration                            │     │
│  │  - Linux: Secret Service / kwallet                │     │
│  │  - Fallback: ~/.local/share/xelanote/*.enc        │     │
│  └───────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘

Data Flow (Token Storage):
Frontend → invoke('store_auth_tokens') → commands.rs → keyring.rs
  → Try OS Keyring → Fallback: AES-256-GCM encrypted file

Data Flow (KEK Storage):
Frontend → invoke('store_kek') → commands.rs → kek.rs
  → Store in Rust Mutex<Vec<u8>> (in-memory only)
```

### Komponenten

#### Rust Backend (`frontend/src-tauri/src/`)

- **main.rs**: Tauri-Einstiegspunkt, registriert Commands und KekManager-State
- **kek.rs**: Sichere In-Memory-KEK-Speicherung mit `zeroize` für automatisches Löschen
- **keyring.rs**: Token-Verwaltung mit OS-Keyring (Fallback: AES-256-GCM)
- **commands.rs**: IPC-Commands für Frontend-Backend-Kommunikation

#### TypeScript Integration

- **src/lib/config.ts**: `IS_TAURI` Detection, Server-URL-Management
- **src/lib/api.ts**: Dynamische API-URL per Request
- **src/lib/stores/auth.svelte.ts**: Token-Persistierung via Tauri
- **src/lib/stores/encryption.svelte.ts**: KEK-Storage in Rust Memory
- **src/routes/settings/+page.svelte**: Connection-Tab für Server-Konfiguration

## Installation

### Prerequisites (Linux)

#### Arch Linux / Manjaro

```bash
sudo pacman -S webkit2gtk-4.1 libsoup3 base-devel
```

#### Ubuntu / Debian

```bash
sudo apt update
sudo apt install libwebkit2gtk-4.1-dev libsoup-3.0-dev \
  build-essential curl wget file libssl-dev \
  libgtk-3-dev librsvg2-dev
```

#### Fedora

```bash
sudo dnf install webkit2gtk4.1-devel libsoup3-devel \
  openssl-devel gtk3-devel librsvg2-devel
```

### Rust Installation

Wenn Rust noch nicht installiert ist:

```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source $HOME/.cargo/env
```

### App installieren

#### Option 1: Von Release (wenn verfügbar)

```bash
# .deb installieren
sudo dpkg -i xelanote_0.1.0_amd64.deb
sudo apt-get install -f  # Falls Dependencies fehlen

# .AppImage nutzen
chmod +x xelanote_0.1.0_amd64.AppImage
./xelanote_0.1.0_amd64.AppImage
```

#### Option 2: Selbst bauen

```bash
cd frontend
npm run tauri:build

# Bundles finden unter:
# src-tauri/target/release/bundle/deb/xelanote_0.1.0_amd64.deb
# src-tauri/target/release/bundle/appimage/xelanote_0.1.0_amd64.AppImage
```

## Entwicklung

### Entwicklungsumgebung starten

```bash
cd frontend

# Standard (funktioniert meist)
npm run tauri:dev

# Bei Wayland-Problemen (blank screen, rendering issues)
WEBKIT_DISABLE_COMPOSITING_MODE=1 GDK_BACKEND=x11 npm run tauri:dev
```

Die App startet automatisch:
- Frontend Dev Server auf Port 5173
- Tauri-Fenster verbindet sich mit `http://localhost:5173`

### Entwicklungs-Workflow

1. **Frontend-Änderungen**: Hot-Reload funktioniert automatisch
2. **Rust-Änderungen**: Tauri kompiliert neu und startet App neu
3. **Config-Änderungen** (`tauri.conf.json`): App neu starten

### Debug-Tools

Tauri läuft mit aktivierten DevTools (nur in Dev-Builds):

```rust
// In Cargo.toml
tauri = { version = "2", features = ["devtools"] }
```

Im App-Fenster: Rechtsklick → "Inspect Element" oder `Ctrl+Shift+I`

### Projekt-Struktur

```
frontend/
├── src-tauri/
│   ├── src/
│   │   ├── main.rs           # Entry point
│   │   ├── commands.rs       # IPC commands
│   │   ├── kek.rs            # KEK management
│   │   └── keyring.rs        # Token storage
│   ├── icons/                # App icons
│   ├── Cargo.toml            # Rust dependencies
│   ├── tauri.conf.json       # Tauri config
│   └── build.rs              # Build script
├── src/
│   ├── lib/
│   │   ├── config.ts         # IS_TAURI, server URLs
│   │   ├── api.ts            # API client
│   │   └── stores/           # Svelte stores mit Tauri-Integration
│   └── routes/
│       └── settings/         # Settings UI mit Connection tab
└── package.json              # npm scripts
```

## Build für Production

### Release Build erstellen

```bash
cd frontend
npm run tauri:build
```

Build-Output:
```
src-tauri/target/release/bundle/
├── deb/
│   └── xelanote_0.1.0_amd64.deb       # Debian-Paket
└── appimage/
    └── xelanote_0.1.0_amd64.AppImage  # Universal Linux
```

### Build-Optimierungen

Das Release-Profil in `Cargo.toml` nutzt aggressive Optimierungen:

```toml
[profile.release]
panic = "abort"       # Kleinere Binary
codegen-units = 1     # Bessere Optimierung
lto = true            # Link-Time Optimization
opt-level = "s"       # Optimize for size
strip = true          # Strip debug symbols
```

Typische Binary-Größen:
- **Rust Binary**: ~15-20 MB (nach strip)
- **.deb Package**: ~20-25 MB
- **.AppImage**: ~25-30 MB

### Custom Icon

Icons anpassen:

```bash
# PNG/ICO/ICNS generieren aus einem Source-Image
npm run tauri:icon -- path/to/source.png

# Icons werden generiert in:
# src-tauri/icons/
```

## Konfiguration

### Server-URL einstellen

Die Desktop-App kann mit beliebigen xelanote-Servern verbunden werden:

#### Via Settings UI (empfohlen)

1. App öffnen
2. Settings → Connection (nur in Tauri sichtbar)
3. Server URL eingeben (z.B. `https://<STAGING_URL>`)
4. "Save" klicken
5. App neu starten (noch kein automatischer Reconnect)

#### Via localStorage (für Testing)

```javascript
// In Browser DevTools Console (Ctrl+Shift+I)
localStorage.setItem('xelanote_server_url', 'https://<STAGING_URL>');
location.reload();
```

### Default Server

Wenn keine Server-URL gesetzt ist: `https://xelanote.com` (offizieller Server)

### URL-Normalisierung

- Trailing Slashes werden automatisch entfernt
- Schema muss explizit sein (`https://` oder `http://`)
- Keine automatische HTTPS-Umleitung

### Mehrere Server

Aktuell wird nur ein Server gleichzeitig unterstützt. Beim Wechsel:

1. Logout vom aktuellen Server
2. Server-URL in Settings ändern
3. App neu starten
4. Login auf neuem Server

Tokens von verschiedenen Servern werden separat gespeichert und gehen nicht verloren.

## Sicherheitsfunktionen

### OS Keyring Integration

**Was ist das?**
Der OS Keyring ist ein sicherer Passwort-Speicher des Betriebssystems.

**Linux-Backends:**
- **GNOME**: GNOME Keyring (Secret Service)
- **KDE**: KWallet
- **Fallback**: AES-verschlüsselte Datei (siehe unten)

**Vorteile:**
- Tokens sind nicht im Klartext
- Betriebssystem-Level-Verschlüsselung
- Integration mit Benutzeranmeldung
- Separater Storage pro Server

**Implementierung:**

```rust
// keyring.rs
use keyring::Entry;

let entry = Entry::new("xelanote", "tokens_xelanote_com")?;
entry.set_password(&json)?;  // Store
let json = entry.get_password()?;  // Retrieve
```

### Encrypted Fallback Storage

Wenn OS Keyring nicht verfügbar ist (z.B. Headless-Systeme), nutzt die App einen verschlüsselten Fallback:

**Mechanismus:**
- **Algorithmus**: AES-256-GCM (authentifizierte Verschlüsselung)
- **Schlüsselableitung**: SHA-256 von Machine-ID + Username
- **Speicherort**: `~/.local/share/xelanote/<server>.tokens.enc`

**Format:**
```
[12 bytes Nonce][Variable-length Ciphertext + Auth Tag]
```

**Sicherheitseigenschaften:**
- Tokens sind nur auf derselben Maschine entschlüsselbar
- Gleicher Benutzer auf anderer Maschine kann Tokens nicht lesen
- Authenticated Encryption verhindert Manipulation

**Code:**

```rust
// Schlüssel ableiten
fn derive_fallback_key() -> [u8; 32] {
    let machine_id = fs::read_to_string("/etc/machine-id")?;
    let username = env::var("USER")?;

    let mut hasher = Sha256::new();
    hasher.update(machine_id.trim());
    hasher.update(username);
    hasher.update(b"xelanote-token-encryption");
    hasher.finalize().into()
}

// Verschlüsseln
let cipher = Aes256Gcm::new_from_slice(&key)?;
let nonce = generate_random_nonce();
let ciphertext = cipher.encrypt(nonce, json.as_bytes())?;
```

### KEK (Key Encryption Key) Storage

Der KEK (master encryption key) wird **nur im Rust-Memory** gespeichert, nie auf Disk.

**Architektur:**

```rust
// kek.rs
pub struct KekManager {
    kek: Mutex<Option<Vec<u8>>>,  // In-Memory only
}

impl KekManager {
    pub fn lock(&self) -> Result<(), String> {
        let mut guard = self.kek.lock()?;
        if let Some(mut kek) = guard.take() {
            kek.zeroize();  // Secure memory clearing
        }
        Ok(())
    }
}
```

**Sicherheit:**

1. **XSS-Protection**: JavaScript kann KEK nicht direkt lesen (nur via IPC)
2. **Auto-Lock**: TypeScript Timer locked KEK nach Inaktivität
3. **Secure Clearing**: `zeroize` überschreibt Memory mit Nullen
4. **No Disk Write**: KEK wird nie auf Disk geschrieben

**Lifecycle:**

```
User enters password
  → Frontend derives KEK (via argon2id)
    → invoke('store_kek', kek)
      → Rust stores in Mutex<Vec<u8>>
        → Frontend uses KEK for encryption
          → After timeout: invoke('lock_kek')
            → Rust zeroizes memory
```

### IPC Commands

Alle Tauri-Commands für Frontend-Backend-Kommunikation:

```rust
// Token Management
store_auth_tokens(server_url: String, tokens: AuthTokens)
load_auth_tokens(server_url: String) -> Option<AuthTokens>
delete_auth_tokens(server_url: String)

// KEK Management
store_kek(kek: Vec<u8>)
get_kek() -> Option<Vec<u8>>
lock_kek()
is_kek_locked() -> bool
```

### Content Security Policy

Die App nutzt eine strikte CSP in `tauri.conf.json`:

```json
{
  "security": {
    "csp": "default-src 'self'; connect-src 'self' https: wss:; script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; font-src 'self' data:;"
  }
}
```

**Erlaubt:**
- HTTPS/WSS connections zu allen Servern (für Multi-Server-Support)
- WebAssembly (für libsodium crypto)
- Inline styles (für SvelteKit)
- Data URIs für Icons

**Blockiert:**
- Eval/Function constructor
- Externe Scripts
- HTTP mixed content

## Troubleshooting

### Blank Screen / Rendering Issues (Wayland)

**Problem**: App startet, aber zeigt nur weißes Fenster.

**Ursache**: WebKit hat Probleme mit Wayland-Compositing auf manchen Systemen.

**Lösung**:

```bash
# Force X11 Backend
GDK_BACKEND=x11 npm run tauri:dev

# Disable Compositing
WEBKIT_DISABLE_COMPOSITING_MODE=1 npm run tauri:dev

# Beide kombinieren (meist beste Option)
WEBKIT_DISABLE_COMPOSITING_MODE=1 GDK_BACKEND=x11 npm run tauri:dev
```

**Permanent fix** (in Script einbauen):

```json
// package.json
{
  "scripts": {
    "tauri:dev": "WEBKIT_DISABLE_COMPOSITING_MODE=1 GDK_BACKEND=x11 tauri dev",
    "tauri:build": "tauri build"
  }
}
```

### Keyring Access Failed

**Problem**: "Failed to store tokens" Fehler.

**Diagnose**:

```bash
# Check ob Secret Service läuft (GNOME/freedesktop)
ps aux | grep gnome-keyring

# Check KWallet (KDE)
ps aux | grep kwalletd

# Manuell starten (GNOME)
gnome-keyring-daemon --start
```

**Fallback**: Wenn Keyring nicht verfügbar, nutzt die App automatisch verschlüsselte Dateien unter `~/.local/share/xelanote/`.

**Berechtigungen prüfen**:

```bash
ls -la ~/.local/share/xelanote/
# Sollte nur für User lesbar sein (600 oder 700)
```

### Build Errors

#### "webkit2gtk not found"

```bash
# Arch
sudo pacman -S webkit2gtk-4.1

# Ubuntu
sudo apt install libwebkit2gtk-4.1-dev

# Fedora
sudo dnf install webkit2gtk4.1-devel
```

#### "error: linker 'cc' not found"

```bash
# Arch
sudo pacman -S base-devel

# Ubuntu
sudo apt install build-essential

# Fedora
sudo dnf groupinstall "Development Tools"
```

#### "libsoup-3.0 not found"

```bash
# Arch
sudo pacman -S libsoup3

# Ubuntu
sudo apt install libsoup-3.0-dev

# Fedora
sudo dnf install libsoup3-devel
```

### Connection Issues

**Problem**: "Failed to connect to server" beim Start.

**Diagnose**:

1. Server-URL prüfen in Settings → Connection
2. Server erreichbar? `curl https://your-server.com/health`
3. CORS richtig konfiguriert auf Server?

**Server-CORS für Tauri**:

```bash
# Auf Server: CORS muss 'tauri://localhost' erlauben
export CORS_ALLOWED_ORIGINS="https://xelanote.com,tauri://localhost"
```

**Note**: Tauri nutzt `tauri://localhost` als Origin, nicht `http://localhost`.

### Performance Issues

**Problem**: App ist langsam oder verbraucht viel RAM.

**Debug**:

1. DevTools öffnen: Rechtsklick → Inspect Element
2. Performance Tab → Record während langsamer Operation
3. Memory Tab → Take Heap Snapshot

**Mögliche Ursachen**:
- Zu viele Notizen geladen (Virtualisierung prüfen)
- Memory Leak im Frontend
- Große verschlüsselte Blobs

**Temporäre Lösung**:

```bash
# App neu starten
killall xelanote
```

### Dev Server Port Conflict

**Problem**: "Port 5173 already in use"

**Lösung 1** (anderen Port nutzen):

```bash
# In tauri.conf.json
"devUrl": "http://localhost:5175"

# Frontend mit custom port starten
npm run dev -- --port 5175
```

**Lösung 2** (Port freigeben):

```bash
# Finde Prozess
lsof -i :5173

# Kill process
kill -9 <PID>
```

### App startet nicht nach Build

**Problem**: `.AppImage` oder `.deb` startet nicht.

**Diagnose**:

```bash
# AppImage
./xelanote_0.1.0_amd64.AppImage --verbose

# .deb (nach Installation)
xelanote --verbose

# Logs prüfen
journalctl --user -xe | grep xelanote
```

**Häufige Ursachen**:
- Fehlende Dependencies (siehe Prerequisites)
- Fehlende Berechtigungen (chmod +x für AppImage)
- SELinux/AppArmor blockiert (Fedora/Ubuntu Server)

### Encryption Issues

**Problem**: "Failed to unlock" oder "KEK not found".

**Ursachen**:
1. KEK wurde gelocked (Auto-Lock nach Timeout)
2. App wurde neu gestartet (KEK ist In-Memory only)
3. Password falsch eingegeben

**Lösung**:
- Einfach neues Login mit Passwort → KEK wird neu abgeleitet

**KEK-Status prüfen** (in DevTools Console):

```javascript
await window.__TAURI__.invoke('is_kek_locked')
// true = locked, false = unlocked
```

---

## Weiterführende Dokumentation

- **E2E Encryption**: `docs/encryption-v2.md`
- **API-Integration**: `docs/api.md`
- **Deployment**: `docs/deployment.md`
- **Development**: `docs/development.md`
- **Security Audit**: `docs/security_audit_findings.md`

## Roadmap

Geplante Features für die Desktop-App:

- [ ] Auto-Update Mechanismus (Tauri Updater)
- [ ] Windows Build (.exe, .msi)
- [ ] macOS Build (.dmg, .app)
- [ ] System Tray Integration
- [ ] Desktop Notifications
- [ ] Offline Mode mit lokaler SQLite
- [ ] Multi-Server-Switching ohne Restart
- [ ] Hardware Security Module (HSM) Support
- [ ] Touch ID / Windows Hello Integration

## Support

Bei Problemen:

1. **Logs prüfen**: DevTools Console oder `journalctl`
2. **Issue erstellen**: https://github.com/xela-io/xelanote/issues
3. **Entwicklungsdokumentation**: `docs/development.md`
