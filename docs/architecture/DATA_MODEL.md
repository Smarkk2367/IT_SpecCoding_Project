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
| client_id | uuid | NULL, FK → clients.id | ustawione dla użytkowników z rolą client |
| created_at | timestamptz | NOT NULL | |

---

## Tabela: clients

| Kolumna | Typ | Ograniczenia | Opis |
|---------|-----|--------------|------|
| id | uuid | PK | identyfikator klienta agencji |
| name | text | UNIQUE, NOT NULL | nazwa klienta |
| created_at | timestamptz | NOT NULL | czas utworzenia |

---

## Tabela: links


| Kolumna | Typ | Ograniczenia | Opis |
|---------|-----|--------------|------|
| id | uuid | PK | unikalny identyfikator linku |
| short_code | text | UNIQUE, NOT NULL | generowany kod np. xK9mP |
| original_url | text | NOT NULL | docelowy URL |
| created_by | uuid | FK → users.id | właściciel linku (marketer) |
| campaign_name | text | NULL | opcjonalna nazwa kampanii |
| client_id | uuid | NULL, FK → clients.id | klient agencji (multi-tenant separation) |
| expires_at | timestamptz | NULL | nie wygasa |
| is_active | boolean | NOT NULL DEFAULT true | szybka walidacja aktywności |
| created_at | timestamptz | NOT NULL | czas utworzenia |
| deleted_at | timestamptz | NULL | soft delete po DELETE /api/links/:id |

---

## Tabela: clicks

| Kolumna | Typ | Ograniczenia | Opis |
| id | bigserial | PK | szybki insert (time-series) |
| link_id | uuid | FK → links.id | referencja do linku |
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
| created_at | timestamptz | NOT NULL | czas zapisu rekordu |

---

## Tabela: reports


| Kolumna | Typ | Ograniczenia | Opis |
|---------|-----|--------------|------|
| id | uuid | PK | identyfikator raportu |
| status | text | NOT NULL, enum: pending/processing/done/failed | |
| requested_by | uuid | FK → users.id | marketer |
| client_id | uuid | NULL, FK → clients.id | filtr raportu per klient |
| date_from | timestamptz | NOT NULL | początek zakresu raportu |
| date_to | timestamptz | NOT NULL | koniec zakresu raportu |
| file_path | text | NULL | ścieżka lub URL do PDF gdy gotowy |
| error_message | text | NULL | |
| created_at | timestamptz | NOT NULL | |
| completed_at | timestamptz | NULL | |

---

## Tabela: report_links

| Kolumna | Typ | Ograniczenia | Opis |
|---------|-----|--------------|------|
| report_id | uuid | PK, FK → reports.id | raport |
| link_id | uuid | PK, FK → links.id | link ujęty w raporcie |
| created_at | timestamptz | NOT NULL | czas przypisania |

---

## Tabela: failed_events

| Kolumna | Typ | Ograniczenia | Opis |
|---------|-----|--------------|------|
| id | bigserial | PK | identyfikator rekordu DLQ |
| event_id | uuid | UNIQUE, NOT NULL | id eventu z koperty |
| event_type | text | NOT NULL | typ eventu |
| stream | text | NOT NULL | nazwa kolejki/streama |
| payload | jsonb | NOT NULL | pełny payload/koperta do reprocessingu |
| error_message | text | NOT NULL | ostatni błąd przetwarzania |
| failed_at | timestamptz | NOT NULL | czas przeniesienia do DLQ |
| reprocessed_at | timestamptz | NULL | czas skutecznego reprocessingu |

---

## Tabela: outbox_events

| Kolumna | Typ | Ograniczenia | Opis |
|---------|-----|--------------|------|
| id | bigserial | PK | lokalny bufor API gdy Redis publish się nie uda |
| event_id | uuid | UNIQUE, NOT NULL | id eventu z koperty |
| event_type | text | NOT NULL | typ eventu |
| stream | text | NOT NULL | docelowy Redis Stream |
| payload | jsonb | NOT NULL | pełna koperta eventu |
| status | text | NOT NULL, enum: pending/published/failed | status wysyłki |
| created_at | timestamptz | NOT NULL | czas zapisu do outboxa |
| published_at | timestamptz | NULL | czas skutecznej publikacji |
| last_error | text | NULL | ostatni błąd publikacji |

---

## Relacje

```
clients   1--* users       (klient agencji może mieć wielu użytkowników read-only)
clients   1--* links       (link może być przypisany do klienta)
users     1--* links       (marketer tworzy wiele linków)
links     1--* clicks      (link ma wiele kliknięć)
users     1--* reports     (marketer generuje raporty)
clients   1--* reports     (raport może dotyczyć klienta)
reports   *--* links       (report_links zapisuje link_ids z requestu)
```

---

## Co NIE idzie do PostgreSQL

| Co | Gdzie | Dlaczego nie w PG |
|----|-------|-------------------|
| Cache redirectu (short_code → URL) | Redis | wymagany sub-ms lookup w hot path redirectu |
| Kolejka kliknięć (event stream raw) | Redis Streams | buffer dla at-least-once delivery + odciążenie DB |
| Realtime liczniki dashboardu | Redis | uniknięcie agregacji SQL na milionach rekordów |
| Session / burst dedup buffer | Redis | szybka idempotencja i ochrona przed spam/bot traffic |
