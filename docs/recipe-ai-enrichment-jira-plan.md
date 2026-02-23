# Jira-Plan: KI-Anreicherung manueller Rezepte

## Ziel
Manuell erstellte Rezepte per KI anreichern (Features + Embeddings), Originaldaten unangetastet lassen und die Similarity-Suche durch hybrides Scoring verbessern.

## Annahmen
- Backend mit async Job/Worker vorhanden (oder kurzfristig einführbar)
- Datenbank unterstützt JSON/JSONB; für Embeddings idealerweise `pgvector`
- Similarity-Suche existiert bereits in einer Basisform

## Epic Übersicht
- `RAI-EPIC-1` Datenmodell & Persistenz
- `RAI-EPIC-2` Enrichment-Worker Pipeline
- `RAI-EPIC-3` Resolver & Stale-Handling
- `RAI-EPIC-4` Hybrid Similarity Search
- `RAI-EPIC-5` UI/Admin Overrides (optional)
- `RAI-EPIC-6` QA, Evaluation & Rollout
- `RAI-EPIC-7` Monitoring & Debuggability

## Definition of Done (global)
- Code ist gemerged und deploybar
- Relevante Unit/Integration-Tests vorhanden
- Logging für Fehlerfälle vorhanden
- Dokumentation der Felder/Flows aktualisiert
- Keine Überschreibung manueller Originaldaten durch KI

## RAI-EPIC-1: Datenmodell & Persistenz

### RAI-101: `recipes` um KI-Felder erweitern
- Typ: Story
- Story Points: 3
- Beschreibung:
  - DB-Felder ergänzen: `ai_status`, `ai_features`, `user_overrides`, `resolved_features` (optional cache), `embedding`
- Akzeptanzkriterien:
  - Manuelle Rezepte können ohne KI-Daten gespeichert werden
  - `ai_status` defaultet auf `pending`
  - Migration ist rückwärtskompatibel
- Dependencies:
  - Keine
- Definition of Done:
  - Migration läuft lokal/CI erfolgreich
  - Read/Write im ORM/Repository unterstützt

### RAI-102: AI-Metadaten & Versionierung spezifizieren
- Typ: Story
- Story Points: 2
- Beschreibung:
  - Struktur für `ai_features.version`, `generated_at`, `confidence`, `field_confidence` festlegen und implementieren
- Akzeptanzkriterien:
  - KI-Version pro Rezept nachvollziehbar
  - Feld-Confidence speicherbar
- Dependencies:
  - `RAI-101`
- Definition of Done:
  - JSON-Schema/Contract dokumentiert
  - Serialization/Validation vorhanden

## RAI-EPIC-2: Enrichment-Worker Pipeline

### RAI-201: Async Enrichment-Job bei manuellen Rezepten triggern
- Typ: Story
- Story Points: 3
- Beschreibung:
  - Nach `recipe create` für `source=user` Enrichment-Job einreihen
- Akzeptanzkriterien:
  - Rezept wird synchron gespeichert
  - Enrichment startet asynchron
  - Keine Jobs für nicht-manuelle Rezepte (falls nicht gewünscht)
- Dependencies:
  - `RAI-101`
- Definition of Done:
  - Trigger integriert
  - Job-Payload dokumentiert

### RAI-202: Zutatennormalisierung implementieren
- Typ: Story
- Story Points: 5
- Beschreibung:
  - Mengen/Einheiten parsen, Synonyme mappen, Zutatenrollen vorbereiten (`signature`, `secondary`, `seasoning`)
- Akzeptanzkriterien:
  - Normalisierte Zutatenliste wird erzeugt
  - Gewürze/Standardzutaten werden abgewertet
  - Synonym-Mapping ist erweiterbar
- Dependencies:
  - `RAI-201`
- Definition of Done:
  - Unit-Tests für Parser/Synonyme
  - Beispiel-Mappings dokumentiert

### RAI-203: KI-Feature-Extraktion implementieren
- Typ: Story
- Story Points: 8
- Beschreibung:
  - Extraktion von `dish_type`, `cuisine`, `techniques`, `diet_tags`, `flavor_profile`, `difficulty`, `time_estimate_min`, `semantic_summary`
- Akzeptanzkriterien:
  - Ergebnisse werden in `ai_features` gespeichert
  - `confidence` + `field_confidence` vorhanden
  - Fehlerfälle führen nicht zum Verlust des Rezepts
- Dependencies:
  - `RAI-102`
  - `RAI-201`
  - `RAI-202` (für angereicherte Inputs empfohlen)
- Definition of Done:
  - Prompt/Contract versioniert
  - Integration-Test mit Beispielrezept

### RAI-204: Embedding-Erzeugung & Speicherung
- Typ: Story
- Story Points: 5
- Beschreibung:
  - Embedding aus `title + semantic_summary + relevanten Schritten` erzeugen und speichern
- Akzeptanzkriterien:
  - `embedding` wird persistiert
  - Fehlerfall setzt `ai_status=failed`
  - Wiederholbarkeit über definierte Input-Bildung
- Dependencies:
  - `RAI-201`
  - `RAI-203`
- Definition of Done:
  - Embedding-Input-Builder getestet
  - Fehlerlogging vorhanden

### RAI-205: Retry, Fehlerbehandlung, Logging
- Typ: Story
- Story Points: 3
- Beschreibung:
  - Retry bei transienten Fehlern, saubere Statusübergänge, Logging der Fehlerursachen
- Akzeptanzkriterien:
  - Retries konfigurierbar
  - `failed`-Status sichtbar
  - Endlosschleifen werden verhindert
- Dependencies:
  - `RAI-201`
  - `RAI-203`
  - `RAI-204`
- Definition of Done:
  - Retry-Policy dokumentiert
  - Monitoring-Events vorhanden

## RAI-EPIC-3: Resolver & Stale-Handling

### RAI-301: `resolved_features` Builder implementieren
- Typ: Story
- Story Points: 5
- Beschreibung:
  - Merge-Priorität: `user_overrides` > `content` > `ai_features`
- Akzeptanzkriterien:
  - User-Daten werden nie überschrieben
  - Fehlende Felder bleiben `null`
  - Resolver ist deterministisch
- Dependencies:
  - `RAI-101`
  - `RAI-102`
- Definition of Done:
  - Unit-Tests für Merge-Priorität
  - Dokumentation der Prioritätslogik

### RAI-302: `stale`-Handling bei Rezeptänderungen
- Typ: Story
- Story Points: 3
- Beschreibung:
  - Bei Änderungen an Titel/Zutaten/Schritten `ai_status=stale` setzen und Re-Enrichment triggern
- Akzeptanzkriterien:
  - Alte KI-Daten bleiben bis Recompute nutzbar
  - Neuer Enrichment-Job wird erstellt
- Dependencies:
  - `RAI-201`
  - `RAI-301`
- Definition of Done:
  - Änderungsfelder sauber erkannt
  - Statusübergänge getestet

## RAI-EPIC-4: Hybrid Similarity Search

### RAI-401: Candidate Search über Embeddings (Top-K)
- Typ: Story
- Story Points: 5
- Beschreibung:
  - Top-K Kandidaten über Vektorähnlichkeit laden (z. B. Top 100)
- Akzeptanzkriterien:
  - Suchlauf performant
  - Rezept selbst wird ausgeschlossen
  - Fehlende Embeddings werden robust behandelt
- Dependencies:
  - `RAI-204`
- Definition of Done:
  - Query dokumentiert
  - Performance-Benchmark (Basis) vorhanden

### RAI-402: Hybrid Re-Ranking implementieren
- Typ: Story
- Story Points: 8
- Beschreibung:
  - Re-Ranking mit Komponenten:
    - `semantic_similarity`
    - `weighted_ingredient_similarity`
    - `technique_match`
    - `time_similarity`
- Akzeptanzkriterien:
  - Endscore ist nachvollziehbar
  - Ranking ist gegenüber Basisvergleich evaluierbar
  - Score-Komponenten sind separat berechenbar
- Dependencies:
  - `RAI-202`
  - `RAI-301`
  - `RAI-401`
- Definition of Done:
  - Komponenten-Tests vorhanden
  - Default-Gewichte konfigurierbar

### RAI-403: Dynamische Gewichtung bei fehlenden Feldern
- Typ: Story
- Story Points: 3
- Beschreibung:
  - Gewichte neu normieren, wenn `technique/time/etc.` fehlen
- Akzeptanzkriterien:
  - Unvollständige manuelle Rezepte werden nicht unfair benachteiligt
  - Score bleibt im erwarteten Bereich (0..1 oder definierter Range)
- Dependencies:
  - `RAI-402`
- Definition of Done:
  - Tests für fehlende Felder
  - Verhalten dokumentiert

## RAI-EPIC-5: UI/Admin Overrides (optional)

### RAI-501: KI-Vorschläge im Rezept-Editor anzeigen (Read-only)
- Typ: Story
- Story Points: 3
- Beschreibung:
  - `ai_features` im Admin/Editor sichtbar machen
- Akzeptanzkriterien:
  - User sieht erkannte Küche/Gerichtstyp/Tags
  - Anzeige unterscheidet KI-Daten von Originaldaten
- Dependencies:
  - `RAI-203`
  - `RAI-301`
- Definition of Done:
  - UI-Texte gekennzeichnet (KI-Vorschlag)
  - Basis-UI-Test oder Screenshot-Doku

### RAI-502: User-Overrides speichern und im Resolver berücksichtigen
- Typ: Story
- Story Points: 5
- Beschreibung:
  - Editierbare Override-Felder speichern (`cuisine`, `dish_type`, `diet_tags`, `time`, `difficulty`)
- Akzeptanzkriterien:
  - Änderungen landen nur in `user_overrides`
  - Suche nutzt danach `resolved_features`
  - KI-Daten bleiben unverändert nachvollziehbar
- Dependencies:
  - `RAI-301`
  - `RAI-501`
- Definition of Done:
  - End-to-End Test: Override ändert Ranking/Filterbasis
  - Auditierbarkeit gegeben

## RAI-EPIC-6: QA, Evaluation & Rollout

### RAI-601: Eval-Datensatz erstellen (50-100 Rezepte)
- Typ: Story
- Story Points: 5
- Beschreibung:
  - Goldset mit positiven/negativen Similarity-Beispielen erstellen
- Akzeptanzkriterien:
  - Pro Rezept sind gute und schlechte Treffer dokumentiert
  - Datensatz reproduzierbar nutzbar
- Dependencies:
  - Keine (parallel möglich)
- Definition of Done:
  - Ablageort + Format dokumentiert
  - Review mit Fachseite/Product erfolgt

### RAI-602: Ranking-Metriken & A/B-Vergleich (alt vs hybrid)
- Typ: Story
- Story Points: 5
- Beschreibung:
  - `Precision@5` und manuelle Qualitätsbewertung messen; Vergleich altes vs neues Ranking
- Akzeptanzkriterien:
  - Vorher/Nachher Ergebnisse liegen vor
  - Entscheidung für Rollout ist datenbasiert
- Dependencies:
  - `RAI-402`
  - `RAI-403`
  - `RAI-601`
- Definition of Done:
  - Ergebnisbericht abgelegt
  - Empfehlungen für Gewichtsanpassungen dokumentiert

### RAI-603: Stufenweiser Rollout planen & umsetzen
- Typ: Story
- Story Points: 3
- Beschreibung:
  - Phase 1 neue manuelle Rezepte, Phase 2 Bestand, Phase 3 Tuning
- Akzeptanzkriterien:
  - Feature-Flag oder Rollout-Steuerung vorhanden
  - Rollout ohne Suchausfall möglich
- Dependencies:
  - `RAI-205`
  - `RAI-602`
- Definition of Done:
  - Rollout-Runbook dokumentiert
  - Rollback-Plan definiert

## RAI-EPIC-7: Monitoring & Debuggability

### RAI-701: Monitoring / Dashboards für Enrichment & Suche
- Typ: Story
- Story Points: 3
- Beschreibung:
  - Dashboard für `ai_status`, Erfolgsrate, Fehlerrate, Latenz, CTR ähnliche Rezepte
- Akzeptanzkriterien:
  - Kern-KPIs sichtbar
  - Alerts für Fehlerrate/Latenz definiert (mindestens Basis)
- Dependencies:
  - `RAI-205`
  - `RAI-603` (für Rollout-Monitoring empfohlen)
- Definition of Done:
  - Dashboard-Link dokumentiert
  - Alarm-Schwellen dokumentiert

### RAI-702: Score-Debug-Logging / Explainability intern
- Typ: Story
- Story Points: 3
- Beschreibung:
  - Pro Treffer Score-Komponenten intern loggen oder debug-endpoint bereitstellen
- Akzeptanzkriterien:
  - Ranking-Entscheidungen debugbar
  - Sensible Daten werden nicht unnötig exponiert
- Dependencies:
  - `RAI-402`
  - `RAI-403`
- Definition of Done:
  - Debug-Ausgabe für mindestens einen Suchpfad verfügbar
  - Zugriff intern beschränkt

## Empfohlene Sprint-Reihenfolge (MVP zuerst)

### Sprint 1 (Basis)
- `RAI-101`
- `RAI-102`
- `RAI-201`
- `RAI-202`

### Sprint 2 (KI + Embeddings)
- `RAI-203`
- `RAI-204`
- `RAI-205`
- `RAI-301`

### Sprint 3 (Search Hybrid MVP)
- `RAI-401`
- `RAI-402`
- `RAI-403`
- `RAI-302`

### Sprint 4 (Qualität & Rollout)
- `RAI-601`
- `RAI-602`
- `RAI-603`
- `RAI-701`
- `RAI-702`

### Sprint 5 (optional UI/Admin)
- `RAI-501`
- `RAI-502`

## Risiken / offene Punkte (für Jira-Refinement)
- Welches Embedding-Modell und welche Kosten/Latenz sind akzeptabel?
- Welche Felder dürfen KI schätzen vs müssen explizit `unknown` bleiben?
- Wie groß ist das Synonym-Lexikon initial?
- Reicht JSONB-Resolver oder werden dedizierte Spalten für häufige Filter benötigt?
- Datenschutz/Logging: welche Rezeptinhalte dürfen im Debug landen?
