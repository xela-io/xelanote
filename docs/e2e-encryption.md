# Ende-zu-Ende-Verschluesselung (E2E Encryption)

## Status

Diese Dokumentation beschreibt den aktuell implementierten Stand in diesem Repository (Stand: 2026-02-28).

## Sicherheitsmodell in Kurzform

- Notizinhalte werden clientseitig verschluesselt, bevor sie an den Server gesendet werden.
- Der Schluessel (KEK) wird aus dem Benutzerpasswort und einem servergespeicherten Salt lokal abgeleitet.
- Pro Notiz wird ein eigener DEK erzeugt; der DEK wird mit dem KEK gewrappt.
- Das System verwendet Argon2id + XChaCha20-Poly1305.
- Aktuelle Verschluesselungsversion ist v3 (AAD-Binding an `note_id`); v2 bleibt fuer Alt-Daten lesbar.

## Kryptografie und Formate

### KDF

- Algorithmus: Argon2id
- Parameter: interactive preset (`opslimit=3`, `memlimit=64MB`)
- Salt: 16 Byte pro Benutzer

### AEAD

- Algorithmus: XChaCha20-Poly1305
- Nonce: 24 Byte (zufaellig)
- Content und gewrappter DEK werden AEAD-geschuetzt gespeichert
- v3 bindet AAD an Notizkontext (`note_id`, Purpose, Material)

### Metadatenstruktur (Payload)

Die verschluesselte Payload enthaelt:

- `ciphertext` (Base64)
- `metadata`:
  - `version` (2 oder 3)
  - `algorithm`
  - `kdf`
  - `kdf_strength`
  - `nonce_bytes`
  - `wrapped_dek`

## Was ist durch E2EE geschuetzt?

### Geschuetzt

- Notizinhalt (`encrypted_content`)
- Optional Notiztitel (`encrypted_title`)
- Uploads/Anhaenge aus verschluesselten Notizen (clientseitig verschluesselt als `.xenc`)

### Nicht durch E2EE geschuetzt (serverseitig sichtbar)

- Ordnerpfade
- Strukturmetadaten (z. B. Timestamps)
- Tags
- Keywords (wenn aktiviert)
- Uploads/Anhaenge in unverschluesselten Notizen
- Upload-Metadaten (z. B. Upload-Zeitpunkt, Groesse, user-scoped Storage-Pfad)

## AI-Grenzen

- Serverseitige AI-Verarbeitung fuer verschluesselte Notizen ist deaktiviert (Summary, Tag/Link-Vorschlaege, Formatieren, Transform).
- Bei unverschluesselten Notizen koennen AI-Features Klartext an Backend/Provider uebertragen.
- E2EE gilt daher weiterhin nur fuer verschluesselte Payload-Bereiche.

## Recovery-Key und Passwortverlust

Wichtig:

- Recovery-Key-Management (Setzen/Salt) ist vorhanden.
- Der Recovery-Reset fuer Konten mit verschluesselten Notizen ist derzeit absichtlich blockiert.
- Grund: Ein sicherer Recovery-basierter DEK-Rewrap-Flow ist noch nicht implementiert.

Konsequenz:

- Wenn du dein Passwort verlierst, sind bestehende verschluesselte Notizen aktuell nicht wiederherstellbar.

## Passwortaenderung

- Passwortaenderung fuer Benutzer mit verschluesselten Notizen nutzt DEK-Rewrap.
- Dabei werden vorhandene gewrappte DEKs clientseitig mit altem KEK entpackt und mit neuem KEK neu gewrappt.
- Der Server validiert Vollstaendigkeit der Rewrap-Payloads vor Persistierung.

## Sharing und E2EE

- Verschluesselte Notizen koennen nicht direkt geteilt werden.
- Vor Sharing muessen Notizen entschluesselt werden.
- Wird eine geteilte Notiz wieder verschluesselt, werden Shares entfernt.

## Praktische Empfehlungen

1. Starkes, einzigartiges Passwort verwenden.
2. Titelverschluesselung aktivieren, wenn Titel-Metadaten relevant sind.
3. Keywords deaktiviert lassen, wenn Metadaten-Leakage minimiert werden soll.
4. Sensible Infos nicht in Ordnernamen/Tags ablegen.
5. Bei Uploads beachten: Nur Uploads aus verschluesselten Notizen sind E2EE (`.xenc`).
6. AI bei unverschluesselten Notizen nur nutzen, wenn Klartextuebertragung fuer den konkreten Inhalt akzeptabel ist.

## FAQ

**Kann ein Admin meinen verschluesselten Notizinhalt lesen?**  
Nicht ohne Passwort/Schluesselmaterial. Sichtbar bleiben jedoch nicht-E2EE-Metadaten.

**Ist das Zero-Knowledge fuer alles?**  
Nein. Zero-Knowledge gilt nur fuer geschuetzte Payload-Bereiche, nicht fuer alle Metadaten. Uploads aus verschluesselten Notizen sind verschluesselt; Upload-Metadaten bleiben sichtbar.

**Kann ich mit Recovery-Key verschluesselte Notizen nach Passwortverlust wiederherstellen?**  
Aktuell nein.

**Sind verschluesselte Notizen durchsuchbar?**  
Ja, clientseitig. Optional koennen Keywords serverseitig im Klartext gespeichert werden (Trade-off).

**Kann ich verschluesselte Notizen teilen?**  
Nein, nicht direkt.
