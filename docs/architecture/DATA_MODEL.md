# Model danych — TrackFlow

> 📝 DO WYPEŁNIENIA
> Agent wygeneruje schemat bazy na podstawie tego dokumentu.
> Bądź precyzyjny — każde pole, każdy typ, każde ograniczenie.

---

## Zasady
- Każda tabela ma `id` (UUID lub auto-increment — uzasadnij), `created_at`
- Soft delete gdzie potrzebny: dodaj `deleted_at`
- Indeksy opisujesz w ARCHITECTURE.md sekcja 8

---

## Tabela: users

| Kolumna | Typ | Ograniczenia | Opis |
|---------|-----|--------------|------|
| id | uuid | PK | |
| email | text | UNIQUE, NOT NULL | |
| password_hash | text | NOT NULL | |
| role | text | NOT NULL, enum: marketer/client | |
| created_at | timestamptz | NOT NULL | |

---

## Tabela: links


| Kolumna | Typ | Ograniczenia | Opis |
|---------|-----|--------------|------|
| id | uuid | PK | unikalny identyfikator linku |
| short_code | text | UNIQUE, NOT NULL | generowany kod np. xK9mP |
| original_url | text | NOT NULL | docelowy URL |
| created_by | uuid | FK → users.id | właściciel linku (marketer) |
| campaign_name | text | NULL | opcjonalna nazwa kampanii |
| client_id | uuid | NULL | klient agencji (multi-tenant separation) |
| expires_at | timestamptz | NULL | nie wygasa |
| is_active | boolean | NOT NULL DEFAULT true | szybka walidacja aktywności |
| created_at | timestamptz | NOT NULL | czas utworzenia |

---

## Tabela: clicks

| Kolumna | Typ | Ograniczenia | Opis |
| id | bigserial | PK | szybki insert (time-series) |
| link_id	uuid | FK → links.id | referencja do linku |
| clicked_at | timestamptz | NOT NULL | czas kliknięcia |
| country | text | NULL | GeoIP |
| city | text | NULL | GeoIP |
| device_type | text | NULL | enum: mobile/desktop/tablet |
| browser | text | NULL | |
| os | text | NULL | |	
| referrer | text | NULL | źródło ruchu |
| ip_hash | text | NULL | zanonimizowane IP |
| event_id| uuid | UNIQUE, NOT NULL | idempotency key |
| user_agent | text | NULL | raw UA (do późniejszej analizy) |

---

## Tabela: reports


| Kolumna | Typ | Ograniczenia | Opis |
|---------|-----|--------------|------|
| id | uuid | PK | identyfikator raportu |
| status | text | NOT NULL, enum: pending/processing/done/failed | |
| requested_by | uuid | FK → users.id | marketer |
| file_path | uuid | NULL | ścieżka do PDF gdy gotowy |
| error_message | text | NULL | |
| created_at | timestampz | NOT NULL | |
| completed_at | timestampz | NULL | |

---

## Relacje

```
users     1--* links       (marketer tworzy wiele linków)
links     1--* clicks      (link ma wiele kliknięć)
users     1--* reports     (marketer generuje raporty)
links     *--1 users       (created_by)
```

---

## Co NIE idzie do PostgreSQL

| Co | Gdzie | Dlaczego nie w PG |
|----|-------|-------------------|
| Cache redirectu (short_code → URL) | Redis | wymagany sub-ms lookup w hot path redirectu |
| Kolejka kliknięć (event stream raw) | Redis Streams | buffer dla at-least-once delivery + odciążenie DB |
| Realtime liczniki dashboardu | Redis | uniknięcie agregacji SQL na milionach rekordów |
| Session / burst dedup buffer | Redis | szybka idempotencja i ochrona przed spam/bot traffic |
