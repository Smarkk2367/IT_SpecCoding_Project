# Kontrakt Workera — TrackFlow

## Odpowiedzialności

- [x] Consumer: click.recorded
- [x] Consumer: report.requested
- [x] Consumer: notification.send
- [x] Cron: weekly-report (poniedziałek 8:00)
- [x] Cron: alert-no-clicks (co 15 minut)
- [ ] Retry/DLQ processor
- [ ] Event reprocessing (manual trigger / admin endpoint worker-side handler)

---

## Integracje

### Geolokalizacja IP
Biblioteka: geoip-lite (lokalna DB) lub ip-api.com (fallback HTTP)

Timeout: max 100ms

Przy timeout:
- country: null
- city: null
- NIE failuj eventu

---

### Parser User-Agent
Biblioteka: ua-parser-js

Pola:
- device_type: mobile | desktop | tablet | null
- browser: string | null
- os: string | null

---

### Generowanie PDF
Biblioteka: Puppeteer (HTML → PDF render)

Gdzie zapisujesz:
- filesystem (VPS) + opcjonalnie MinIO (S3-compatible)

Format nazwy:
report_{report_id}.pdf

Po wygenerowaniu:
1. zapis file_path w tabeli reports
2. update reports.status = 'done'
3. publish event: notification.send (report_ready)

---

### Wysyłanie e-maili
Dev: Mailhog (SMTP sandbox, UI http://localhost:8025)  
Prod: Nodemailer + SMTP (np. Amazon SES / Resend / SendGrid)

From: noreply@trackflow.io

---

## Zmienne środowiskowe

```env
DATABASE_URL=
REDIS_URL=

SMTP_HOST=
SMTP_PORT=
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@trackflow.io

PDF_STORAGE_PATH=/data/reports

GEOIP_PROVIDER=geoip-lite

APP_ENV=production