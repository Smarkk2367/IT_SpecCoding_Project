# Architecture Decision Records

> 📝 DO WYPEŁNIENIA — minimum 4 ADR.
> Bez alternatyw to nie jest decyzja — to ogłoszenie.

---

## Szablon

```
## ADR-XXX — [Tytuł: co konkretnie decydujesz]

**Status:** Zaakceptowana

**Kontekst:**
Jakie wymaganie z briefu wymusiło tę decyzję.

**Problem:**
Jedno zdanie: "Jak _______ żeby _______?"

**Opcje:**
- A: opis — zalety / wady
- B: opis — zalety / wady

**Decyzja:** Wybieram opcję X.

**Uzasadnienie:**
Odnieś się do liczb z back-of-envelope i wymagań briefu.

**Konsekwencje:**
(+) Co zyskujesz
(-) Co tracisz

**Kiedy zrewidować:**
```

---

## ADR-001 — Wybór języka i frameworka backendu

**Status:**

**Kontekst:**

**Problem:** Jak zbudować API i Worker żeby spełnić redirect < 80ms przy wzroście 10x?

**Opcje:**
- A:
- B:
- C:

**Decyzja:**

**Uzasadnienie:**

**Konsekwencje:**
(+)
(-)

**Kiedy zrewidować:**

---

## ADR-002 — Wybór bazy danych

**Status:**

**Kontekst:**

**Problem:** Jak przechowywać dane żeby zapewnić spójność i obsłużyć miliony kliknięć?

**Opcje:**
- A:
- B:

**Decyzja:**

**Uzasadnienie:**

**Konsekwencje:**
(+)
(-)

**Kiedy zrewidować:**

---

## ADR-003 — Strategia cache dla redirectu

**Status:**

**Kontekst:**

**Problem:** Jak zapewnić redirect < 80ms bez przeciążania bazy?

**Opcje:**
- A:
- B:

**Decyzja:**

**Uzasadnienie:**

**Konsekwencje:**
(+)
(-)

**Kiedy zrewidować:**

---

## ADR-004 — Wybór brokera kolejki

**Status:**

**Kontekst:**

**Problem:** Jak zagwarantować at-least-once delivery kliknięć i nie blokować redirectu?

**Opcje:**
- A: RabbitMQ — dojrzały broker AMQP, DLQ, management UI. Wada: osobny serwis.
- B: BullMQ (Redis) — Redis jako broker i cache. Zaleta: mniej serwisów. Wada: Redis jako SPOF dla obu.
- C: Kafka — duży throughput. Wada: overengineering przy ~7 req/s.

**Decyzja:**

**Uzasadnienie:**

**Konsekwencje:**
(+)
(-)

**Kiedy zrewidować:**

---

## ADR-005 — [Dodaj swoją decyzję]

> Wskazówki: geolokalizacja IP, przechowywanie PDF, generowanie kodu krótkiego, framework frontendu.

**Status:**

**Kontekst:**

**Problem:**

**Opcje:**
- A:
- B:

**Decyzja:**

**Uzasadnienie:**

**Konsekwencje:**
(+)
(-)

**Kiedy zrewidować:**
