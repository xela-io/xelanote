# KI-Anreicherung manueller Rezepte: Ticket-Plan

## Ziel
Manuell erstellte Rezepte sollen per KI angereichert werden (Features + Embeddings), ohne Originaldaten zu überschreiben, damit die Similarity-Suche mit einem hybriden Scoring verbessert wird.

## Epic 1: Datenmodell für KI-Anreicherung

### Ticket 1.1 `recipes` um AI-Felder erweitern
- Felder hinzufügen:
  - `ai_status` (`pending|done|stale|failed`)
  - `ai_features` (JSON/JSONB)
  - `user_overrides` (JSON/JSONB)
  - `resolved_features` (JSON/JSONB, optional gecached)
  - `embedding` (Vector)
- Akzeptanzkriterien:
  - Manuelle Rezepte können ohne AI-Daten gespeichert werden
  - Default `ai_status = pending`

### Ticket 1.2 Versionierung & Metadaten für AI-Features
- In `ai_features` aufnehmen:
  - `version`
  - `generated_at`
  - `confidence`
  - `field_confidence`
- Akzeptanzkriterien:
  - AI-Version ist pro Rezept nachvollziehbar
  - Re-Enrichment später möglich

## Epic 2: Enrichment-Worker (Async)

### Ticket 2.1 Job-Trigger bei manuell erstelltem Rezept
- Nach `recipe create` Async-Job enqueue
- Nur für `source=user`
- Akzeptanzkriterien:
  - Rezept wird sofort gespeichert
  - Enrichment läuft asynchron

### Ticket 2.2 Zutatennormalisierung implementieren
- Parser für Mengen/Einheiten
- Synonym-Mapping (z. B. `Cherrytomaten -> tomate`)
- Rollenklassifikation vorbereiten (`signature/secondary/seasoning`)
- Akzeptanzkriterien:
  - Normalisierte Zutatenliste wird erzeugt
  - Gewürze werden erkannt/abgewertet

### Ticket 2.3 KI-Feature-Extraktion
- Extrahieren:
  - `dish_type`, `cuisine`, `techniques`, `diet_tags`
  - `flavor_profile`, `difficulty`, `time_estimate_min`
  - `semantic_summary`
- Akzeptanzkriterien:
  - Ergebnis wird in `ai_features` gespeichert
  - Confidence-Werte vorhanden

### Ticket 2.4 Embedding-Erzeugung
- Embedding auf Basis von:
  - `title + semantic_summary + relevante Schritte`
- Akzeptanzkriterien:
  - `embedding` wird gespeichert
  - Fehlerfall setzt `ai_status=failed`

### Ticket 2.5 Retry / Fehlerbehandlung / Logging
- Retry bei transienten Fehlern
- Logging + Fehlerdetails
- Akzeptanzkriterien:
  - Wiederholte Fehler blockieren System nicht
  - Fehlgeschlagene Jobs sind sichtbar

## Epic 3: Merge-/Resolver-Logik

### Ticket 3.1 `resolved_features` Builder
- Priorität:
  1. `user_overrides`
  2. `content`
  3. `ai_features`
- Akzeptanzkriterien:
  - Kein Überschreiben von User-Daten
  - Fehlende Felder bleiben `null`

### Ticket 3.2 `stale`-Handling bei Rezeptänderung
- Bei Änderung von Titel/Zutaten/Schritten:
  - `ai_status = stale`
  - Re-Enrichment Job enqueue
- Akzeptanzkriterien:
  - Alte AI-Daten bleiben bis Neu-Berechnung nutzbar
  - Neuer Job wird automatisch ausgelöst

## Epic 4: Similarity Search (Hybrid)

### Ticket 4.1 Candidate Search via Vektor (Top-K)
- Query gegen `embedding`
- Rückgabe z. B. `Top 100`
- Akzeptanzkriterien:
  - Performante Kandidatensuche
  - Rezept selbst wird ausgeschlossen

### Ticket 4.2 Re-Ranking mit Hybrid-Score
- Startgewichte:
  - `semantic_similarity`
  - `weighted_ingredient_similarity`
  - `technique_match`
  - `time_similarity`
- Akzeptanzkriterien:
  - Endscore ist nachvollziehbar/logbar
  - Ranking verbessert sich ggü. reinem Zutatenvergleich

### Ticket 4.3 Dynamische Gewichtung bei fehlenden Feldern
- Wenn Faktor fehlt -> Gewichte neu normieren
- Akzeptanzkriterien:
  - Unvollständige manuelle Rezepte werden nicht unfair bestraft

## Epic 5: Admin/UI (optional, aber wertvoll)

### Ticket 5.1 KI-Vorschläge anzeigen (Read-only)
- Anzeige von `ai_features` im Rezept-Editor/Admin
- Akzeptanzkriterien:
  - User sieht, was KI erkannt hat

### Ticket 5.2 User-Overrides speichern
- Editierbare Felder:
  - `cuisine`, `dish_type`, `diet_tags`, `time`, `difficulty`
- Akzeptanzkriterien:
  - Änderungen landen nur in `user_overrides`
  - Suche nutzt danach `resolved_features`

## Epic 6: QA / Evaluation / Rollout

### Ticket 6.1 Eval-Datensatz aufbauen (50-100 Rezepte)
- Pro Rezept:
  - gute ähnliche Treffer
  - schlechte Treffer
- Akzeptanzkriterien:
  - Testset dokumentiert und reproduzierbar

### Ticket 6.2 Ranking-Metriken & Vergleich
- Messen:
  - `Precision@5`
  - manuelle Qualitätsbewertung
- Vorher/Nachher Vergleich (alt vs hybrid)
- Akzeptanzkriterien:
  - Entscheidung auf Basis von Daten möglich

### Ticket 6.3 Stufenweiser Rollout
- Phase 1: nur neue manuelle Rezepte
- Phase 2: Batch-Enrichment Bestand
- Phase 3: Gewichts-Tuning
- Akzeptanzkriterien:
  - Rollout ohne Such-Ausfall
  - Monitoring aktiv

## Technische Tasks (Querschnitt)

### Ticket Q1 Monitoring / Dashboards
- KPIs:
  - `ai_status` Verteilung
  - Erfolgsrate / Fehlerrate
  - Enrichment-Latenz
  - Klickrate "ähnliche Rezepte"
- Akzeptanzkriterien:
  - Probleme früh sichtbar

### Ticket Q2 Audit/Debug-Logging für Scores
- Pro Treffer Score-Komponenten loggen (intern)
- Akzeptanzkriterien:
  - Ranking-Entscheidungen sind debugbar

## MVP-Reihenfolge (empfohlen)
1. `DB-Felder` + `ai_status`
2. `Enrichment-Worker` (Features + Embedding)
3. `resolved_features`
4. `Hybrid-Re-Ranking`
5. `Eval-Testset`
6. `UI-Overrides` (danach)
