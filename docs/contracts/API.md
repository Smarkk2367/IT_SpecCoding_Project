# Kontrakt REST API — TrackFlow

## Konwencje
Autentykacja:  Bearer JWT w headerze Authorization
Format:        JSON
Błędy:         { "code": "ERROR_CODE", "message": "opis" }
Paginacja:     ?page=1&limit=20 → { data: [], total: N, page: N }

---

# AUTH

## POST /auth/login
Request:
{ "email": "string", "password": "string" }

Response 200:
{ "token": "JWT", "user": { "id": "uuid", "email": "string", "role": "marketer|client" } }

Response 401:
{ "code": "INVALID_CREDENTIALS" }

---

## POST /auth/change-password
Auth: JWT

Request:
{
  "old_password": "string",
  "new_password": "string"
}

Response 204

---

# REDIRECT

## GET /:short_code
Auth: none

Response 302:
Location: <original_url>

Response 404:
{ "code": "LINK_NOT_FOUND" }

---

# LINKS

## GET /api/links
Auth: Marketer

Query:
page, limit, client_id (optional), campaign_name (optional), active (optional)

Response 200:
{
  "data": [
    {
      "id": "uuid",
      "short_code": "xK9mP",
      "original_url": "https://...",
      "campaign_name": "string",
      "client_id": "uuid",
      "expires_at": "ISO8601 | null",
      "created_at": "ISO8601"
    }
  ],
  "total": 42,
  "page": 1
}

---

## POST /api/links
Auth: Marketer

Request:
{
  "original_url": "string",
  "campaign_name": "string (optional)",
  "client_id": "uuid (optional)",
  "expires_at": "ISO8601 (optional)"
}

Response 201:
{
  "id": "uuid",
  "short_code": "xK9mP",
  "original_url": "string",
  "campaign_name": "string",
  "client_id": "uuid",
  "expires_at": "ISO8601 | null",
  "created_at": "ISO8601"
}

---

## GET /api/links/:id
Auth: Marketer

Response 200:
{
  "id": "uuid",
  "short_code": "xK9mP",
  "original_url": "string",
  "campaign_name": "string",
  "client_id": "uuid",
  "expires_at": "ISO8601 | null",
  "created_at": "ISO8601"
}

---

## DELETE /api/links/:id
Auth: Marketer

Response 204

---

# STATYSTYKI

## GET /api/links/:id/stats
Auth: Marketer | Client

Query:
period=hour|day|week
date_from (optional)
date_to (optional)
group_by=time|country|device|referrer

Response 200:
{
  "total_clicks": 1234,
  "unique_clicks": 890,
  "clicks_over_time": [
    { "timestamp": "ISO8601", "count": 45 }
  ],
  "by_country": [
    { "country": "PL", "count": 500 }
  ],
  "by_device": [
    { "device_type": "mobile", "count": 700 }
  ],
  "by_referrer": [
    { "referrer": "instagram.com", "count": 300 }
  ]
}

---

# RAPORTY

## POST /api/reports
Auth: Marketer

Request:
{
  "date_from": "ISO8601",
  "date_to": "ISO8601",
  "client_id": "uuid (optional)",
  "link_ids": ["uuid"]
}

Response 202:
{ "report_id": "uuid", "status": "pending" }

---

## GET /api/reports
Auth: Marketer

Query:
page, limit, status (optional)

Response 200:
{
  "data": [
    {
      "id": "uuid",
      "status": "done",
      "created_at": "ISO8601",
      "completed_at": "ISO8601 | null"
    }
  ],
  "total": 10,
  "page": 1
}

---

## GET /api/reports/:id
Auth: Marketer

Response 200:
{
  "id": "uuid",
  "status": "pending | processing | done | failed",
  "download_url": "string | null",
  "error_message": "string | null",
  "created_at": "ISO8601",
  "completed_at": "ISO8601 | null"
}

---

# CLIENTS

## GET /api/clients
Auth: Marketer

Response 200:
{
  "data": [
    {
      "id": "uuid",
      "name": "string",
      "created_at": "ISO8601"
    }
  ]
}

---

## POST /api/clients
Auth: Marketer

Request:
{ "name": "string" }

Response 201:
{ "id": "uuid", "name": "string", "created_at": "ISO8601" }

---

# USER

## GET /api/me
Auth: JWT

Response 200:
{
  "id": "uuid",
  "email": "string",
  "role": "marketer|client"
}

---

## PATCH /api/me/password
Auth: JWT

Request:
{
  "old_password": "string",
  "new_password": "string"
}

Response 204