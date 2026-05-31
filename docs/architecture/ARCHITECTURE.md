# Architektura systemu — TrackFlow

> 📝 DO WYPEŁNIENIA przez zespół przed uruchomieniem agenta.

---

## 1. Back-of-envelope math

Dane z briefu: 200 000 kliknięć/miesiąc dziś → 2 000 000 za rok

```
Kliknięć / dzień (dziś):           ~ 6700
Kliknięć / sekundę (dziś):         ~ 0.08
Kliknięć / dzień (za rok):        ~ 66 700
Kliknięć / sekundę (za rok):       ~ 0.8
Rekordów w tabeli clicks po roku:  ~ 24 000 000
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
| Email Provider | System zewnętrzny | wysyłanie e-maili |
| GeoIP Database/Service | System zewnętrzny | geolokalizacja IP |
| Docelowa strona WWW | System zewnętrzny | strona docelowa redirectu |

---

## 3. C2 — Container Diagram

| Kontener | Technologia | Odpowiedzialność |
|----------|-------------|-----------------|
| Web API | Go + net/http + chi | Tworzenie linków, obsługa dashboardu, szybkie redirecty |
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
| 1 | Redirect Service zapisuje zdarzenie kliknięcia do Redis Streams | Redirect Service |
| 2 | Worker odczytuje zdarzenie z kolejki i wykonuje GeoIP oraz parsowanie User-Agent | Worker |
| 3 | Worker zapisuje pełny rekord kliknięcia do PostgreSQL oraz aktualizuje agregaty statystyk | Worker |
| 4 | Dashboard odczytuje zaktualizowane statystyki z Redis/PostgreSQL; kliknięcie staje się widoczne dla marketera | Web Api|

```
Co gwarantuje że dane nie zginą:   Redis Streams z włączoną persystencją oraz zapis kliknięcia do PostgreSQL
Jak zapewniasz idempotentność:     Każde kliknięcie otrzymuje unikalny event_id, przed zapisem Worker sprawdza, czy event_id został już przetworzony
```

---

## 6. Przepływ — Generowanie raportu PDF

| Krok | Opis |
|------|------|
| 1 | Marketer klika "Generuj raport" |
| 2 | Web API tworzy job raportowy i zapisuje go do kolejki (Redis Stream / queue) |
| 3 | Worker pobiera job, agreguje dane (PostgreSQL + Redis), generuje PDF |
| 4 | 	PDF jest zapisywany i udostępniany (storage + link), a status joba aktualizowany |

```
Dlaczego async:          Bo generowanie raportu jest kosztowne i nie może blokować requestu HTTP ani wpływać na SLA redirectów / dashboardu
Gdzie jest przechowywany PDF:     W obiekcie storage, a w PostgreSQL trzymana jest tylko metadana i link do pliku
Jak marketer dostaje info:        Dashboard polling/WebSocket/webhook status update + e-mail z linkiem do pobrania PDF po zakończeniu generowania
```

---

## 7. Failure scenarios

| Komponent pada | Co robi system | Dane bezpieczne? |
|---------------|----------------|-----------------|
| Redis | Redirect przełącza się na fallback (PostgreSQL lookup lub lokalny cache), zapis eventów trafia do outbox/persistent queue | Tak |
| Broker kolejki | API zapisuje eventy do lokalnego outboxu (PostgreSQL lub disk queue), worker po powrocie przetwarza backlog | Tak |
| Worker | Eventy pozostają w kolejce (Redis Streams / retry), po restarcie worker kontynuuje od ostatniego offsetu | Tak |
| PostgreSQL | System przechodzi w tryb degraded: redirect działa (cache), eventy buforowane w Redis/outbox; brak możliwości trwałego zapisu agregatów | Tak |
| API geo | Worker używa fallback cache / batch enrichment później; kliknięcia trafiają bez enrichu i są uzupełniane asynchronicznie | Tak |

---

## 8. Indeksy bazy danych

| Tabela | Kolumna(y) | Uzasadnienie |
|--------|-----------|--------------|
| links | short_code (UNIQUE) | Krytyczne dla redirectu — O(1) lookup skróconego linku do URL docelowego |
| links | client_id, campaign_name | Szybkie filtrowanie linków per klient/kampania w dashboardzie |
| links | expires_at| Skanowanie i wygaszanie linków (cleanup joby / walidacja aktywności)|
| clicks | link_id, created_at | Podstawowy wzorzec zapytań statystycznych (time-series per link) |
| clicks | country | Top countries bez pełnego skanowania tabeli |
| clicks | device_type | Top devices |
| clicks | referrer | Analiza źródeł ruchu |
| clicks | event_id (UNIQUE)| idempotencja |
| reports | status, created_at | Polling listy raportów i filtrowanie po statusie |
| report_links | link_id | Wyszukiwanie raportów obejmujących konkretny link |
| failed_events | event_id (UNIQUE) | Idempotencja zapisu DLQ i reprocessing |
| outbox_events | status, created_at | Requeue lokalnych eventów, gdy Redis był chwilowo niedostępny |
| outbox_events | event_id (UNIQUE) | Idempotencja lokalnego bufora publishera |

---

## 9. Ujednolicenia implementacyjne

- Backend API implementujemy w Go na `net/http` + `chi`. Wzmianka o Fiber była niespójna ze stackiem projektu i nie obowiązuje.
- Worker implementujemy w Go jako osobny kontener. Nie używamy Node.js jako drugiego runtime dla workerów w v1.
- Biblioteki wskazane w WORKER.md należy czytać jako wymagania funkcjonalne, nie jako narzucony ekosystem JS.
- Tabela `clients` jest wymagana, bo istnieją endpointy `/api/clients`, raporty per klient oraz separacja danych klientów.
- `reports.file_path` jest typem `text`, bo przechowuje ścieżkę lub URL do PDF, nie UUID.
