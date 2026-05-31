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
| | | |
| | | |
| | | |
| | | |

```
Diagram połączeń (ASCII lub opis, z protokołami):




```

> Dlaczego Worker jest osobnym kontenerem?
> _________________________________________________________________

> Dlaczego kliknięcie nie idzie od razu do bazy?
> _________________________________________________________________

> Co trzymasz w cache i dlaczego?
> _________________________________________________________________

---

## 4. Przepływ — Redirect (< 80ms)

| Krok | Opis | Czas (ms) |
|------|------|-----------|
| 1 | | ~___ |
| 2 | | ~___ |
| 3 | | ~___ |
| **Suma** | | **~___** |

```
Co przy cache miss:      _______________________________________
Co gdy Redis jest down:  _______________________________________
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
