# план разработки

## Что уже реализовано

### Инфраструктура

Весь стек поднимается одной командой `docker compose up`: nginx → listing-service → search-service → worker → PostgreSQL → Redis → RabbitMQ. У каждого контейнера выставлены resource limits и healthcheck, порядок старта управляется через `depends_on: condition: service_healthy`.

PostgreSQL настроен под HDD: `random_page_cost=4.0`, `shared_buffers=128MB`. Redis работает как чистый кэш без персистентности.

### Реализованные API-эндпоинты

- `POST /api/v1/listings` — создание объявления, запись в PostgreSQL, публикация задачи модерации в RabbitMQ. Объявление уходит в статус `pending` немедленно, не блокируя ответ клиенту.
- `GET /api/v1/listings/{id}` — карточка объявления с Cache-Aside через Redis (TTL 5 минут). При промахе кэша — JOIN с таблицей пользователей.
- `GET /api/v1/listings/search` — полнотекстовый поиск через PostgreSQL FTS (`search_vector @@ plainto_tsquery('russian', ...)`). Фильтры: q, category, price_min, price_max, location, limit, offset. Результаты кэшируются в Redis (TTL 30 секунд, ключ — MD5 от параметров). Ответ содержит заголовок `X-Search-Degraded: true` — маркер того, что используется PG FTS, а не Elasticsearch.

### Сервисы

**Listing Service** — write path и card view. Помимо HTTP, слушает очередь `promotion-activated` в RabbitMQ и инвалидирует кэш карточки при активации продвижения.

**Search Service** — read path. Реализует Cache-Aside для поисковых результатов.

**Moderation Worker** — потребляет очередь `moderation` из RabbitMQ. Имитирует ML-модерацию (задержка 500–2500 мс, 80% approve / 20% reject), обновляет статус объявления в PostgreSQL. При ошибке делает nack с повторной обработкой, prefetch=1.

**nginx** — единственная точка входа. Роутинг по префиксу пути, rate limiting (200 req/s, burst=50) с JSON-ответом на 429, keepalive-соединения к upstream.

### База данных

Миграция `001_init.sql` создаёт таблицы `users` и `listings`. Поле `search_vector` — вычисляемый `tsvector` (GENERATED ALWAYS AS STORED) с GIN-индексом. Индексы: `(status, created_at DESC)`, `(category, price)`, `(is_promoted DESC, created_at DESC) WHERE status = 'published'`, `(user_id, status, created_at DESC)`.

Seed `002_seed.sql` — 1000 пользователей и ~50 000 объявлений с рандомными данными.

### Реализованные паттерны

**Паттерны проектирования:**

_API Gateway_ — nginx как единственная точка входа, скрывает топологию сервисов, централизует rate limiting. `services/nginx/nginx.conf`

_CQRS_ — Listing Service обрабатывает только мутации, Search Service — только чтение. Разные модели данных, независимое масштабирование. `services/listing/`, `services/search/`

_Cache-Aside_ — Redis используется по классической схеме в обоих сервисах с разными TTL под разный профиль нагрузки. `services/listing/internal/cache/redis/listing.go`, `services/search/internal/cache/redis/search.go`

**Паттерны устойчивости:**

_Fallback_ — Search Service возвращает результат через PostgreSQL FTS вместо ошибки при недоступности основного backend. `services/search/internal/repository/postgres/search.go`

_Rate Limiting_ — nginx ограничивает входящий поток, защищает VM при спайке. `services/nginx/nginx.conf`

_Health Check_ — каждый контейнер имеет healthcheck, сервисы выставляют `GET /health`. `docker-compose.yml`

_Bulkhead_ — resource limits в docker-compose изолируют контейнеры друг от друга. `docker-compose.yml`

---

## Что ещё нужно сделать

**k6-скрипты** — папка `loadtest/` пустая. Без них нельзя провести ни одну итерацию. Нужны четыре файла:

- `smoke.js` — минимальная проверка работоспособности перед основным тестом (5 VU, 30 секунд)
- `load.js` — рабочий тест: ramp-up до целевого RPS, затем 5 минут устойчивой нагрузки
- `stress.js` — плавный рост нагрузки до деградации, находим предел системы
- `spike.js` — 2x spike от целевого RPS, проверяем восстановление за 1 минуту

Профиль трафика: read-heavy — 80% GET-запросы (search + card view), 20% POST /listings. Запускать со своей машины, не с VM.

**README.md** — почти пустой. По заданию обязательны: как запустить (`docker compose up`, порты, healthcheck URL), curl-примеры для каждого эндпоинта с ожидаемым ответом, как запустить load-тест, список паттернов со ссылками на код, таблица итераций оптимизации.

**git-теги после каждой итерации:**
```
git tag iter-0   # baseline
git tag iter-1   # после первой оптимизации
git tag iter-2   # после второй оптимизации
```
