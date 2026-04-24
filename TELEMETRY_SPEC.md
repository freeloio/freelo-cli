# Freelo Claude Skill — Telemetrie & Analytics Spec

**Pro:** Freelo backend tým
**Status:** Draft — návrh k realizaci po public launchi skillu
**Autor:** Marek Urban <marek.urban@freelo.io>
**Datum:** 2026-04-15

---

## 🎯 Cíl

Po veřejném spuštění [`claude-freelo-skill`](https://github.com/freeloio/claude-freelo-skill) budeme mít v řádech desítek až nižších stovek uživatelů. Chceme vědět:

1. **Kolik lidí skill skutečně používá** (DAU / WAU / MAU)
2. **Jak intenzivně** (requesty / user / den)
3. **Co dělají nejčastěji** (top endpointy)
4. **Kde narážejí** (error rate, typy chyb)
5. **Jak rychle se aktualizují** po nové verzi skillu (adoption)
6. **Jestli se vrací** (retence 7d / 30d)

---

## 🔑 Jak rozpoznat skill-originated traffic

Skill posílá **unikátní `User-Agent` hlavičku** u každého requestu:

```
User-Agent: Freelo-Claude-Skill/1.0.0
```

Formát: `Freelo-Claude-Skill/<semver>` — např. `Freelo-Claude-Skill/1.0.0`, `Freelo-Claude-Skill/1.1.2`, atd.

**Identifikátory pro parsing:**
- Prefix `Freelo-Claude-Skill` → je to skill traffic (vs. jiný klient / web UI / MCP server / CLI)
- Za lomítkem verze v semver formátu → adoption tracking po release

---

## 📐 Schéma logovaných událostí

**Per request záznam** (nebo aspoň agregované per den/user):

| Field | Type | Popis | Example |
|---|---|---|---|
| `timestamp` | datetime | Čas requestu (UTC) | `2026-04-15T14:30:22Z` |
| `user_id` | int | Freelo user ID z autentizace | `12345` |
| `user_email` | string | Email (pro tagging v reportech) | `marek.urban@freelo.io` |
| `skill_version` | string (semver) | Verze skillu parsnutá z UA | `1.0.0` |
| `endpoint` | string | Normalized path (bez IDs) | `/task/{id}/comments` |
| `method` | string | HTTP metoda | `POST` |
| `status_code` | int | HTTP response code | `200`, `404`, `429`, atd. |
| `response_time_ms` | int | Trvání requestu v ms | `87` |
| `plan_tier` | string (optional) | Free / Paid — pokud je známé | `paid` |

> **Endpoint normalization**: cestu nahraďte placeholder kde jsou dynamické ID. Např. `/task/29284300/comments` → `/task/{id}/comments`. Jinak budete mít tisíce unikátních "endpointů" místo ~70.

**Ukládat:** raw events preferovaně do analytics DB (ClickHouse / BigQuery / Postgres time-series table). Pokud produkční DB, agregujte předtím na denní / hodinové bucket.

**Retention:** doporučuji 13 měsíců (umožňuje year-over-year srovnání + respektuje GDPR preference pro interní analytics bez důvodu držet déle).

---

## 📊 Metriky a reporty

### Primární KPI (týdenní dashboard)

1. **Unique Active Users** (MAU / WAU / DAU)
   ```sql
   SELECT COUNT(DISTINCT user_id) AS mau
   FROM skill_requests
   WHERE ua LIKE 'Freelo-Claude-Skill%'
     AND timestamp >= NOW() - INTERVAL '30 days';
   ```

2. **Total requests**
   ```sql
   SELECT COUNT(*) AS total_reqs
   FROM skill_requests
   WHERE ua LIKE 'Freelo-Claude-Skill%'
     AND timestamp >= NOW() - INTERVAL '30 days';
   ```

3. **Avg requests per active user**
   ```sql
   total_reqs / mau
   ```

4. **Error rate**
   ```sql
   SELECT
     SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) * 100.0 / COUNT(*) AS error_pct
   FROM skill_requests
   WHERE ua LIKE 'Freelo-Claude-Skill%'
     AND timestamp >= NOW() - INTERVAL '7 days';
   ```

### Sekundární (měsíční review)

5. **Top 10 endpointů**
   ```sql
   SELECT endpoint, COUNT(*) AS hits
   FROM skill_requests
   WHERE ua LIKE 'Freelo-Claude-Skill%'
     AND timestamp >= NOW() - INTERVAL '30 days'
   GROUP BY endpoint
   ORDER BY hits DESC
   LIMIT 10;
   ```

6. **Retence 7-day / 30-day** (cohort analysis)
   - User, který byl aktivní v týdnu N, je "retained" pokud byl aktivní i v týdnu N+1 (7d) / N+4 (30d).
   - Standardní cohort tabulka: řádky = týden první aktivity, sloupce = weeks since first.

7. **Version adoption trend**
   ```sql
   SELECT
     date_trunc('day', timestamp) AS day,
     skill_version,
     COUNT(DISTINCT user_id) AS users_on_version
   FROM skill_requests
   WHERE ua LIKE 'Freelo-Claude-Skill%'
     AND timestamp >= NOW() - INTERVAL '60 days'
   GROUP BY day, skill_version
   ORDER BY day;
   ```
   Sleduje: po vydání v1.1.0 — kolik dní trvá než 50 % users updatují, kolik zůstane na staré verzi navždy.

8. **Error type distribution**
   ```sql
   SELECT status_code, COUNT(*) AS hits
   FROM skill_requests
   WHERE ua LIKE 'Freelo-Claude-Skill%'
     AND status_code >= 400
     AND timestamp >= NOW() - INTERVAL '7 days'
   GROUP BY status_code
   ORDER BY hits DESC;
   ```
   Pokud roste specifický status (např. 422 nebo 404), skill má bug nebo Freelo API změnilo behaviour.

### Alerting (doporučeno)

- **Spike 4xx/5xx error rate >10 % za hodinu** → Slack alert do #freelo-api kanálu
- **Drop DAU >30 % den-na-den** → někdo vydal broken verzi nebo Freelo API má outage
- **Excessive 429 per user** → indicates abuse / pathological skill behavior

---

## 🎨 Dashboard

### MVP (stačí pro launch)

**Notion / Confluence page** ručně aktualizovaná jednou týdně z SQL výpisů:
- Týdenní MAU, WAU, DAU
- Top 10 endpointů
- Error rate + top 3 error codes
- Kolik userů na v1.0.0 vs. v1.1.0 (až bude)

**Efort:** ~1 hod dev práce + 15 min/týden udržování.

### Plný dashboard (až MAU > 50)

**Grafana / Metabase / Amplitude**:
- Real-time graf requests/sec
- Cohort retention table (7d / 30d)
- Funnel analysis (uživatel se poprvé přihlásí → druhý den → týden → měsíc)
- Heatmap endpointů po dnech v týdnu
- Version adoption timeline

**Efort:** 3–5 dní dev práce podle nástroje.

---

## 🔒 Privacy & Compliance

- **Žádná nová PII** — skill používá user's vlastní API klíč, takže už teď víte kdo requestuje co. Jen přidáváme tag "byl to skill" do existujících logs.
- **GDPR**: data se mažou po 13 měsících (nebo dle vaší retention policy pro ostatní analytics).
- **Uživatel může vypnout** pouze tak, že přestane používat skill. Neposkytujeme opt-out hlavičku — byl by to signál "false/null" který by vypnul měření. Dokumentace ve skill README transparentně uvádí, že skill posílá `Freelo-Claude-Skill/{version}` header.

---

## 🛠️ Implementation checklist

### Fáze 1: Basic logging (do 1 týdne od public launch skillu)

- [ ] Middleware / log parser na API gateway nebo aplikaci — detekce `ua LIKE 'Freelo-Claude-Skill%'`
- [ ] Schema do analytics DB (nebo event table v existujícím Postgres)
- [ ] Ingestion: per-request event → normalize endpoint path → insert
- [ ] Manuální SQL dotazy pro týdenní report

### Fáze 2: Dashboard (do 1 měsíce)

- [ ] Grafana / Metabase dashboard s MVP panely (primární KPI)
- [ ] Scheduled report do Slacku každé pondělí ráno s předchozím týdnem
- [ ] Alert rules pro error spikes

### Fáze 3: Deep analytics (až bude >50 MAU)

- [ ] Cohort retention analysis
- [ ] Per-plan-tier breakdown (free vs. paid users)
- [ ] A/B testing infrastructure pro budoucí verze skillu (testování jestli skill změna zlepší success rate)

---

## 📝 Dotazy / decisions pro backend tým

1. **Kde logovat?** — API gateway (Kong/nginx/Traefik log parsing) vs. aplikační middleware (kód v PHP/Node/cokoli). Doporučuju middleware pro strukturovaná data.

2. **Storage?** — Existující Postgres s time-series table? Nebo dedikovaná analytics DB (ClickHouse / BigQuery)?

3. **Sampling?** — Pro desítky users je 100 % sample OK. Pokud by se MAU vyšplhalo na tisíce, zvážit 10 % sample pro per-request events + plný sample pro agregáty.

4. **Tool pro dashboard?** — Zjistit co už Freelo používá (Metabase, Grafana, Amplitude, …) a navázat.

5. **Kdo je owner?** — Doporučuju jmenovat 1 backend dev jako maintainer tohohle dashboardu + týdenní report to stake holderům.

---

## 🚀 Timeline návrh

| Týden | Akce |
|---|---|
| **0** (launch skill) | Skill je public, User-Agent je v produkci, logs přicházejí |
| **1** | Basic logging + schema ready, první manuální SQL report |
| **2–4** | MVP Notion dashboard, týdenní review trendů |
| **měsíc 2** | Grafana/Metabase dashboard, automated weekly Slack report |
| **měsíc 3+** | Podle dat — A/B testing, cohort analysis, version adoption reports |

---

## 📎 Reference

- Skill repo: https://github.com/freeloio/claude-freelo-skill
- Primární UA pattern: `Freelo-Claude-Skill/<semver>`
- Kontakt na autora skillu: Marek Urban <marek.urban@freelo.io>
