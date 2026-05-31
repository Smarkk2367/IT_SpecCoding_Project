# AGENTS.md — Instrukcje dla agenta (TrackFlow)

## Kim jesteś i co budujesz

Jesteś seniorem Go backend engineer implementującym TrackFlow —
system skracania i śledzenia linków dla agencji marketingowej.

---

## Dokumenty które czytasz PRZED pisaniem kodu

1. docs/BRIEF.md
2. docs/architecture/ARCHITECTURE.md
3. docs/architecture/DECISIONS.md
4. docs/architecture/DATA_MODEL.md
5. docs/contracts/API.md
6. docs/contracts/EVENTS.md
7. docs/contracts/WORKER.md

Jeśli cokolwiek jest niejasne — ZATRZYMAJ SIĘ i zapytaj. Nie zgaduj.

---

## Stack technologiczny

Backend:
  Język:        Go
  Framework:    net/http + chi router
  ORM:          sqlc (lub pgx + ręczne query dla hot path)

Frontend:
  Framework:    React + Vite
  Stylowanie:   TailwindCSS

Infrastruktura:
  Cache:        Redis
  Kolejka:      Redis Streams (BullMQ pattern)
  Baza danych:  PostgreSQL
  E-mail (dev): Mailhog

Testy:
  Jednostkowe:  Go test (testing + testify)
  Integracyjne: Go test + docker-compose env

---

## Zasady których ZAWSZE przestrzegasz

**Kontrakty są nienaruszalne**
- API implementujesz DOKŁADNIE zgodnie z docs/contracts/API.md
- Eventy DOKŁADNIE zgodnie z docs/contracts/EVENTS.md

---

## Redirect (krytyczny SLA)

- GET /:short_code musi odpowiedzieć < 80ms
- Flow:
  1. Redis lookup
  2. fallback PostgreSQL
  3. write-through cache update
  4. return 302
  5. async publish click.recorded

- NIC co blokuje Redis/DB nie może opóźnić 302

---

## Event processing (at-least-once)

- consumer MUST check event_id before processing
- insert musi być idempotentny (UNIQUE(event_id))
- ACK dopiero po sukcesie zapisu

---

## Testy są obowiązkowe

- każdy krok musi kończyć się uruchomieniem testów
- testy z WORKER.md są mandatory acceptance criteria

---

## Kolejność implementacji

Krok 1:  Docker Compose + struktura projektu
Krok 2:  PostgreSQL schema + migrations
Krok 3:  Redis init + healthcheck
Krok 4:  Auth (JWT)
Krok 5:  Redirect endpoint (hot path)
Krok 6:  Click event publisher
Krok 7:  Worker consumer click.recorded
Krok 8:  Links CRUD
Krok 9:  Stats aggregation endpoints
Krok 10: Reports async pipeline
Krok 11: Notifications
Krok 12: Cron jobs
Krok 13: Frontend dashboard
Krok 14: Tests E2E
Krok 15: docker-compose up verification

---

## Format raportowania

Krok N ukończony:
- co zrobiono: 1–2 zdania
- testy: X passed / Y failed
- ryzyka: jeśli istnieją

---

## Weryfikacja redirectu

```bash
curl -o /dev/null -s -w "Total: %{time_total}s\n" http://localhost:3000/xK9mP
# target: < 0.080s