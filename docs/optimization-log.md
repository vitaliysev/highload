# Optimization Log

Профиль трафика: **read-heavy** — 80% read (`GET /search` + `GET /{id}`), 20% write (`POST /listings`).

## Таблица прогресса

| Метрика                   | NFR (ДЗ1)                        | Iter 0 | Iter 1 | Iter 2 |
|---------------------------|----------------------------------|--------|--------|--------|
| Search latency p50        | ≤ 100 ms                         |        |        |        |
| Search latency p99        | ≤ 300 ms                         |        |        |        |
| Card view latency p99     | ≤ 250 ms                         |        |        |        |
| Create listing latency p99| ≤ 500 ms                         |        |        |        |
| Max RPS (read)            | ≥ 100 RPS (мин. планка задания)  |        |        |        |
| Max RPS (write)           | ≥ 10 RPS (read-heavy профиль)    |        |        |        |
| Error rate                | < 1% при устойчивой нагрузке     |        |        |        |
| 2x spike error rate       | < 5%, восстановление за 1 мин    |        |        |        |
| CPU на пике               | 70–90%                           |        |        |        |
| Bottleneck                | —                                |        |        |        |
| NFR достигнут?            | —                                |        |        |        |

---

## Iteration 0 — Baseline

> Заполнить после первого запуска нагрузочного теста на VM.

**Дата:** —

**Конфигурация (docker-compose.yml):**

| Контейнер        | CPU limit | RAM limit | Примечания                                      |
|------------------|-----------|-----------|--------------------------------------------------|
| nginx            | 0.10      | 64M       |                                                  |
| listing-service  | 0.40      | 256M      | pgxpool MaxConns=20, MinConns=2                  |
| search-service   | 0.40      | 256M      | pgxpool MaxConns=20, MinConns=2                  |
| worker           | 0.20      | 128M      | pgxpool MaxConns=5, prefetch=1                   |
| postgresql       | 0.40      | 512M      | shared_buffers=128MB, random_page_cost=4.0 (HDD) |
| redis            | 0.10      | 256M      | maxmemory 200mb, allkeys-lru, no persistence      |
| rabbitmq         | 0.20      | 256M      |                                                  |

**RED-метрики (сервис):**

| Метрика               | Значение |
|-----------------------|----------|
| Max RPS до деградации |          |
| Error rate            |          |
| p50 latency           |          |
| p95 latency           |          |
| p99 latency           |          |

**USE-метрики (VM):**

| Ресурс         | Utilization | Saturation | Errors |
|----------------|-------------|------------|--------|
| CPU            |             |            |        |
| RAM            |             |            |        |
| Disk I/O       |             |            |        |
| DB connections |             |            |        |

**Bottleneck:** —

**Gap vs NFR:** —

---

## Iteration 1 — ...

> Заполнить после первой итерации оптимизации.

**Гипотеза:** —

**Что сделали:** —

**RED-метрики:**

| Метрика               | Iter 0 | Iter 1 | Δ |
|-----------------------|--------|--------|---|
| Max RPS до деградации |        |        |   |
| Error rate            |        |        |   |
| p50 latency           |        |        |   |
| p99 latency           |        |        |   |

**USE-метрики:**

| Ресурс         | Utilization | Saturation |
|----------------|-------------|------------|
| CPU            |             |            |
| RAM            |             |            |
| Disk I/O       |             |            |
| DB connections |             |            |

**Вывод:** —

---

## Iteration 2 — ...

> Заполнить после второй итерации оптимизации.

**Гипотеза:** —

**Что сделали:** —

**RED-метрики:**

| Метрика               | Iter 1 | Iter 2 | Δ |
|-----------------------|--------|--------|---|
| Max RPS до деградации |        |        |   |
| Error rate            |        |        |   |
| p50 latency           |        |        |   |
| p99 latency           |        |        |   |

**USE-метрики:**

| Ресурс         | Utilization | Saturation |
|----------------|-------------|------------|
| CPU            |             |            |
| RAM            |             |            |
| Disk I/O       |             |            |
| DB connections |             |            |

**Вывод:** —
