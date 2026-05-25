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

> 📝 Wypełnij wszystkie pola.

| Kolumna | Typ | Ograniczenia | Opis |
|---------|-----|--------------|------|
| id | | | |
| short_code | | UNIQUE, NOT NULL | generowany kod np. xK9mP |
| original_url | | NOT NULL | |
| created_by | | FK → users.id | |
| expires_at | | NULL = nie wygasa | |
| created_at | | | |
| [WYPEŁNIJ inne] | | | |

---

## Tabela: clicks

> 📝 Wypełnij. Ta tabela urośnie do milionów rekordów — typy danych mają znaczenie.

| Kolumna | Typ | Ograniczenia | Opis |
|---------|-----|--------------|------|
| id | | | |
| link_id | | FK → links.id | |
| clicked_at | | NOT NULL | czas kliknięcia |
| country | | NULL | z geolokalizacji |
| city | | NULL | |
| device_type | | NULL, enum: mobile/desktop/tablet | |
| browser | | NULL | |
| os | | NULL | |
| referrer | | NULL | |
| ip_hash | | NULL | zanonimizowane IP |
| event_id | | UNIQUE, NOT NULL | idempotency key z eventu |

---

## Tabela: reports

> 📝 Wypełnij.

| Kolumna | Typ | Ograniczenia | Opis |
|---------|-----|--------------|------|
| id | | | |
| status | | NOT NULL, enum: pending/processing/done/failed | |
| requested_by | | FK → users.id | |
| file_path | | NULL | ścieżka do PDF gdy gotowy |
| error_message | | NULL | |
| created_at | | | |
| completed_at | | NULL | |

---

## Relacje

```
users     1--* links       (marketer tworzy wiele linków)
links     1--* clicks      (link ma wiele kliknięć)
[UZUPEŁNIJ]
```

---

## Co NIE idzie do PostgreSQL

| Co | Gdzie | Dlaczego nie w PG |
|----|-------|-------------------|
| Cache redirectu (short_code → URL) | Redis | |
| [UZUPEŁNIJ] | | |
