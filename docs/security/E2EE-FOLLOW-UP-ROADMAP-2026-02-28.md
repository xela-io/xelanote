# E2EE Follow-up Roadmap (2026-02-28)

## Ziel

Konkrete Nachfolgeplanung fuer die verbleibenden E2EE-Risiken, die **nicht** durch die bisherigen Remediations (PR1-PR7) vollständig gelöst werden konnten.

Referenzen:
- `docs/security/E2EE-SECURITY-AUDIT-ADDENDUM-2026-02-28.md`
- `docs/security/E2EE-REMEDIATION-PLAN-2026-02-28.md`

## Ausgangslage (offene Punkte)

1. Browser-/XSS-Risiko bleibt ein strukturelles Restrisiko bei KEK-Persistenz.
2. Kein Forward Secrecy / Post-Compromise Security (kein Ratchet-Modell).
3. Kein Multi-Device-Key-Exchange/Revocation-Modell.
4. E2EE deckt Metadaten weiterhin nur teilweise ab (bewusster Scope).

---

## P1: Operative Härtung (kurzfristig)

## 1) XSS-Defense-in-Depth Paket

Scope:
1. CSP-Policy auf strikte Allowlist validieren und nonces/hashes hart machen.
2. Subresource Integrity fuer externe Assets, wo sinnvoll.
3. Security-Regression-Checks fuer kritische UI-Flows (Settings/Recovery/Editor) erweitern.

Akzeptanzkriterien:
1. Dokumentierte CSP-Baseline inkl. Test-Fail bei unerwarteten Lockerungen.
2. Kein bekannter Inline-Script-Pfad ohne explizite Absicherung.
3. CI-Checks fuer Security-Headers/CSP laufen stabil.

## 2) Recovery Readiness Sichtbarkeit

Scope:
1. User-seitige Anzeige, ob Recovery-Wrapper vollstaendig vorhanden sind.
2. UI-Hinweis nach verschluesselten Create/Update-Flows: Recovery-Key ggf. neu einrichten.
3. Optional: dedizierter Endpoint fuer Recovery-Readiness-Status.

Akzeptanzkriterien:
1. Nutzer sehen klar, ob encrypted recovery aktuell funktioniert.
2. Keine stillen Failure-Zustaende mehr bei unvollstaendigem Wrapper-Stand.

## 3) Metadaten-Scope explizit in UX

Scope:
1. Sichtbaren Scope-Block in Security/Encryption Settings mit Link auf Doku.
2. Konsistente Formulierung in UI-Strings und Handbuch.

Akzeptanzkriterien:
1. Keine impliziten "Zero-Knowledge fuer alles"-Missverstaendnisse mehr.

---

## P2: Kryptographie- und Protokoll-Weiterentwicklung (mittelfristig)

## 1) Multi-Device Trust Modell

Scope:
1. Device-Key-Identitaeten (pro Geraet) + Authorisierung neuer Geraete.
2. Revocation/Removal-Flow fuer kompromittierte oder verlorene Geraete.
3. Dokumentiertes Trust-Onboarding (z. B. QR/Out-of-Band Bestätigung).

Akzeptanzkriterien:
1. Neues Geraet kann nicht stillschweigend Key-Material untergeschoben bekommen.
2. Entfernte Geraete verlieren Zugriff auf neue Inhalte.

## 2) Forward Secrecy / PCS Evaluierung

Scope:
1. Architekturentscheidung dokumentieren, ob Ratcheting fuer Produktmodell sinnvoll ist.
2. Falls ja: PoC fuer geeignete Datenpfade (nicht blind auf alle Note-Operationen erzwingen).

Akzeptanzkriterien:
1. Entscheidung "einfuehren vs. nicht einfuehren" ist schriftlich und begruendet.
2. Bei Einfuehrung: messbarer Sicherheitsgewinn ohne untragbare UX-/Komplexitaetskosten.

---

## Laufende Governance

1. Quartalsweises Security-Delta-Review fuer E2EE (Code + Doku).
2. Addendum mit Datum fortschreiben, statt nur einmaligen Snapshot zu behalten.
3. Offene Punkte in Roadmap/TODO als explizite Security-Epics fuehren.

## Definition of Done (Follow-up abgeschlossen)

1. P1-Paket umgesetzt und in CI/Dokumentation verankert.
2. P2-Architekturentscheidung formal getroffen und dokumentiert.
3. Keine widerspruechlichen Recovery-/E2EE-Claims mehr in Repo-Dokumenten.
