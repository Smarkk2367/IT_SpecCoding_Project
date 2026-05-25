# Architektura systemu — TrackFlow

> 📝 DO WYPEŁNIENIA przez zespół przed uruchomieniem agenta.

---

## 1. Back-of-envelope math

Dane z briefu: 200 000 kliknięć/miesiąc dziś → 2 000 000 za rok

```
Kliknięć / dzień (dziś):           ______________________
Kliknięć / sekundę (dziś):         ______________________
Kliknięć / sekundę (za rok):       ______________________
Rekordów w tabeli clicks po roku:  ______________________
Szacowana wielkość tabeli clicks:  ______________________
Raporty PDF / tydzień:             ______________________
```

Wnioski:

```
Bottleneck #1 to __________________ ponieważ _________________________.
Bottleneck #2 to __________________ ponieważ _________________________.
Redirect NIE może iść do bazy ponieważ ________________________________.
Cache jest potrzebny dla ______________ i trzymam w nim ________________.
Zapis kliknięcia jest asynchroniczny ponieważ _________________________.
```

---

## 2. C1 — Context Diagram

```
Narysuj ASCII lub opisz.
Format: [Aktor] --co robi--> [TRACKFLOW] --co wysyła--> [Zewnętrzny system]




```

| Element | Typ | Co robi |
|---------|-----|---------|
| Marketer | Aktor | |
| Klient agencji | Aktor | |
| Osoba klikająca | Aktor | |
| [?] | System zewnętrzny | geolokalizacja IP |
| [?] | System zewnętrzny | wysyłanie e-maili |

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
