# Kontrakt Workera

> 📝 DO WYPEŁNIENIA
> Worker to osobny proces który konsumuje eventy i uruchamia crony.
> NIE obsługuje requestów HTTP.

---

## Odpowiedzialności

- [x] Consumer: click.recorded
- [x] Consumer: report.requested
- [x] Consumer: notification.send
- [x] Cron: weekly-report (poniedziałek 8:00)
- [x] Cron: alert-no-clicks (co 15 minut)
- [ ] [WYPEŁNIJ inne]

---

## Integracje

### Geolokalizacja IP
```
Biblioteka:  [WYPEŁNIJ — np. geoip-lite, ip-api.com]
Timeout:     max ___ms
Przy timeout: zapisz null, nie failuj eventu
```

### Parser User-Agent
```
Biblioteka:  [WYPEŁNIJ — np. ua-parser-js]
Pola:        device_type (mobile/desktop/tablet), browser, os
```

### Generowanie PDF
```
Biblioteka:       [WYPEŁNIJ — np. Puppeteer, WeasyPrint]
Gdzie zapisujesz: [WYPEŁNIJ — filesystem / S3-compatible]
Format nazwy:     [WYPEŁNIJ — np. report_{id}.pdf]
Po wygenerowaniu: [WYPEŁNIJ — co aktualizujesz, co publikujesz]
```

### Wysyłanie e-maili
```
Dev:   Mailhog — lokalny SMTP, UI: http://localhost:8025
Prod:  [WYPEŁNIJ — nodemailer+SMTP, SendGrid, Resend]
From:  noreply@trackflow.io
```

---

## Zmienne środowiskowe
```env
DATABASE_URL=
[RABBITMQ_URL lub REDIS_URL — z ADR-004]
SMTP_HOST=
SMTP_PORT=
SMTP_FROM=noreply@trackflow.io
PDF_STORAGE_PATH=
[WYPEŁNIJ inne]
```

---

## Testy które agent musi napisać

### Jednostkowe
- [ ] Parser UA: iPhone → device_type: "mobile"
- [ ] Parser UA: nieznany → null, nie rzuca wyjątku
- [ ] Idempotency: drugi event z tym samym event_id jest ignorowany
- [ ] Geolokalizacja: timeout → { country: null, city: null }

### Integracyjne
- [ ] click.recorded → rekord w tabeli clicks
- [ ] Ten sam event_id dwa razy → jeden rekord
- [ ] report.requested → plik PDF istnieje + reports.status = done
- [ ] weekly-report → e-mail w Mailhogu
