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

**Status: Zaakceptowane**

**Kontekst: Wymaganie: redirect < 80 ms, brak utraty kliknięć, wzrost do 2M klików/miesiąc, prosty deployment**

**Problem:** Jak zbudować API i Worker żeby spełnić redirect < 80ms przy wzroście 10x?

**Opcje:**
- A: Node.js: szybki development, duży ekosystem / wyższy narzut CPU, GC pause, trudniej utrzymać ultra-low latency przy burstach
- B: Python: bardzo szybki development, prostota / gorsza wydajność runtime, większe ryzyko latency spike
- C: Go: bardzo niski latency (kompilowany binary), przewidywalne zużycie CPU, brak GC pause problemów w skali redirectów łatwy concurrencymodel (goroutines) / wolniejszy development niż Node/Python

**Decyzja: C (Go)**

**Uzasadnienie: Redirect path musi utrzymywać stabilne sub-10ms execution time, aby cały system zmieścił się w 80 ms SLA z dużym marginesem na network i Redis**

**Konsekwencje:**
(+)stabilny redirect < 10 ms w hot path, łatwe utrzymanie wysokiego RPS na VPS, mniejsze ryzyko jitteru latency (kluczowe dla UX konwersji), prosta architektura workerów
(-)wolniejszy development niż Node/Python, mniejszy ekosystem „plug-and-play” niż JS, większa odpowiedzialność za strukturę kodu

**Kiedy zrewidować:**
jeśli RPS przekroczy ~10k/s i potrzebna będzie pozioma dystrybucja wielu regionów
jeśli zespół rozrośnie się >10 developerów i speed of delivery stanie się ważniejszy niż latency
jeśli system przejdzie z redirect-first na analytics-heavy platform (np. real-time bidding / adtech core)
---

## ADR-002 — Wybór bazy danych

**Status: Zaakceptowane**

**Kontekst: System musi przechowywać:

linki (CRUD + lookup po short_code),
kliknięcia (write-heavy time-series),
agregaty statystyk (dashboard + raporty),
dane o kliknięciach nie mogą zaginąć.**

**Problem:** Jak przechowywać dane żeby zapewnić spójność i obsłużyć miliony kliknięć?

**Opcje:**
- A: PostgreSQL jako monolit: jedna technologia, ACID dla wszystkiego / write amplification przy clicks, indeksy spowalniają inserty, brak streamingu, ryzyko przeciążenia
- B: PostgreSQL + Redis (cache + stream + hot data layer): szybki loop redirectu, kolejka, trwały storage / większa złożoność, konieczność obsługi idempotencji

**Decyzja: B**

**Uzasadnienie: Wymagania systemu wymuszają rozdzielenie ścieżek:

redirect musi działać w ~8–10 ms (Redis cache path)
analytics mogą działać w sekundach (worker async)
dane nie mogą zginąć → potrzebny durable buffer (Redis Streams + PostgreSQL sink)**

**Konsekwencje:**
(+) spełnienie redirect < 80 ms
brak utraty danych (stream + DB sink)
skalowalność write-heavy eventów
możliwość realtime stats (<5s) dzięki Redis counters
elastyczne raportowanie
(-) większa złożoność systemu (2 warstwy storage)
konieczność:
 idempotent processing
 retry logic w workerach
 monitoring backlogu streama
trudniejsze debugowanie niż monolit DB

**Kiedy zrewidować:**
jeśli Redis stanie się bottleneckiem (stream throughput / memory pressure)
jeśli wymagane będzie multi-region active-active replication

---

## ADR-003 — Strategia cache dla redirectu

**Status: Zaakceptowane**

**Kontekst: Redirect musi działać w < 80 ms przy burstach do ~1000 RPS, a kliknięcia nie mogą obciążać PostgreSQL.**

**Problem:** Jak zapewnić redirect < 80ms bez przeciążania bazy?

**Opcje:**
- A: Brak cache: prostota, brak dodatkowej infrastruktury / ryzyko przekroczenia 80 ms przy burstach, DB bottleneck, brak stabilności latency
- B: Redis cache (short_code → target_url + metadata): sub-ms lookup, odciążenie PostgreSQL, stabilne tail latency, łatwa skalowalność /
dodatkowa warstwa, konieczność synchronizacji cache invalidation

**Decyzja: B**

**Uzasadnienie: Redirect hot path musi być deterministycznie szybki**

**Konsekwencje:**
(+) stabilny redirect latency < 10 ms
brak obciążenia PostgreSQL w hot path
łatwe skalowanie do 10x–50x traffic
lepsza odporność na bursty kampanii
(-) dodatkowa warstwa infrastruktury
potencjalna niespójność chwilowa

**Kiedy zrewidować:**
jeśli Redis memory cost stanie się krytyczny (>30–40 GB)
jeśli system przejdzie na multi-region active-active routing
jeśli wymagane będzie 100% strong consistency dla redirectów

---

## ADR-004 — Wybór brokera kolejki

**Status: Zaakceptowana**

**Kontekst: System musi: nie blokować redirectu (< 80 ms SLA), zapewnić at-least-once delivery kliknięć, obsłużyć bursty (do ~1000 RPS)**

**Problem:** Jak zagwarantować at-least-once delivery kliknięć i nie blokować redirectu?

**Opcje:**
- A: RabbitMQ — dojrzały broker AMQP, DLQ, management UI. Wada: osobny serwis.
- B: BullMQ (Redis) — Redis jako broker i cache. Zaleta: mniej serwisów. Wada: Redis jako SPOF dla obu.
- C: Kafka — duży throughput. Wada: overengineering przy ~7 req/s.

**Decyzja: B**

**Uzasadnienie: Przy skali systemu (2M klików/miesiąc, bursty do ~1000 RPS) kluczowe są:
minimalna złożoność operacyjna, brak dodatkowych serwisów, szybki development i utrzymanie.**

**Konsekwencje:**
(+)
jeden system (Redis) = minimalna infrastruktura
szybki deployment w Docker Compose
wystarczająca wydajność dla peaków
prosty model dla 1 developera
niskie latency enqueue (<1–2 ms)
(-)
brak naturalnej separacji storage vs messaging
konieczność solidnego AOF + backup strategy
ograniczenia przy przyszłym skalowaniu (100M+ klików/miesiąc)

**Kiedy zrewidować:**
gdy throughput przekroczy ~50–100M klików/miesiąc
gdy wymagane będzie multi-region processing
gdy Redis stanie się bottleneckiem

---

## ADR-005 — Worker jako proces Go

**Status: Zaakceptowana**

**Kontekst:** Worker przetwarza kliknięcia, raporty i notyfikacje poza hot path redirectu. System ma być prosty w utrzymaniu przez jednego developera i uruchamiany przez Docker Compose.

**Problem:** Jak zaimplementować workera, żeby nie mnożyć runtime'ów i zachować spójność z backendem?

**Opcje:**
- A: Worker w Node.js — łatwy dostęp do bibliotek typu `ua-parser-js`, `geoip-lite`, Puppeteer i Nodemailer / drugi runtime, osobny styl kodu, więcej zależności operacyjnych.
- B: Worker w Go — jeden język dla API i workera, prostszy deployment, spójne modele i testy / trzeba użyć Go-odpowiedników bibliotek.

**Decyzja:** Wybieram opcję B.

**Uzasadnienie:** Skala v1 nie wymaga osobnego ekosystemu workerów. Jeden binarny runtime Go upraszcza Docker Compose, testy integracyjne i utrzymanie, a zadania workera można obsłużyć sprawdzonymi bibliotekami Go.

**Konsekwencje:**
(+) jeden język i wspólne typy kontraktów eventów
(+) prostsze obrazy Docker i mniej ruchomych części
(+) łatwiejsze testowanie `go test`
(-) część bibliotek z WORKER.md ma inne odpowiedniki niż pierwotnie zapisane nazwy JS

**Kiedy zrewidować:**
jeśli generowanie PDF albo enrichment danych zacznie wymagać specjalistycznego środowiska Node/Chromium, którego nie da się stabilnie utrzymać w procesie Go
