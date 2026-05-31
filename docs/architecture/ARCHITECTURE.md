# Architektura systemu — TrackFlow

> 📝 DO WYPEŁNIENIA przez zespół przed uruchomieniem agenta.

---

## 1. Back-of-envelope math

Dane z briefu: 200 000 kliknięć/miesiąc dziś → 2 000 000 za rok

```
Kliknięć / dzień (dziś):           ~ 6700
Kliknięć / sekundę (dziś):         ~ 0.08
Kliknięć / dzień (za rok): ~ 66 700
Kliknięć / sekundę (za rok):       ~ 0.8
Rekordów w tabeli clicks po roku:   ~ 24 000
Szacowana wielkość tabeli clicks:  ~ 13/14 GB (500 B na klik)
Raporty PDF / tydzień:             teraz 40, za rok 200 (każdy dostaje 8.00 + max 15 min)
```
Wnioski:

```
Bottleneck #1 to baza ponieważ jednoczesne, częste inserty tworzą konflikty (contention) w bazie.
Bottleneck #2 to parsing danych jak UserAgent i GeoIp ponieważ obciąża to CPU.
Redirect NIE może iść do bazy ponieważ grozi to przebiciem limitu 80 ms.
Cache jest potrzebny dla short linków i trzymam w nim mapowanie - short -> url + metadata.
Zapis kliknięcia jest asynchroniczny ponieważ redirect musi być natychmiastowy, a zapisy do baz itp mogą mieć lekki delay.
```

---

## 2. C1 — Context Diagram


| Element | Typ | Co robi |
|---------|-----|---------|
| Marketer | Aktor | Tworzy linki, przegląda dane, generuje raporty |
| Klient agencji | Aktor | Otrzymuje raporty, udostępnia linki klikaczom |
| Osoba klikająca | Aktor | Klika w link |
| Email Provider | System zewnętrzny | geolokalizacja IP |
| Docelowa strona WWW | System zewnętrzny | wysyłanie e-maili |

---

## 3. C2 — Container Diagram

| Kontener | Technologia | Odpowiedzialność |
|----------|-------------|-----------------|
| Web API | Go + Fiber | Tworzenie linków, obsługa dashboardu, szybkie redirecty |
| Redis | Redis + Redis Streams | cache, kolejkowanie, statystyki |
| Worker |  Go Worker | Przetwarzanie kliknięć, GeoIP, User-Agent parsing, agregacje, alerty, generowanie raportów|
| PostgreSQL | PostgreSQL | Przetwarzanie linków, kliknięć i agregatów statystycznych |

```
Marketer komunikuje się z WebAPI przez HTTPS

WebAPI odczytuje dane z PostgreSQL

Gdy osoba klikająca klika w link, wysyłane jest żądanie do WebAPI, które pobiera informacje o linku z redis, i kliknięcie zostaje zapisane w redis Streams

Worker odczytuje zdarzenia z redis streams

Dashboard statystyk pobiera dane z Redis i Postgres

Worker cyklicznie generuje raporty PDF

Podczas generowania raportów i alertów Worker komunikuje się z Email Providerem

Worker korzysta z GeoIP Database/Service, aby na podstawie adresu IP określić

```

> Dlaczego Worker jest osobnym kontenerem?
> Wykonuje zadania asynchroniczne jak generowanie pdf czy zbieranie geoip, oddzielenie go od redirectu pozwala na obniżenie latency

> Dlaczego kliknięcie nie idzie od razu do bazy?
> Uzależniłoby to redirect od bazy (trzeba by czekać na zapis)
> Co trzymasz w cache i dlaczego?
> Mapowanie (shorturl - targeturl), Datę wygaśnięcia, ID kampanii i klienta, Licznik statystyk realtime (Redirect nie musi strzelać do postgresa - mniejsze latency) 

---

## 4. Przepływ — Redirect (< 80ms)

| Krok | Opis | Czas (ms) |
|------|------|-----------|
| 1 | odczyt shorturl z redisa i walidacja daty wygaśnięcia | ~ 2 ms |
| 2 | Zapis zdarzenia kliknięcia do Redis Streams | ~1 ms |
| 3 | Wysłanie odpowiedzi HTTP z docelowym URL| ~ 5 ms |
| **Suma** | | **~8 ms** |

```
Co przy cache miss:      Pobieram link z PostgreSQL, zapisuję go do Redis i obsługuję redirect, cache zostaje uzupełniony dla kolejnych żądań
Co gdy Redis jest down:  Fallback do PostgreSQL dla lookupu linku. Kliknięcie zapisuję lokalnie (np. persistent queue/outbox) i odtwarzam po odzyskaniu Redis. Redirect nadal działa, ale z większą latencją.
```

---

## 5. Przepływ — Przetwarzanie kliknięcia (max 5s)

| Krok | Opis | Kto |
|------|------|-----|
| 1 | | |
| 2 | | |
| 3 | | |
| 4 | | |

```
Co gwarantuje że dane nie zginą:   ____________________
Jak zapewniasz idempotentność:     ____________________
```

---

## 6. Przepływ — Generowanie raportu PDF

| Krok | Opis |
|------|------|
| 1 | Marketer klika "Generuj raport" |
| 2 | |
| 3 | |
| 4 | |

```
Dlaczego async:                    ____________________
Gdzie jest przechowywany PDF:      ____________________
Jak marketer dostaje info:         ____________________
```

---

## 7. Failure scenarios

| Komponent pada | Co robi system | Dane bezpieczne? |
|---------------|----------------|-----------------|
| Redis | | |
| Broker kolejki | | |
| Worker | | |
| PostgreSQL | | |
| API geo | | |

---

## 8. Indeksy bazy danych

| Tabela | Kolumna(y) | Uzasadnienie |
|--------|-----------|--------------|
| | | |
| | | |
