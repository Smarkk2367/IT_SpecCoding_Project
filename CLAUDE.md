# CLAUDE.md — Instrukcje dla agenta

> 📝 DO WYPEŁNIENIA przez zespół.
> Ten plik agent czyta jako pierwszy przed każdą sesją.
> Im dokładniejszy — tym mniej agent pyta i zgaduje.

---

## Kim jesteś i co budujesz

```
Jesteś seniorem [WYPEŁNIJ: język i framework] implementującym TrackFlow —
system skracania i śledzenia linków dla agencji marketingowej.
```

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

```
Backend:
  Język:        [WYPEŁNIJ]
  Framework:    [WYPEŁNIJ]
  ORM:          [WYPEŁNIJ]

Frontend:
  Framework:    [WYPEŁNIJ]
  Stylowanie:   [WYPEŁNIJ]

Infrastruktura:
  Cache:        Redis
  Kolejka:      [WYPEŁNIJ — z ADR-004]
  Baza danych:  PostgreSQL
  E-mail (dev): Mailhog

Testy:
  Jednostkowe:  [WYPEŁNIJ]
  Integracyjne: [WYPEŁNIJ]
```

---

## Zasady których ZAWSZE przestrzegasz

**Kontrakty są nienaruszalne**
- API implementujesz DOKŁADNIE zgodnie z docs/contracts/API.md
- Payload eventów DOKŁADNIE zgodny z docs/contracts/EVENTS.md

**Redirect jest krytyczny**
- GET /:short_code musi odpowiedzieć w < 80ms
- Kolejność: sprawdź Redis → miss → sprawdź PG → zapisz do Redis → 302 → opublikuj event
- Publikacja eventu jest ASYNCHRONICZNA — nie blokuje 302

**At-least-once delivery**
- Consumer sprawdza event_id przed przetworzeniem
- ACK dopiero po zapisie do bazy

**Testy są obowiązkowe**
- Po każdym module uruchom testy
- Testy z WORKER.md sekcja "Testy które agent musi napisać" są obowiązkowe

---

## Kolejność implementacji

Po każdym kroku uruchom testy i zaraportuj.

```
Krok 1:  Inicjalizacja projektu, Docker Compose, Dockerfile(i), zmienne środowiskowe
Krok 2:  Schemat bazy danych + migracje
Krok 3:  Auth — login, JWT middleware
Krok 4:  Endpoint redirect GET /:short_code (z cache Redis)
Krok 5:  Publisher eventu click.recorded
Krok 6:  CRUD linków
Krok 7:  Consumer click.recorded (UA parser + geo + zapis)
Krok 8:  Endpointy statystyk
Krok 9:  Consumer report.requested + PDF
Krok 10: Consumer notification.send + e-mail
Krok 11: Cron weekly-report
Krok 12: Cron alert-no-clicks
Krok 13: Frontend — auth, dashboard, lista linków
Krok 14: Frontend — statystyki i wykresy
Krok 15: Frontend — raporty (polling statusu)
Krok 16: Testy integracyjne end-to-end
Krok 17: Weryfikacja docker-compose up
```

---

## Format raportowania

```
Krok N ukończony
  Zbudowałem: [1 zdanie]
  Testy: [X passed, Y failed]
  Do sprawdzenia przez zespół: [tak/nie + co]
```

---

## Weryfikacja redirectu

```bash
curl -o /dev/null -s -w "Total: %{time_total}s\n" http://localhost:3000/xK9mP
# Oczekiwane: < 0.080s
```

---

## Dane testowe

Utwórz seed który dodaje:
- 2 użytkowników: marketer@test.com i client@test.com (hasło: test123)
- 5 linków z różnymi krótkimi kodami
- 100 kliknięć z ostatnich 7 dni

---

## Dodatkowe instrukcje

> Wpisz tutaj co agent powinien wiedzieć a czego nie ma wyżej.

```
Przykłady:
- Limit linków per użytkownik: ___
- Kod krótki: ___ znaków, generowany przez ___
- Wygląd dashboardu: ___
- Nie używaj biblioteki X ponieważ ___
```

1.
2.
3.
