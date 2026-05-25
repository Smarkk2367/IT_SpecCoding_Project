# Kontrakt REST API — TrackFlow

> 📝 DO WYPEŁNIENIA
> Agent implementuje endpointy DOKŁADNIE według tego dokumentu.
> Brakujący endpoint = agent go nie zbuduje.

---

## Konwencje

```
Autentykacja:  Bearer JWT w headerze Authorization
Format:        JSON
Błędy:         { "code": "ERROR_CODE", "message": "opis" }
Paginacja:     ?page=1&limit=20 → { data: [], total: N, page: N }
```

---

## AUTH

### POST /auth/login

**Request:**
```json
{ "email": "string", "password": "string" }
```

**Response 200:**
```json
{ "token": "JWT", "user": { "id": "uuid", "email": "string", "role": "marketer|client" } }
```

**Response 401:** `{ "code": "INVALID_CREDENTIALS" }`

---

## REDIRECT

### GET /:short_code

> Krytyczny endpoint — musi odpowiedzieć w < 80ms.
> Publikuje event kliknięcia do kolejki ASYNCHRONICZNIE — nie blokuje redirectu.

**Auth:** Brak (publiczny)

**Response 302:** Header `Location: <original_url>`

**Response 404:** Link nie istnieje lub wygasł

---

## LINKS

### GET /api/links

**Auth:** Marketer

**Query params:** `page`, `limit` [WYPEŁNIJ inne filtry]

**Response 200:**
```json
{
  "data": [{ "id": "uuid", "short_code": "xK9mP", "original_url": "https://...", [WYPEŁNIJ] }],
  "total": 42,
  "page": 1
}
```

---

### POST /api/links

**Auth:** Marketer

**Request:**
```json
{ "original_url": "string (URL)", [WYPEŁNIJ opcjonalne pola] }
```

**Response 201:** `{ [WYPEŁNIJ pełny obiekt linku] }`

**Response 400:** Błąd walidacji

---

### GET /api/links/:id

**Auth:** Marketer

**Response 200:** `{ [WYPEŁNIJ] }`

**Response 404:** Link nie istnieje

---

### DELETE /api/links/:id

**Auth:** Marketer

**Response 204:** Brak body

---

## STATYSTYKI

### GET /api/links/:id/stats

**Auth:** Marketer lub Client

**Query params:**
```
period:    "hour" | "day" | "week"
date_from: ISO 8601 (opcjonalny)
date_to:   ISO 8601 (opcjonalny)
```

**Response 200:**
```json
{
  "total_clicks": 1234,
  "unique_clicks": 890,
  "clicks_over_time": [{ "timestamp": "ISO8601", "count": 45 }],
  "by_country": [{ "country": "PL", "count": 500 }],
  "by_device": [{ "device_type": "mobile", "count": 700 }],
  "by_referrer": [{ "referrer": "instagram.com", "count": 300 }]
}
```

---

## RAPORTY

### POST /api/reports

> Async — zwraca 202 natychmiast. PDF generuje się w tle.

**Auth:** Marketer

**Request:**
```json
{ [WYPEŁNIJ parametry: zakres dat, które linki/kampanie] }
```

**Response 202:**
```json
{ "report_id": "uuid", "status": "pending" }
```

---

### GET /api/reports/:id

**Auth:** Marketer

**Response 200:**
```json
{
  "id": "uuid",
  "status": "pending | processing | done | failed",
  "download_url": "string | null",
  "error_message": "string | null",
  "created_at": "ISO8601",
  "completed_at": "ISO8601 | null"
}
```

> Frontend polluje co 3 sekundy gdy status != done.

---

## [WYPEŁNIJ brakujące endpointy]

> Przemyśl: lista raportów, lista klientów agencji, zmiana hasła.
