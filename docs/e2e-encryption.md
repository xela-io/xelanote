# Ende-zu-Ende-Verschlüsselung (E2E Encryption)

## Übersicht

xelanote bietet vollständige Ende-zu-Ende-Verschlüsselung für deine Notizen. Das bedeutet, dass nur du deine Notizen lesen kannst – nicht einmal der Server oder die Administratoren haben Zugriff auf den Inhalt deiner verschlüsselten Notizen.

## 🔒 Was ist Ende-zu-Ende-Verschlüsselung?

Bei Ende-zu-Ende-Verschlüsselung (E2E) werden deine Notizen direkt auf deinem Gerät verschlüsselt, bevor sie an den Server gesendet werden. Der Server speichert nur den verschlüsselten Text – er hat niemals Zugriff auf den Schlüssel, der zum Entschlüsseln benötigt wird.

### Zero-Knowledge Architektur

xelanote verwendet eine "Zero-Knowledge" Architektur:
- Der Verschlüsselungsschlüssel wird aus deinem Passwort abgeleitet
- Dies geschieht nur auf deinem Gerät in deinem Browser
- Der Schlüssel wird niemals an den Server gesendet
- Der Server speichert nur verschlüsselte Daten
- Selbst bei einem Server-Hack bleiben deine Daten geschützt

## 🔑 Technische Details

### Verwendete Kryptografie (v2)

xelanote verwendet modernste Verschlüsselungsstandards (seit v2 / 2026-01-20):

- **Bibliothek:** libsodium.js
  - Battle-tested seit 2013 (Signal, WireGuard, etc.)
  - IETF-standardisierte Algorithmen
  - WASM-Performance-optimiert

- **Key Derivation Function (KDF):** Argon2id
  - Memory-hard (64MB RAM pro Ableitung)
  - GPU-resistent
  - ~600ms - 2s Berechnungszeit (Web Worker, non-blocking)
  - Schützt vor Brute-Force-Angriffen

- **Verschlüsselung:** XChaCha20-Poly1305-IETF
  - 256-bit Schlüssel
  - Authenticated Encryption with Associated Data (AEAD)
  - 24-byte Nonces (keine Kollisionsgefahr)
  - IETF RFC 8439 Standard
  - Verhindert Manipulationen durch Poly1305 Auth-Tag

- **Schlüssel-Wrapping:** XChaCha20-Poly1305-IETF
  - Jede Notiz hat einen eigenen Verschlüsselungsschlüssel (DEK)
  - DEKs werden mit deinem Master-Key (KEK) verschlüsselt
  - Ermöglicht einfache Schlüsselrotation

- **Salt:** 16 Byte zufällig pro Benutzer
  - Verhindert Rainbow-Table-Angriffe
  - Wird sicher auf dem Server gespeichert

- **Password Normalization:** Unicode NFC
  - Verhindert Key-Inkonsistenzen zwischen verschiedenen Geräten
  - Automatisch bei Anmeldung

### Pro-Notiz-Verschlüsselung

Jede Notiz wird mit einem eigenen, zufällig generierten Schlüssel (DEK) verschlüsselt. Dieser DEK wird dann mit deinem Master-Key (KEK) verschlüsselt. Das bietet mehrere Vorteile:
- Schnelle Schlüsselrotation ohne alle Notizen neu verschlüsseln zu müssen
- Isolation zwischen Notizen
- Bessere Sicherheit bei kompromittierten einzelnen Notizen

## 🚀 Erste Schritte

### Automatische Aktivierung

Die Verschlüsselung wird automatisch aktiviert, wenn du dich anmeldest:

1. **Registrierung oder Anmeldung**
   - Bei der Registrierung wird automatisch ein Verschlüsselungssalz erstellt
   - Bei der Anmeldung wird aus deinem Passwort und dem Salt dein Verschlüsselungsschlüssel (KEK) abgeleitet
   - Dieser Prozess dauert ca. 600ms (Argon2id)

2. **Notizen erstellen**
   - Neue Notizen werden standardmaessig automatisch verschluesselt
   - Ausnahme: In Ordnern mit deaktivierter Verschluesselung (Encryption Default = off) werden Notizen im Klartext erstellt
   - Du siehst eine kurze Verzoegerung beim Speichern (1-2ms pro Notiz)
   - Im Browser wird der entschluesselte Text angezeigt

3. **Status prüfen**
   - Gehe zu **Einstellungen → Verschlüsselung**
   - Hier siehst du den Status: "Verschlüsselung aktiv" oder "Verschlüsselung gesperrt"

### Verschlüsselungsstatus

**🟢 Verschlüsselung aktiv**
- Dein Verschlüsselungsschlüssel (KEK) ist im Speicher verfügbar
- Du kannst Notizen erstellen, bearbeiten und lesen
- Alle Speichervorgänge werden automatisch verschlüsselt

**🟡 Verschlüsselung gesperrt**
- Du bist nicht angemeldet
- Der Verschlüsselungsschlüssel ist nicht verfügbar
- Du musst dich anmelden, um auf verschlüsselte Notizen zuzugreifen

## ⚙️ Einstellungen

### Verschlüsselungseinstellungen öffnen

Navigiere zu: **Einstellungen → Verschlüsselung**

### Verfügbare Optionen

#### 1. Titel verschlüsseln

**Standard:** Deaktiviert

Wenn aktiviert, werden auch die Titel deiner Notizen verschlüsselt.

**Vorteile:**
- ✅ Maximaler Datenschutz
- ✅ Titel sind nicht auf dem Server sichtbar
- ✅ Schützt auch Metadaten

**Nachteile:**
- ❌ Titel können nicht mehr durchsucht werden
- ❌ Sortierung funktioniert nicht mehr nach Titel
- ❌ Nur Suche nach Ordnern und Keywords (falls aktiviert)

**Wann aktivieren?**
- Wenn du sensible Informationen in Titeln hast
- Wenn du maximale Privatsphäre möchtest
- Wenn du bereit bist, auf Titel-Suche zu verzichten

#### 2. Suchbare Keywords extrahieren

**Standard:** Deaktiviert

**⚠️ SICHERHEITSWARNUNG:** Diese Funktion hat ein Datenleck-Risiko!

Wenn aktiviert, werden häufige Wörter aus deinen Notizen extrahiert und **unverschlüsselt** gespeichert, um die Volltextsuche zu ermöglichen.

**Vorteile:**
- ✅ Ermöglicht Volltextsuche in verschlüsselten Notizen
- ✅ Schnelle Suche über alle Notizen

**Nachteile:**
- ⚠️ Häufige Wörter werden im Klartext gespeichert
- ⚠️ Kann sensible Informationen preisgeben (Namen, Orte, Themen)
- ⚠️ Beispiel: "Bitcoin Investment Strategie" → Keywords: "bitcoin", "investment", "strategie"

**Wann aktivieren?**
- Nur wenn du die Suche unbedingt benötigst
- Nur wenn deine Notizen keine hochsensiblen Daten enthalten
- Nur wenn du das Datenleck-Risiko akzeptierst

**Empfehlung:** Für maximale Sicherheit deaktiviert lassen!

## 🔐 Recovery Key (Wiederherstellungsschlüssel)

### Warum ein Recovery Key?

**WICHTIG:** Wenn du dein Passwort vergisst, sind deine verschlüsselten Notizen unwiederbringlich verloren!

Ein Recovery Key ist deine einzige Möglichkeit, Zugriff auf deine Notizen wiederherzustellen, falls du dein Passwort vergisst.

### Recovery Key erstellen

1. Gehe zu **Einstellungen → Verschlüsselung**
2. Scrolle nach unten und klicke auf **"Recovery Key erstellen"** (oder öffne die RecoveryKeySetup-Komponente)
3. Klicke auf **"Recovery Key generieren"**
4. **WICHTIG:** Speichere den Key sofort:
   - Klicke auf **"Als Textdatei herunterladen"**
   - Oder kopiere ihn in die Zwischenablage
5. Bewahre den Key sicher auf (siehe unten)

### Sicherer Aufbewahrungsort

**✅ Empfohlen:**
- Passwort-Manager (1Password, Bitwarden, KeePass)
- Verschlüsselter USB-Stick im Tresor
- Ausgedruckt in einem Safe
- Verschlüsselte Notiz in einem anderen System

**❌ Nicht empfohlen:**
- Unverschlüsselte Cloud (Dropbox, Google Drive)
- E-Mail an dich selbst
- Unverschlüsselte Textdatei auf dem Computer
- Foto auf dem Smartphone

**🚨 KRITISCH:** Jeder mit deinem Recovery Key kann deine Notizen entschlüsseln!

### Recovery Key verwenden

*(Hinweis: Diese Funktion ist noch nicht implementiert)*

Im Falle eines vergessenen Passworts:
1. Gehe zur Login-Seite
2. Klicke auf "Passwort vergessen?"
3. Gib deinen Recovery Key ein
4. Setze ein neues Passwort
5. Deine Notizen werden mit dem neuen Passwort neu verschlüsselt

## 🔄 Migration (Bestehende Notizen verschlüsseln)

### Übersicht

Wenn du xelanote bereits vor der E2E-Verschlüsselung genutzt hast, sind deine bestehenden Notizen noch im Klartext gespeichert. Mit dem Migrations-Tool kannst du sie verschlüsseln.

### Migrations-Tool öffnen

Navigiere zu: **Einstellungen → Migration**

### Vor der Migration

**✅ Checkliste:**
- [ ] Recovery Key erstellt und sicher aufbewahrt
- [ ] Backup der Notizen erstellt (optional, aber empfohlen)
- [ ] Ausreichend Zeit (Migration kann mehrere Minuten dauern)
- [ ] Stabile Internetverbindung

### Migrations-Prozess

1. **Statistiken prüfen**
   - Das Dashboard zeigt:
     - **Gesamt:** Anzahl aller Notizen
     - **Verschlüsselt:** Bereits verschlüsselte Notizen
     - **Klartext:** Noch zu verschlüsselnde Notizen

2. **Migration starten**
   - Klicke auf **"Migration starten"**
   - Der Fortschrittsbalken zeigt den Status
   - Du siehst: "Migriert X von Y Notizen"

3. **Während der Migration**
   - ✅ Du kannst das Fenster geöffnet lassen
   - ✅ Du kannst in anderen Tabs weiterarbeiten
   - ❌ Schließe die Seite nicht (Migration wird abgebrochen)

4. **Nach der Migration**
   - ✅ Erfolgreiche Migration wird angezeigt
   - ✅ Fehlgeschlagene Notizen werden aufgelistet
   - ✅ Statistiken werden aktualisiert

### Geschwindigkeit

- **Pro Notiz:** ~50-100ms (Verschlüsselung + Netzwerk)
- **50 Notizen:** ~2-5 Sekunden
- **500 Notizen:** ~20-50 Sekunden
- **5000 Notizen:** ~3-8 Minuten

### Fehlerbehandlung

Wenn eine Notiz nicht migriert werden konnte:
1. Die Migration fährt mit den anderen Notizen fort
2. Am Ende siehst du eine Liste der fehlgeschlagenen Notizen
3. Du kannst die Migration erneut starten (bereits migrierte Notizen werden übersprungen)

### Nach der Migration

**✅ Was sich ändert:**
- Alle Notizen sind nun verschlüsselt
- Notizen können nur noch mit deinem Passwort geöffnet werden
- Server-Backups enthalten nur verschlüsselte Daten

**❌ Nicht mehr möglich (ohne Passwort):**
- Notizen auf dem Server lesen
- Server-seitige Suche im Inhalt (außer Keywords, falls aktiviert)
- Notizen wiederherstellen ohne Passwort/Recovery Key

## 🔍 Was wird verschlüsselt?

### ✅ Vollständig geschützt (Zero-Knowledge)

**Immer verschlüsselt:**
- **Notiz-Inhalt:** Der gesamte Text deiner Notiz
- **Verschlüsselungsschlüssel:** Wird aus deinem Passwort abgeleitet, niemals gespeichert

**Optional verschlüsselt:**
- **Notiz-Titel:** Nur wenn du "Titel verschlüsseln" aktivierst
- **Anhänge:** (Zukünftige Funktion)

### ⚠️ Sichtbar für Server/Administratoren

**Immer sichtbar:**
- **Ordner-Struktur:** Pfade wie "/Privat/Projekte/2024"
- **Metadaten:**
  - Erstellungsdatum
  - Änderungsdatum
  - Anzahl der Notizen
  - Größe der Notizen (in Bytes)
- **Benutzer-Informationen:**
  - Username
  - E-Mail-Adresse
  - Anmelde-Zeitpunkte

**Sichtbar wenn aktiviert:**
- **Notiz-Titel:** Wenn "Titel verschlüsseln" nicht aktiviert ist (Standard)
- **Keywords:** Wenn "Keywords extrahieren" aktiviert ist (Standard: deaktiviert)

### Empfehlungen für maximale Privatsphäre

1. **✅ Aktiviere "Titel verschlüsseln"**
   - Schützt auch Metadaten
   - Akzeptiere die eingeschränkte Suche

2. **✅ Deaktiviere "Keywords extrahieren"**
   - Keine Datenlecks
   - Verzichte auf Volltextsuche

3. **✅ Verwende generische Ordnernamen**
   - ❌ Schlecht: "/Privat/Arzttermine/Onkologie"
   - ✅ Gut: "/Privat/Gesundheit"

4. **✅ Vermeide sensible Daten in Metadaten**
   - Ordnernamen
   - Dateinamen (bei Uploads)
   - Tags (falls verwendet)

5. **✅ Erstelle einen Recovery Key**
   - Bewahre ihn offline auf
   - Teile ihn mit niemandem

## 🛡️ Sicherheitshinweise

### Was E2E-Verschlüsselung schützt

**✅ Geschützt vor:**
- Server-Kompromittierung (Admin kann Notizen nicht lesen)
- Datenbank-Diebstahl (Backups enthalten nur verschlüsselte Daten)
- Man-in-the-Middle-Angriffen (mit HTTPS)
- Cloud-Provider-Zugriff (Zero-Knowledge)
- Behördlicher Datenaufforderung (Server hat keinen Schlüssel)

### Was E2E-Verschlüsselung NICHT schützt

**❌ Nicht geschützt vor:**
- **Kompromittiertes Gerät:**
  - Keylogger können dein Passwort abfangen
  - Bildschirmaufnahmen zeigen entschlüsselte Notizen
  - Malware kann Daten aus dem Browser-Speicher lesen

- **Browser-Extensions:**
  - Bösartige Extensions können DOM-Zugriff haben
  - Extensions können entschlüsselte Daten lesen

- **XSS-Angriffe:**
  - Cross-Site-Scripting kann Daten stehlen
  - CSP-Header bieten Schutz, aber nicht 100%

- **Phishing:**
  - Gefälschte Login-Seiten können Passwort stehlen
  - Überprüfe immer die URL

- **Soziale Manipulation:**
  - Jemand könnte dich überreden, dein Passwort zu teilen
  - Teile niemals dein Passwort oder deinen Recovery Key

- **Physischer Zugriff:**
  - Jemand mit Zugriff auf dein angemeldetes Gerät kann Notizen lesen
  - Melde dich immer ab, wenn du fertig bist

### Beste Praktiken

1. **Starkes Passwort verwenden**
   - Mindestens 12 Zeichen
   - Mix aus Buchstaben, Zahlen, Sonderzeichen
   - Nicht in Wörterbüchern zu finden
   - Einzigartig für xelanote

2. **Passwort-Manager nutzen**
   - Generiere ein zufälliges Passwort
   - Speichere es sicher
   - Aktiviere 2FA für den Passwort-Manager

3. **Regelmäßig abmelden**
   - Besonders auf gemeinsam genutzten Geräten
   - Der Verschlüsselungsschlüssel wird beim Abmelden gelöscht

4. **HTTPS verwenden**
   - Überprüfe das Schloss-Symbol in der Adressleiste
   - Niemals über HTTP auf xelanote zugreifen

5. **Gerät sichern**
   - Virenschutz installiert und aktuell
   - Betriebssystem auf dem neuesten Stand
   - Keine verdächtigen Browser-Extensions

6. **Regelmäßige Backups**
   - Exportiere deine Notizen regelmäßig
   - Speichere Backups verschlüsselt
   - Teste die Wiederherstellung

## 🔄 Versionshistorie (Note Versions)

### Übersicht

xelanote speichert automatisch Versionen deiner Notizen, sodass du frühere Zustände wiederherstellen kannst. Dies funktioniert auch vollständig mit verschlüsselten Notizen.

### Was wird in Versionen gespeichert?

Für **verschlüsselte Notizen:**
- `encrypted_content` - Der verschlüsselte Inhalt
- `wrapped_dek` - Der verschlüsselte Data Encryption Key
- `encrypted_title` - Der verschlüsselte Titel (falls aktiviert)
- `encryption_version` - Die Verschlüsselungsversion (1 oder 2)

Für **Klartext-Notizen:**
- `title` - Der Titel
- `content` - Der Inhalt

### Version wiederherstellen

Beim Wiederherstellen einer Version werden 4 Szenarien unterstützt:

| Aktuelle Notiz | Ziel-Version | Verhalten |
|----------------|--------------|-----------|
| Klartext | Klartext | Standard-Restore |
| Verschlüsselt | Verschlüsselt | Encrypted Restore mit DEK-Transfer |
| Klartext | Verschlüsselt | Notiz wird verschlüsselt |
| Verschlüsselt | Klartext | Notiz wird entschlüsselt |

### Technische Details

Bei der Wiederherstellung einer verschlüsselten Version:
1. Der aktuelle Zustand wird als Snapshot gespeichert (non-destructive)
2. Die `encryption_metadata` wird aus Konstanten rekonstruiert:
   ```json
   {
     "version": 2,
     "algorithm": "XChaCha20-Poly1305",
     "kdf": "Argon2id",
     "kdf_strength": "interactive",
     "nonce_bytes": 24,
     "wrapped_dek": "<aus Version>"
   }
   ```
3. Die Notiz wird mit dem `wrapped_dek` der Ziel-Version aktualisiert
4. Der Client entschlüsselt mit dem aktuellen KEK

**Hinweis:** Da der `wrapped_dek` in der Version gespeichert ist, kann die Notiz nur entschlüsselt werden, wenn der User noch denselben KEK hat (gleiches Passwort).

## 🔄 Passwort ändern

### Aktueller Prozess

*(Hinweis: Automatische DEK-Rewrapping ist noch nicht implementiert)*

Wenn du dein Passwort änderst:
1. Gehe zu **Einstellungen → Konto**
2. Klicke auf **"Passwort ändern"**
3. Gib dein aktuelles Passwort ein
4. Gib dein neues Passwort ein
5. Bestätige das neue Passwort

**WICHTIG:** Nach einer Passwortänderung:
- Dein Verschlüsselungsschlüssel ändert sich
- Bestehende Notizen können nicht mehr entschlüsselt werden
- Du musst die Notizen manuell migrieren (siehe unten)

### Manuelle Migration nach Passwortänderung

1. **VOR der Passwortänderung:**
   - Exportiere alle Notizen
   - Speichere sie lokal

2. **Nach der Passwortänderung:**
   - Importiere die Notizen wieder
   - Oder erstelle sie manuell neu

### Zukünftige Verbesserung

In einer zukünftigen Version wird xelanote automatisch:
1. Alle DEKs mit dem alten KEK entschlüsseln
2. Alle DEKs mit dem neuen KEK neu verschlüsseln
3. Die Notizen bleiben lesbar

## ❓ FAQ (Häufig gestellte Fragen)

### Allgemein

**F: Sind meine Notizen wirklich sicher?**
A: Ja, solange dein Passwort sicher ist und nicht kompromittiert wurde. Der Server hat niemals Zugriff auf deine unverschlüsselten Daten.

**F: Kann der Administrator meine Notizen lesen?**
A: Nein. Selbst mit vollem Datenbankzugriff sieht der Admin nur verschlüsselte Daten. Ohne dein Passwort sind die Notizen unlesbar.

**F: Was passiert, wenn ich mein Passwort vergesse?**
A: Ohne Recovery Key sind deine verschlüsselten Notizen unwiederbringlich verloren. Mit Recovery Key kannst du Zugriff wiederherstellen.

**F: Wird die App langsamer durch Verschlüsselung?**
A: Nur minimal. Die Verschlüsselung dauert 1-2ms pro Notiz. Bei der Anmeldung gibt es eine ~600ms Verzögerung für die Schlüsselableitung.

**F: Kann ich verschlüsselte Notizen mit anderen teilen?**
A: Verschluesselte Notizen koennen nicht direkt geteilt werden, da jeder User seinen eigenen Schluessel hat. Du kannst aber einzelne Notizen ueber das More-Menu im Editor entschluesseln und sie dann teilen. Beim erneuten Verschluesseln werden bestehende Shares automatisch entfernt.

**F: Kann ich einen verschluesselten Ordner teilen?**
A: Nein. Ordner mit aktivierter Verschluesselung (`encryption_default = true`) oder mit verschluesselten Notizen koennen nicht geteilt werden. Du musst zuerst alle Notizen entschluesseln und den Encryption-Default auf `false` setzen. Umgekehrt kann die Verschluesselung nicht aktiviert werden solange ein Ordner geteilt ist -- alle Shares muessen vorher entfernt werden. Falls eine einzelne Notiz in einem geteilten Ordner nachtraeglich verschluesselt wird, wird sie aus der Shared-Ansicht gefiltert (Defense-in-Depth).

### Verschlüsselung

**F: Warum dauert die Anmeldung länger?**
A: Die Schlüsselableitung mit Argon2id dauert ~600ms. Das ist bewusst so, um Brute-Force-Angriffe zu verhindern.

**F: Kann ich die Verschluesselung deaktivieren?**
A: Du kannst einzelne Notizen ueber das More-Menu im Editor entschluesseln (und spaeter wieder verschluesseln). Ordner koennen als "unverschluesselt" markiert werden -- neue Notizen darin werden dann ohne Verschluesselung erstellt. Die globale Verschluesselungs-Infrastruktur bleibt aber immer aktiv.

**F: Werden auch Anhänge verschlüsselt?**
A: Aktuell noch nicht. Die Verschlüsselung von Anhängen ist für eine zukünftige Version geplant.

**F: Was ist der Unterschied zwischen DEK und KEK?**
A:
- KEK (Key Encryption Key): Wird aus deinem Passwort abgeleitet, verschlüsselt die DEKs
- DEK (Data Encryption Key): Pro Notiz ein eigener Schlüssel, verschlüsselt den Inhalt

### Recovery Key

**F: Wo soll ich meinen Recovery Key aufbewahren?**
A: In einem Passwort-Manager, ausgedruckt in einem Safe, oder auf einem verschlüsselten USB-Stick. Niemals unverschlüsselt in der Cloud!

**F: Kann ich mehrere Recovery Keys haben?**
A: Aktuell nicht. Du kannst aber einen neuen Recovery Key generieren (der alte wird dann ungültig).

**F: Was ist, wenn ich meinen Recovery Key verliere?**
A: Dann hast du nur noch dein Passwort. Verlierst du auch das Passwort, sind die Notizen unwiederbringlich verloren.

### Migration

**F: Muss ich meine alten Notizen migrieren?**
A: Nein, das ist optional. Alte Notizen können im Klartext bleiben. Nur neue Notizen werden automatisch verschlüsselt.

**F: Kann ich die Migration rückgängig machen?**
A: Nein, die Migration ist irreversibel. Erstelle vorher ein Backup, falls du zurück willst.

**F: Was passiert, wenn die Migration fehlschlägt?**
A: Bereits migrierte Notizen bleiben verschlüsselt. Fehlgeschlagene Notizen bleiben im Klartext. Du kannst die Migration erneut starten.

**F: Wie lange dauert die Migration?**
A: Ca. 50-100ms pro Notiz. Für 100 Notizen etwa 5-10 Sekunden, für 1000 Notizen etwa 1-2 Minuten.

### Fehlerbehandlung

**F: Ich bekomme "Encryption locked - please re-login". Was tun?**
A: Dein Verschlüsselungsschlüssel ist nicht mehr im Speicher. Melde dich ab und wieder an.

**F: Meine Notizen werden nicht entschlüsselt. Was ist los?**
A: Mögliche Ursachen:
1. Du bist nicht angemeldet
2. Falsches Passwort bei der Anmeldung
3. Korrupte Verschlüsselungs-Metadaten
4. Browser-Cache löschen und neu versuchen

**F: Kann ich meine verschlüsselten Notizen auf einem anderen Gerät öffnen?**
A: Ja! Melde dich einfach mit deinem Passwort an. Der Schlüssel wird automatisch neu abgeleitet.

## 🔧 Technische Implementierung

### Für Entwickler

Detaillierte technische Informationen findest du in:
- `ENCRYPTION_IMPLEMENTATION_STATUS.md` - Implementierungsstatus
- `ENCRYPTION_TEST_REPORT.md` - Testergebnisse
- `ENCRYPTION_FINAL_SUMMARY.md` - Vollständige Zusammenfassung
- `docs/architecture.md` - Systemarchitektur

### Kryptografische Spezifikationen (v2)

**Argon2id Parameter:**
```
Algorithm: Argon2id (Hybrid Mode)
Operations limit: 3 (INTERACTIVE preset)
Memory limit: 67108864 bytes (64 MB)
Hash length: 32 bytes (256 bit KEK)
Salt length: 16 bytes (128 bit)
Execution: Web Worker (non-blocking UI)
```

**XChaCha20-Poly1305-IETF:**
```
Algorithm: XChaCha20 (stream cipher) + Poly1305 (MAC)
Key length: 256 bit
Nonce length: 192 bit (24 bytes)
Authentication tag: 128 bit (16 bytes, in ciphertext)
Standard: IETF RFC 8439 (extended ChaCha20)
```

**Key Wrapping:**
```
Algorithm: XChaCha20-Poly1305-IETF (same as content encryption)
Format: [nonce (24 bytes)][encrypted_dek (32 bytes)][auth_tag (16 bytes)]
```

**Siehe auch:** [Encryption v2 Technical Documentation](./encryption-v2.md)

### Datenformat (v2)

**Verschlüsselte Notiz in der Datenbank:**
```sql
encrypted_content BLOB       -- Verschlüsselter Inhalt (XChaCha20-Poly1305)
wrapped_dek TEXT             -- Verschlüsselter DEK (URL-safe Base64, no padding)
encryption_metadata TEXT     -- JSON mit Metadaten
encryption_version INTEGER   -- Schema-Version (2 = v2, 1 = v1 legacy)
content_encrypted INTEGER    -- 1 wenn Inhalt verschlüsselt
title_encrypted INTEGER      -- 1 wenn Titel verschlüsselt
encrypted_title TEXT         -- Verschlüsselter Titel (optional)
```

**Encryption Metadata JSON (v2):**
```json
{
  "version": 2,
  "algorithm": "XChaCha20-Poly1305-IETF",
  "kdf": "Argon2id",
  "kdf_params": {
    "opslimit": 3,
    "memlimit": 67108864
  }
}
```

**Hinweis:** Nonce und Auth-Tag sind im Ciphertext enthalten (nicht separat in Metadata).

## 📞 Support

### Probleme melden

Bei Problemen mit der Verschlüsselung:
1. Erstelle ein Issue auf GitHub: [xelanote Issues](https://github.com/xela-io/xelanote/issues)
2. Gib an:
   - Browser und Version
   - Fehlermeldung (falls vorhanden)
   - Schritte zur Reproduktion
   - Screenshots (keine sensiblen Daten!)

### Sicherheitslücken melden

Wenn du eine Sicherheitslücke findest:
1. **NICHT öffentlich posten**
2. E-Mail an: security@xelanote.com (oder repo-maintainer)
3. Beschreibe die Lücke detailliert
4. Warte auf Antwort, bevor du Details veröffentlichst

## 📚 Weitere Ressourcen

### xelanote Documentation
- [Encryption v2 Technical Documentation](./encryption-v2.md) - Detaillierte v2 Implementierung
- [E2E Encryption Quick Start](./e2e-encryption-quickstart.md) - Schnelleinstieg
- [E2E Encryption Deployment](./e2e-encryption-deployment.md) - Deployment-Details

### Specifications & Standards
- [RFC 8439: ChaCha20-Poly1305 for IETF Protocols](https://datatracker.ietf.org/doc/html/rfc8439)
- [RFC 9106: Argon2 Memory-Hard Function](https://datatracker.ietf.org/doc/html/rfc9106)
- [libsodium Documentation](https://doc.libsodium.org/)
- [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html)

## 🎓 Glossar

**E2E (Ende-zu-Ende):** Verschlüsselung, bei der nur Sender und Empfänger die Daten lesen können, nicht der Server.

**KEK (Key Encryption Key):** Der Hauptschlüssel, der aus deinem Passwort abgeleitet wird und zum Verschlüsseln der DEKs verwendet wird.

**DEK (Data Encryption Key):** Der Schlüssel, der zum Verschlüsseln einer einzelnen Notiz verwendet wird. Jede Notiz hat einen eigenen DEK.

**Salt:** Ein zufälliger Wert, der zur Passwortverschlüsselung hinzugefügt wird, um Rainbow-Table-Angriffe zu verhindern.

**Argon2id:** Eine moderne Key Derivation Function, die sowohl gegen GPU- als auch gegen Side-Channel-Angriffe resistent ist.

**XChaCha20-Poly1305:** Authenticated Encryption Algorithm (AEAD) mit extended nonce space. Kombination aus ChaCha20 Stream-Cipher und Poly1305 MAC. IETF RFC 8439 Standard.

**libsodium:** Battle-tested Cryptography Library (seit 2013), JavaScript-Port der libsodium C-Bibliothek. Verwendet von Signal, WireGuard, etc.

**AEAD (Authenticated Encryption with Associated Data):** Verschlüsselung mit integrierter Authentifizierung. Verhindert Tampering und bietet Confidentiality + Integrity in einem Schritt.

**Zero-Knowledge:** Ein System, bei dem der Service-Provider keinen Zugriff auf die Daten der Nutzer hat.

**Brute-Force:** Ein Angriffsversuch, bei dem alle möglichen Passwörter systematisch ausprobiert werden.

**Rainbow Table:** Vorgenerierte Tabellen von Hash-Werten, die Angriffe auf schlecht geschützte Passwörter beschleunigen.

---

**Letzte Aktualisierung:** 7. Februar 2026
**Version:** 1.2.0
**Autor:** xelanote Development Team
