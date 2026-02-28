# E2E-Verschluesselung - Quick Start Guide

## Schnellstart

### 1. Anmelden

- Melde dich normal an.
- Beim Login wird der Schluessel aus Passwort + Salt lokal abgeleitet (Argon2id).
- Nach erfolgreichem Login ist die Verschluesselung entsperrt.

### 2. Notiz erstellen

- Neue Notiz erstellen und speichern.
- Der Notizinhalt wird clientseitig verschluesselt gespeichert.
- Optional kannst du in den Einstellungen auch Titel verschluesseln.

### 3. Bestehende Notizen migrieren (optional)

- Gehe zu `Einstellungen -> Migration`.
- Verschluessle alte Klartext-Notizen mit dem Migrations-Flow.

## Was ist durch E2EE geschuetzt?

### Geschuetzt

- Notizinhalt (`encrypted_content`)
- Optional: Notiztitel (`encrypted_title`)
- Uploads/Anhaenge aus verschluesselten Notizen (`.xenc`, clientseitig verschluesselt)

### Nicht durch E2EE geschuetzt

- Ordnerpfade und strukturelle Metadaten (z. B. Zeitstempel)
- Tags
- Keywords (wenn die Option aktiv ist)
- Upload-Dateien/Anhaenge in unverschluesselten Notizen
- Upload-Metadaten (z. B. Zeitpunkt, Groesse, Storage-Pfad)

## Wichtige Grenzen (Stand: 2026-02-28)

- Recovery-Reset zur Entschluesselung bereits verschluesselter Notizen ist aktuell **nicht verfuegbar**.
- Wenn du dein Passwort verlierst, sind bestehende verschluesselte Notizen derzeit nicht wiederherstellbar.
- AI-Features koennen (je nach Aktion) Klartext an Backend/Provider senden.
- AI-Zusammenfassung ist fuer verschluesselte Notizen derzeit deaktiviert.

## Empfehlungen

1. Nutze ein starkes, einzigartiges Passwort (am besten aus Passwortmanager).
2. Aktiviere Titelverschluesselung, wenn du Titel-Metadaten minimieren willst.
3. Lass Keyword-Extraktion deaktiviert, wenn du Metadaten-Leakage reduzieren willst.
4. Verwende keine sensiblen Informationen in Ordnernamen/Tags.
5. Beruecksichtige: Uploads aus verschluesselten Notizen sind E2EE, in Klartext-Notizen nicht.

## FAQ

**F: Kann ein Server-Admin meinen verschluesselten Notizinhalt lesen?**  
A: Ohne Passwort/Keymaterial nicht. Sichtbar bleiben aber nicht-E2EE-Metadaten.

**F: Kann ich verschluesselte Notizen teilen?**  
A: Nein. Verschluesselte Notizen muessen vor dem Teilen entschluesselt werden.

**F: Hilft der Recovery Key bei Passwortverlust fuer verschluesselte Notizen?**  
A: Aktuell nein. Der Recovery-Reset fuer verschluesselte Notizen ist blockiert, bis DEK-Rewrap im Recovery-Flow umgesetzt ist.

Weitere Details: [E2E-Verschluesselung (vollstaendig)](./e2e-encryption.md)
