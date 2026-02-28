# Ende-zu-Ende-Verschlüsselung (E2E)

## Prinzip: Zero-Knowledge

Der Server sieht **niemals den Klartext** verschlüsselter Notizen. Alle Verschlüsselung passiert im Browser (Client-Side). Der Server speichert nur verschlüsselte Blobs.

## Schlüssel-Hierarchie

```
User-Passwort
    ↓ Argon2id (Key Derivation)
KEK (Key Encryption Key)
    ↓ Verschlüsselt jeden...
DEK (Data Encryption Key)     ← Zufällig pro Notiz
    ↓ Verschlüsselt...
Notiz-Inhalt + optional Titel
```

### Warum zwei Schlüssel-Ebenen?

Wenn du dein Passwort änderst, muss **nur der KEK** neu abgeleitet werden. Die DEKs (einer pro Notiz) werden dann mit dem neuen KEK neu verschlüsselt (`batch-reencrypt-deks`). Der Notiz-Inhalt selbst muss nie neu verschlüsselt werden.

## Algorithmen

| Komponente | Algorithmus |
|-----------|-------------|
| Key Derivation | **Argon2id** (Passwort → KEK) |
| Notiz-Verschlüsselung | **XChaCha20-Poly1305** (libsodium) |
| DEK-Wrapping | XChaCha20-Poly1305 (KEK verschlüsselt DEK) |
| API-Key-Verschlüsselung | **AES-256-GCM** (serverseitig) |

**XChaCha20-Poly1305** wurde wegen seiner Nonce-Sicherheit gewählt — mit 192-Bit Nonce ist die Wahrscheinlichkeit einer Nonce-Kollision selbst bei Millionen von Verschlüsselungen vernachlässigbar.

## Verschlüsselungs-Flow

### Notiz verschlüsseln (beim Speichern)

```
1. [Browser] DEK generieren (32 Bytes random)
              ↓
2. [Browser] Notiz-Inhalt mit DEK verschlüsseln
             XChaCha20-Poly1305(content, DEK, nonce)
              ↓
3. [Browser] DEK mit KEK verschlüsseln ("wrappen")
             XChaCha20-Poly1305(DEK, KEK, nonce)
              ↓
4. [Browser] An Server senden:
             {
               encrypted_content: <blob>,
               wrapped_dek: <base64>,
               encryption_metadata: {algo, nonce, version},
               content_encrypted: true
             }
              ↓
5. [Server]  Speichert alles in SQLite
             (Server kann nichts entschlüsseln!)
```

### Notiz entschlüsseln (beim Laden)

```
1. [Server]  GET /api/notes/{id}
             → {encrypted_content, wrapped_dek, encryption_metadata}
              ↓
2. [Browser] KEK aus Speicher holen (IndexedDB oder Passwort-Eingabe)
              ↓
3. [Browser] DEK entschlüsseln (un-wrappen):
             DEK = decrypt(wrapped_dek, KEK)
              ↓
4. [Browser] Inhalt entschlüsseln:
             content = decrypt(encrypted_content, DEK)
              ↓
5. [Browser] Klartext im Editor anzeigen
```

## Security Levels

Der User kann zwischen drei Sicherheitsstufen wählen:

| Level | KEK-Speicherung | Verhalten |
|-------|-----------------|-----------|
| **Convenient** | IndexedDB (persistent) | KEK bleibt auch nach Browser-Schließen |
| **Balanced** | IndexedDB (Session) | KEK wird bei Tab-Schließen gelöscht |
| **Paranoid** | Nur im RAM | KEK nie persistiert, immer Passwort-Eingabe |

### Auto-Lock

Im "Balanced"- und "Paranoid"-Modus gibt es einen Auto-Lock-Timer:
- Nach X Minuten Inaktivität wird der KEK gelöscht
- User muss Passwort erneut eingeben
- Konfigurierbar in den Einstellungen

## Verschlüsselte Notizen und Features

### Wikilinks bei verschlüsselten Notizen

**Problem:** Server kennt den Inhalt nicht → kann `[[Links]]` nicht parsen.

**Lösung:**
1. Client entschlüsselt Notiz
2. Client extrahiert `[[...]]` Titel
3. Client sendet Titel-Liste an Server
4. Server löst Titel → Note-IDs auf
5. Server speichert in `links`-Tabelle

### Suche bei verschlüsselten Notizen

**Serverseitig:** FTS5 kann verschlüsselte Inhalte nicht durchsuchen.

**Serverseitige Keyword-Indexierung fuer verschluesselte Notizen:** ist deaktiviert/deprecated.

**Client-seitig:** Fuse.js durchsucht entschluesselte Notizen lokal im Browser (fuer Offline und verschluesselte Suche).

### KI bei verschlüsselten Notizen

Serverseitige KI-Features (Zusammenfassung, Tags, Links, Format, Transform) sind fuer verschluesselte Notizen blockiert.

**Hinweis:** Bei unverschluesselten Notizen kann Klartext an Backend/LLM-Provider gehen.

## Passwort-Änderung

```
1. User gibt altes + neues Passwort ein
2. Client: KEK mit altem Passwort ableiten → alte DEKs entschlüsseln
3. Client: KEK mit neuem Passwort ableiten
4. Client: Alle DEKs mit neuem KEK neu verschlüsseln
5. Client: POST /api/notes/batch-reencrypt-deks
           [{noteId, newWrappedDEK}, ...]
6. Server: Speichert neue wrapped_deks
```

Der **Notiz-Inhalt selbst wird nie neu verschlüsselt** — nur die DEK-Wrapper.

## DB-Verschlüsselung (SQLCipher)

Optional (nur lokale Builds) kann die gesamte SQLite-Datei mit **SQLCipher** verschlüsselt werden:

- Build-Flag: `-tags "fts5 sqlite_crypt"`
- Umgebungsvariable: `SQLITE_CIPHER_KEY`
- Nicht in Docker-Production (kein SQLCipher-Package im Alpine-Image)
- Schützt die DB-Datei at-rest (z.B. auf einer gestohlenen Festplatte)

## Nächste Seiten

- [Authentifizierung-und-Sicherheit](Authentifizierung-und-Sicherheit.md) — Auth und Token-System
- [KI-Integration](KI-Integration.md) — Wie KI mit verschlüsselten Notizen funktioniert
