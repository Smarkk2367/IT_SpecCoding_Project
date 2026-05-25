# Kontrakt eventów — kolejka wiadomości

> 📝 DO WYPEŁNIENIA
> Agent implementuje publisher i consumer na podstawie tego dokumentu.
> Event nie opisany tutaj = nie istnieje.

---

## Konfiguracja

> Wypełnij zgodnie z brokerem z ADR-004.

```
Broker:   [RabbitMQ / BullMQ / inne]

[Jeśli RabbitMQ:]
Exchange:  trackflow.events  (typ: topic)
DLQ:       trackflow.dead
Kolejki:
  trackflow.clicks
  trackflow.reports
  trackflow.notifications

[Jeśli BullMQ:]
Queues: clicks / reports / notifications
```

---

## Format koperty (envelope)

```json
{
  "event_id":   "uuid",
  "event_type": "click.recorded",
  "version":    "1.0",
  "timestamp":  "ISO8601",
  "payload":    {}
}
```

---

## EVENT: click.recorded

**Publisher:** API Server (endpoint GET /:short_code)
**Consumer:** Worker
**Kiedy:** po wysłaniu odpowiedzi 302, asynchronicznie

**Gwarancja:** at-least-once
**Idempotency:** ten sam event_id może przyjść dwa razy — consumer musi to obsłużyć

**Payload:**
```json
{
  "link_id":    "uuid",
  "short_code": "xK9mP",
  "clicked_at": "ISO8601",
  "ip_address": "192.168.1.0",
  "user_agent": "Mozilla/5.0...",
  "referrer":   "string | null"
}
```

**Co robi consumer:**
```
1. Sprawdź czy event_id istnieje w tabeli clicks (idempotency check)
   → tak: ACK i zakończ
2. Parsuj user_agent → device_type, browser, os
3. Geolokalizuj ip_address → country, city
4. Zapisz do tabeli clicks
5. ACK

Przy błędzie geo/UA: zapisz null dla tych pól, nie failuj eventu
Przy błędzie zapisu: NACK → retry
```

**Retry:** 3 próby, backoff: 1s → 5s → 30s → DLQ

---

## EVENT: report.requested

**Publisher:** API Server (POST /api/reports)
**Consumer:** Worker

**Payload:**
```json
{
  "report_id":    "uuid",
  "requested_by": "uuid",
  "date_from":    "ISO8601",
  "date_to":      "ISO8601",
  [WYPEŁNIJ inne parametry]
}
```

**Co robi consumer:**
```
1. Zaktualizuj reports.status = 'processing'
2. [WYPEŁNIJ]
3. [WYPEŁNIJ]
4. Zaktualizuj reports.status = 'done', file_path = '...'

Przy błędzie: status = 'failed', error_message = '...'
```

---

## EVENT: notification.send

**Publisher:** Worker
**Consumer:** Worker (osobna kolejka)

**Payload:**
```json
{
  "type":            "report_ready | alert_no_clicks | weekly_report",
  "recipient_email": "string",
  "subject":         "string",
  [WYPEŁNIJ dane do szablonu e-maila]
}
```

---

## Zadania cykliczne (cron)

### weekly-report
```
Harmonogram:  0 8 * * 1  (poniedziałek 8:00)
Tolerancja:   max 15 minut

Co robi:
1. [WYPEŁNIJ]
2. [WYPEŁNIJ]
```

### alert-no-clicks
```
Harmonogram:  */15 * * * *  (co 15 minut)

Co robi:
1. Pobierz aktywne linki kampanii
2. Sprawdź ostatnie kliknięcie > 24h temu
3. [WYPEŁNIJ]

Deduplikacja: [WYPEŁNIJ — jak nie wysłać 100 e-maili dla tego samego linku?]
```

---

## Dead-letter queue

```
Kto monitoruje:    [WYPEŁNIJ]
Co się dzieje:     [WYPEŁNIJ]
Możliwy reprocess: [WYPEŁNIJ]
```
