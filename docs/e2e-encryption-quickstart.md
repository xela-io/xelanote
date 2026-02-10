# E2E-Verschlüsselung - Quick Start Guide

## 🚀 In 5 Minuten zur verschlüsselten Notiz

### Schritt 1: Anmelden (30 Sekunden)

Melde dich normal bei xelanote an:
- Gib deinen Benutzernamen und Passwort ein
- Klicke auf "Anmelden"
- ⏱️ Erste Anmeldung dauert ~600ms länger (Schlüsselableitung)

**✅ Fertig!** Die Verschlüsselung ist jetzt aktiv.

### Schritt 2: Notiz erstellen (1 Minute)

Erstelle eine neue Notiz wie gewohnt:
- Klicke auf "Neue Notiz"
- Schreibe deinen Text
- Klicke auf "Speichern" (oder warte auf Auto-Save)

**✅ Deine Notiz ist jetzt Ende-zu-Ende verschlüsselt!**

### Schritt 3: Recovery Key erstellen (2 Minuten)

**WICHTIG:** Ohne Recovery Key kannst du bei vergessenem Passwort nicht mehr auf deine Notizen zugreifen!

1. Gehe zu **Einstellungen → Verschlüsselung**
2. Scrolle zu "Recovery Key"
3. Klicke auf **"Recovery Key erstellen"**
4. Klicke auf **"Recovery Key generieren"**
5. Klicke auf **"Als Textdatei herunterladen"**
6. Speichere die Datei an einem sicheren Ort:
   - ✅ Passwort-Manager (empfohlen)
   - ✅ Verschlüsselter USB-Stick
   - ✅ Ausdrucken und in Safe legen
   - ❌ NICHT auf dem Desktop oder in der Cloud!

**✅ Du bist jetzt vollständig geschützt!**

### Schritt 4: Alte Notizen migrieren (Optional, 2-10 Minuten)

Wenn du bereits Notizen hast, verschlüssele sie:

1. Gehe zu **Einstellungen → Migration**
2. Prüfe die Statistiken:
   - Wie viele Notizen sind noch Klartext?
   - Wie viele sind bereits verschlüsselt?
3. Klicke auf **"Migration starten"**
4. Warte, bis alle Notizen verschlüsselt sind
5. ✅ Fertig! Alle Notizen sind jetzt geschützt.

**Dauer:**
- 10 Notizen: ~1 Sekunde
- 100 Notizen: ~5-10 Sekunden
- 1000 Notizen: ~1-2 Minuten

---

## 🔒 Was wird verschlüsselt?

### ✅ Immer geschützt
- **Notiz-Inhalt** - Nur du kannst ihn lesen

### ⚙️ Optional geschützt
- **Notiz-Titel** - Aktiviere "Titel verschlüsseln" in den Einstellungen

### ⚠️ Nicht verschlüsselt
- Ordner-Namen
- Erstellungs-/Änderungsdatum
- Anzahl der Notizen

---

## 🛡️ Maximale Sicherheit (Optional)

Für maximalen Datenschutz:

1. **Aktiviere "Titel verschlüsseln"**
   - Gehe zu **Einstellungen → Verschlüsselung**
   - Aktiviere "Titel verschlüsseln"
   - Akzeptiere, dass Titel-Suche nicht mehr funktioniert

2. **Deaktiviere "Keywords extrahieren"** (Standard)
   - Sollte bereits deaktiviert sein
   - Wenn nicht: Deaktiviere es sofort
   - Verhindert Datenlecks

3. **Verwende generische Ordnernamen**
   - ❌ "/Privat/Arzttermine/Kardiologie"
   - ✅ "/Privat/Gesundheit"

4. **Starkes Passwort verwenden**
   - Mindestens 12 Zeichen
   - Zufällig generiert (Passwort-Manager)
   - Einzigartig für xelanote

---

## ❓ Häufige Fragen

**F: Ist das wirklich sicher?**
A: Ja! Der Server sieht nur verschlüsselte Daten. Selbst Admins können deine Notizen nicht lesen.

**F: Wird die App langsamer?**
A: Nur minimal. ~600ms bei der Anmeldung, ~1-2ms pro gespeicherter Notiz.

**F: Was wenn ich mein Passwort vergesse?**
A: Mit Recovery Key: Wiederherstellung möglich. Ohne Recovery Key: Daten unwiederbringlich verloren!

**F: Kann ich verschlüsselte Notizen teilen?**
A: Aktuell nein. Jeder User hat seinen eigenen Schlüssel.

**F: Kann der Admin meine Notizen lesen?**
A: Nein! Selbst mit Datenbank-Zugriff nur verschlüsselte Daten sichtbar.

---

## 🚨 WICHTIG: Das musst du wissen!

### ⚠️ Recovery Key ist KRITISCH

Ohne Recovery Key UND ohne Passwort sind deine Notizen **unwiederbringlich verloren**!

**Erstelle JETZT einen Recovery Key!**

### 🔐 Passwort niemals teilen

- Teile dein Passwort mit niemandem
- Teile deinen Recovery Key mit niemandem
- xelanote wird dich niemals nach deinem Passwort fragen

### 💾 Regelmäßige Backups

Auch verschlüsselte Notizen sollten gesichert werden:
- Exportiere Notizen regelmäßig
- Speichere Backups verschlüsselt
- Teste die Wiederherstellung

---

## 📚 Mehr erfahren

Ausführliche Dokumentation: [E2E-Verschlüsselung Vollständige Dokumentation](./e2e-encryption.md)

Themen:
- Technische Details (Argon2id, AES-GCM-256)
- Sicherheitshinweise
- Erweiterte Einstellungen
- Fehlerbehandlung
- FAQ

---

## ✅ Checkliste: Bin ich geschützt?

- [ ] Ich habe mich angemeldet (Verschlüsselung ist aktiv)
- [ ] Ich habe eine verschlüsselte Notiz erstellt
- [ ] Ich habe einen Recovery Key erstellt
- [ ] Ich habe den Recovery Key sicher aufbewahrt
- [ ] Ich habe meine alten Notizen migriert (optional)
- [ ] Ich verwende ein starkes, einzigartiges Passwort
- [ ] Ich weiß, dass ich ohne Passwort+Recovery Key keinen Zugriff mehr habe

**Alle Punkte abgehakt? Dann bist du vollständig geschützt! 🎉**

---

**⏱️ Gesamtzeit:** 5-10 Minuten
**🔒 Sicherheit:** Maximum
**💡 Nächster Schritt:** Notizen schreiben und genießen!
