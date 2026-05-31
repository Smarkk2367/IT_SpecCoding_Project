# CHECKLIST — przed uruchomieniem agenta

> Sprawdź każdy punkt. Niezaznaczony = agent zgadnie źle.

---

## Architektura

- [x] Back-of-envelope: obliczenia i wnioski wypełnione
- [x] C1: wszyscy aktorzy, systemy zewnętrzne z etykietami strzałek
- [x] C2: każdy kontener ma technologię i odpowiedzialność
- [x] C2: każde połączenie ma protokół
- [x] C2: Worker jest osobnym kontenerem
- [x] Przepływ redirect: tabela z czasami, suma < 80ms
- [x] Przepływ kliknięcia: kroki + idempotency
- [x] Przepływ PDF: kroki + gdzie plik + jak user dostaje info
- [x] Failure scenarios: tabela wypełniona dla 5 komponentów
- [x] Indeksy: tabela z uzasadnieniami

## ADR

- [ ] ADR-001 (backend): min. 2 opcje, uzasadnienie do briefu
- [ ] ADR-002 (baza): min. 2 opcje
- [ ] ADR-003 (cache): odwołanie do liczb z back-of-envelope
- [ ] ADR-004 (kolejka): uzasadnienie "dane nie mogą zginąć"
- [ ] ADR-005+: każda inna istotna decyzja

## Model danych

- [ ] Tabela links: wszystkie pola z typami
- [ ] Tabela clicks: pole event_id (idempotency) jest
- [ ] Tabela reports: statusy są
- [ ] Relacje opisane
- [ ] Sekcja Redis wypełniona

## Kontrakty

- [ ] API.md: endpoint redirect opisany
- [ ] API.md: endpoint stats z wszystkimi polami response
- [ ] API.md: endpoint reports — 202 + polling
- [ ] EVENTS.md: click.recorded — payload + kroki consumer + retry
- [ ] EVENTS.md: idempotency opisana
- [ ] EVENTS.md: cron weekly-report — kroki
- [ ] EVENTS.md: cron alert-no-clicks — deduplikacja
- [ ] EVENTS.md: DLQ — co się dzieje
- [ ] WORKER.md: geolokalizacja — biblioteka + fallback
- [ ] WORKER.md: PDF — biblioteka + gdzie plik
- [ ] WORKER.md: e-mail — provider

## CLAUDE.md

- [ ] Stack wypełniony (język, framework, kolejka)
- [ ] Sekcja "Dodatkowe instrukcje" wypełniona
- [ ] Seed danych opisany

## docker-compose.yml

- [ ] Stworzona konwencja infrastruktury
- [ ] JWT_SECRET wypełniony
- [ ] Zmienne środowiskowe API kompletne
- [ ] Zmienne środowiskowe Worker kompletne

---

## Prompt startowy do Claude Code

> Gdy wszystkie punkty odhaczone — otwórz terminal w katalogu projektu, wpisz `claude`, wklej:

```
Jesteś seniorem [WYPEŁNIJ: język i framework].

Przeczytaj w tej kolejności:
1. docs/BRIEF.md
2. docs/architecture/ARCHITECTURE.md
3. docs/architecture/DECISIONS.md
4. docs/architecture/DATA_MODEL.md
5. docs/contracts/API.md
6. docs/contracts/EVENTS.md
7. docs/contracts/WORKER.md
8. CLAUDE.md

Po przeczytaniu odpowiedz na trzy pytania i czekaj na moją zgodę:
1. Jakie kontenery zbudujesz (lista z technologią każdego)?
2. Jak wygląda przepływ redirect — krok po kroku z czasem każdego kroku?
3. Czy czegoś brakuje w dokumentach żebyś mógł zacząć?

Nie pisz żadnego kodu dopóki nie powiem "OK, zacznij od Kroku 1".
```

---

## Kryteria oceny

| Kryterium | Punkty |
|-----------|--------|
| Back-of-envelope — poprawne obliczenia i wnioski | /10 |
| C1 i C2 — kompletne, spójne z implementacją | /10 |
| ADR — min. 4, każdy z alternatywami i uzasadnieniem | /10 |
| Kontrakt API — kompletny | /10 |
| Kontrakt eventów — idempotency, retry, DLQ | /10 |
| Redirect < 80ms (zmierzone curl) | /15 |
| Kliknięcie w statystykach w max 5s | /10 |
| Worker przetwarza eventy | /10 |
| Testy przechodzą | /10 |
| docker-compose up działa bez ręcznej konfiguracji | /5 |
| **SUMA** | **/100** |
