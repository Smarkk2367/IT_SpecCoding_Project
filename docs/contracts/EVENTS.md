# Kontrakt eventów — kolejka wiadomości (TrackFlow)

## Konfiguracja

Broker: Redis Streams (BullMQ)

Queues:
  clicks
  reports
  notifications

---

## Format koperty (envelope)

{
  "event_id": "uuid",
  "event_type": "string",
  "version": "1.0",
  "timestamp": "ISO8601",
  "payload": {}
}

---

## EVENT: click.recorded

Publisher: API Server (GET /:short_code)  
Consumer: Worker  
Kiedy: po zwróceniu 302 (async)

Gwarancja: at-least-once  
Idempotency: event_id + UNIQUE constraint w DB

Payload:
{
  "link_id": "uuid",
  "short_code": "xK9mP",
  "clicked_at": "ISO8601",
  "ip_address": "192.168.1.0",
  "user_agent": "Mozilla/5.0...",
  "referrer": "string | null"
}

Co robi consumer:
1. Sprawdź event_id w tabeli clicks
   → jeśli istnieje: ACK i STOP
2. Parse user_agent → device_type, browser, os
3. GeoIP lookup → country, city
4. INSERT do clicks
5. UPDATE Redis counters (realtime stats)
6. ACK

Retry: 3 próby
Backoff: 1s → 5s → 30s → DLQ

---

## EVENT: report.requested

Publisher: API Server (POST /api/reports)  
Consumer: Worker

Payload:
{
  "report_id": "uuid",
  "requested_by": "uuid",
  "date_from": "ISO8601",
  "date_to": "ISO8601",
  "client_id": "uuid | null",
  "link_ids": ["uuid"]
}

Co robi consumer:
1. UPDATE reports.status = 'processing'
2. Pobierz dane (clicks + links + aggregates)
3. Zbuduj dataset raportu
4. Wygeneruj PDF
5. Zapisz file_path
6. UPDATE reports.status = 'done'

Przy błędzie:
status = 'failed'
error_message = reason

---

## EVENT: notification.send

Publisher: Worker  
Consumer: Worker (notification pipeline)

Payload:
{
  "type": "report_ready | alert_no_clicks | weekly_report",
  "recipient_email": "string",
  "subject": "string",
  "template_data": {
    "report_id": "uuid | null",
    "link_id": "uuid | null",
    "message": "string | null"
  }
}

---

## Zadania cykliczne (cron)

### weekly-report
Harmonogram: 0 8 * * 1
Tolerancja: max 15 minut

Co robi:
1. Pobierz wszystkich klientów
2. Dla każdego wygeneruj report.requested event
3. Enqueue do reports queue

---

### alert-no-clicks
Harmonogram: */15 * * * *

Co robi:
1. Pobierz aktywne linki kampanii
2. Sprawdź last_click_at > 24h
3. Jeśli spełnione → notification.send (alert_no_clicks)

Deduplikacja:
Redis key: alert_sent:{link_id}
TTL: 24h
Jeśli exists → skip
Jeśli not → send + set key

---

## Dead-letter queue

Kto monitoruje:
Worker + alerting job

Co się dzieje:
- zapis do failed_events table
- log error + metadata eventu
- alert (email/Slack)

Możliwy reprocess:
TAK:
- POST /api/events/reprocess/:event_id
- lub batch requeue DLQ → main queue