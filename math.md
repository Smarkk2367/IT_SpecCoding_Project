Klienci teraz: 40
Klienci docelowi: 200
Klikający: ~200 0000 / miesiąc
Klikający docelowi: 2M / miesiąc
Marketerzy: 15

### Architektura musi uwzględniać skok - nie ma opcji przepisywania systemu

Czas aktywności linku: maks. rok
Termin (SaaS B2B): 6 miesięcy

max. latency: 80 ms (od przejścia requestu do wysłania responsa)

max. delay wysyłania raportu: 15 minut

max. delay pojawienia się kliknięcia w statsach: 5s

Server VPS (8 vCPU, 32GB RAM)

### Back-of-envelope:

RPS:
2 000 000 / 30 / 24 / 3600 = 0,77 req/s

Traffic:
 - average: 0,8 rps
 - peak normalny (burst 10x): 20 rps (zaokrąglmy w górę)
 - peak kampanii (burst normal x 5): 100 rps
 - viral/mocna popularność: > 500 rps -- pod to robimy