# Маркетплейс объявлений (PoC)

## Как запустить

```bash
git clone <repo>
cd highload
docker compose up --build
```

Healthcheck:

```bash
curl http://localhost:8080/health
```

**Порты:**

| Сервис | Адрес |
|---|---|
| API (nginx) | http://localhost:8080 |
| RabbitMQ UI | http://localhost:15672 (guest / guest) |

---

## Как проверить

Получить реальные ID:

```bash
USER_ID=$(curl -s 'http://localhost:8080/api/v1/listings/search?q=iPhone&limit=1' \
  | jq -r '.items[0].user_id')

LISTING_ID=$(curl -s 'http://localhost:8080/api/v1/listings/search?q=iPhone&limit=1' \
  | jq -r '.items[0].id')
```

**Создать объявление:**

```bash
curl -s -X POST http://localhost:8080/api/v1/listings \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$USER_ID\",\"title\":\"iPhone 15 Pro\",\"description\":\"как новый\",\"price\":89000,\"category\":\"Электроника\",\"location\":\"Москва\"}"
```

**Получить карточку:**

```bash
curl -s http://localhost:8080/api/v1/listings/$LISTING_ID
```

**Поиск:**

```bash
curl -s 'http://localhost:8080/api/v1/listings/search?q=iPhone&limit=5'
```

**Получить presigned URL для фото:**

```bash
curl -s -X POST http://localhost:8080/api/v1/listings/$LISTING_ID/photos/upload-url \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$USER_ID\",\"filename\":\"photo.jpg\",\"content_type\":\"image/jpeg\",\"size_bytes\":2097152}"
```

**Активировать продвижение:**

```bash
curl -s -X POST http://localhost:8080/api/v1/listings/$LISTING_ID/promote \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$USER_ID\",\"plan\":\"top_7days\",\"payment_method\":\"spb\"}"
```

**Объявления пользователя (личный кабинет):**

```bash
curl -s "http://localhost:8080/api/v1/users/$USER_ID/listings?status=published&per_page=10"
```

---

## Как запустить нагрузочный тест

Запускать с ноутбука, не с VM. Перед тестом убедиться, что стек поднят на VM.

```bash
# smoke — базовая проверка (30 сек)
k6 run -e BASE_URL=http://<VM_IP>:8080 loadtest/smoke.js

# load — устойчивая нагрузка (7 мин)
k6 run -e BASE_URL=http://<VM_IP>:8080 loadtest/load.js

# stress — поиск потолка RPS (7 мин)
k6 run -e BASE_URL=http://<VM_IP>:8080 loadtest/stress.js

# spike — 3x спайк (6 мин)
k6 run -e BASE_URL=http://<VM_IP>:8080 loadtest/spike.js
```

С HTML-отчётом (из корня репозитория):

```bash
./loadtest/run.sh stress
# отчёт сохраняется в loadtest/reports/stress_<timestamp>.html
```

**Где смотреть метрики во время теста:**

| Источник | Команда |
|---|---|
| k6 (RED) | вывод в терминале + `K6_WEB_DASHBOARD=true` открывает браузер |
| docker stats (USE) | `docker stats` на VM |
| CPU / RAM | `htop` на VM |
| Disk I/O | `iostat -x 2` на VM (нужен пакет `sysstat`: `apt install sysstat`) |
| Активные запросы БД | `docker compose exec postgres psql -U postgres -d marketplace -c "SELECT pid, state, wait_event, query_start, left(query,80) FROM pg_stat_activity WHERE state != 'idle' ORDER BY query_start"` |

---

## Паттерны

### 1. API Gateway (проектирование)

**Где:** `services/nginx/nginx.conf:19`

Nginx — единая точка входа. Маршрутизирует `/api/v1/listings/search` на Search Service, остальные `/api/v1/listings` и `/api/v1/users` на Listing Service. Изолирует клиентов от топологии сервисов и позволяет масштабировать каждый сервис независимо.

### 2. Cache-Aside (проектирование)

**Где:** `services/listing/internal/service/listing.go:51`, `services/listing/internal/cache/redis/listing.go`

При запросе карточки сервис сначала проверяет Redis (TTL 5 мин). При промахе читает из PostgreSQL и кладёт в кэш. При получении события `promotion-activated` из RabbitMQ кэш инвалидируется (`services/listing/internal/queue/rabbitmq/consumer.go:68`). Снижает нагрузку на PostgreSQL при пиковых 5 000 RPS на карточку.

### 3. Rate Limiting (устойчивость)

**Где:** `services/nginx/nginx.conf:10`

Nginx ограничивает входящий трафик до 200 req/s с одного IP (`limit_req_zone`). При превышении возвращает `429` с телом `{"error":"rate limit exceeded"}` вместо того чтобы пропустить трафик к сервисам и вызвать OOM. Burst=50 позволяет кратковременные всплески.

### 4. Graceful Degradation / Fallback (устойчивость)

**Где:** `services/search/internal/repository/postgres/search.go:27`, `services/search/internal/handler/search.go:75`

В PoC Elasticsearch не поднят — Search Service автоматически деградирует на PostgreSQL FTS через `tsvector` + GIN-индекс. Клиент получает `200 OK` с заголовком `X-Search-Degraded: true`. Ранжирование по `is_promoted DESC, created_at DESC` сохраняется. Сервис никогда не возвращает `503` из-за недоступности поискового бэкенда.

---
