# Архитектура системы: Маркетплейс объявлений

## 1. Архитектурный стиль и обоснование

### Выбранный стиль: Service-Based Architecture (SBA) с элементами Event-Driven

Система разбита на небольшое количество крупных сервисов (Listing Service, Search Service, Payment Service), каждый из которых отвечает за свою предметную область. Сервисы взаимодействуют синхронно через HTTP там, где важна немедленная обратная связь, и асинхронно через RabbitMQ там, где требуется развязка (модерация, уведомления, обновление статуса после webhook).

### Почему SBA подходит под данные требования

- **Read-heavy нагрузка** (поиск 3 500 RPS пик, карточки 5 000 RPS пик) требует возможности масштабировать read-путь независимо. В SBA Search Service и Listing Service масштабируются горизонтально независимо друг от друга.
- **Ограниченный бюджет** не позволяет городить полноценную MSA с service mesh, оркестрацией и distributed tracing с нуля. SBA даёт управляемую операционную сложность.
- **Асинхронная модерация ≤ 30 сек** органично ложится в event-driven паттерн через RabbitMQ — без него пришлось бы либо блокировать пользователя, либо городить polling.
- **Durability платежей** достигается через async webhook + идемпотентность на уровне БД, а не через синхронный вызов с retry-циклом.

### Сравнение с альтернативами

| Критерий | Layered Monolith | **Service-Based (выбран)** | Microservices |
|---|---|---|---|
| Масштабирование поиска отдельно | Нет | Да | Да |
| Операционная сложность | Низкая | Средняя | Высокая |
| Соответствие бюджету | Да (но не масштабируется) | Да | Нет (service mesh, K8s, observability) |
| Независимый деплой компонентов | Нет | Частично | Да |
| Время до продакшна | Быстро | Быстро | Медленно |

**Вывод**: монолит не позволяет масштабировать поиск и раздачу карточек независимо — при пике 5 000 RPS на просмотр карточки это критично. MSA избыточна для учебного проекта и ограниченного бюджета. SBA — прагматичный компромисс.

Подробнее: [ADR-001: Выбор архитектурного стиля](adr/001-architecture-style.md)

---

## 2. Компоненты системы (C4)

### C4 Level 1 — System Context

Диаграмма: [docs/diagrams/c4-context.puml](diagrams/c4-context.puml)

**Пользователи:**
- **Покупатель/Продавец** — физическое лицо, использует веб-браузер или мобильное приложение для создания объявлений, поиска и оплаты продвижения.
- **Модератор** — сотрудник компании, проверяет объявления, которые не прошли автоматическую модерацию.

**Внешние системы:**
- **Платёжный шлюз (СПБ/эквайринг)** — обрабатывает платежи за продвижение, отправляет webhook с результатом.
- **ML Moderation API** — внешний или внутренний сервис классификации текста и изображений.
- **CDN (Yandex CDN)** — раздаёт фотографии объявлений конечным пользователям.
- **SMS/Push провайдер** — отправка уведомлений пользователям о результатах модерации и платежей.
- **OAuth (VK/Яндекс)** — внешняя аутентификация.

**Граница системы**: всё, что находится внутри — API Gateway, Listing Service, Search Service, Payment Service, PostgreSQL, Elasticsearch, Redis, RabbitMQ, Object Storage.

---

### C4 Level 2 — Container Diagram

Диаграмма: [docs/diagrams/c4-container.puml](diagrams/c4-container.puml)

| # | Компонент | Технология | Тип взаимодействия | Назначение и закрываемые требования |
|---|---|---|---|---|
| 1 | **API Gateway** | Go, nginx | sync (входящий HTTP) | Единая точка входа. JWT-аутентификация, rate limiting, маршрутизация запросов на нижестоящие сервисы. Закрывает: безопасность, версионирование API (`/api/v1/`). |
| 2 | **Listing Service** | Go | sync (HTTP вниз), async (RabbitMQ) | Создание, редактирование, удаление объявлений, просмотр карточки, генерация presigned URL для загрузки фото. Жизненный цикл объявления (draft → pending → published → archived). Закрывает: ФТ-001, ФТ-003, ФТ-006. |
| 3 | **Search Service** | Go | sync (HTTP вниз), sync (ES + Redis) | Полнотекстовый поиск с фильтрами, ранжирование с учётом продвижения. При недоступности Elasticsearch деградирует на FTS в PostgreSQL. Закрывает: ФТ-002, НФТ-001 (поиск ≤ 300 ms p99). |
| 4 | **Payment Service** | Go | sync (HTTP вниз), async (webhook вход) | Инициация платежа через шлюз, приём webhook, идемпотентная обработка. Обновляет статус продвижения объявления. Закрывает: ФТ-005, НФТ-002 (availability 99.99% данных). |
| 5 | **PostgreSQL** | PostgreSQL 16 | sync | Основное транзакционное хранилище: объявления, пользователи, платежи, продвижения. ACID-гарантии. Закрывает: НФТ-004 (durability 11 девяток). |
| 6 | **Elasticsearch** | Elasticsearch 8.x | sync | Поисковый индекс. Полнотекстовый поиск, геофильтры, ранжирование. Шардирование по категориям/регионам. Закрывает: ФТ-002, НФТ-005 (масштабирование поиска). |
| 7 | **Redis** | Redis 7 | sync | Кэш карточек объявлений (TTL 5 мин), кэш популярных поисков (TTL 30 сек), JWT-сессии. Снижает нагрузку на PostgreSQL и ES при пиковых 5 000 RPS. Закрывает: НФТ-001. |
| 8 | **RabbitMQ** | RabbitMQ 3.x | async | Очередь задач модерации (обмен `moderation`), очередь уведомлений (`notifications`). Развязка между Listing Service и ML Moderation API. Закрывает: ФТ-004, НФТ-001 (модерация ≤ 30 сек). |
| 9 | **Object Storage** | Yandex Object Storage (S3-совместимый) | async (presigned URL) | Хранение фотографий объявлений. Клиент загружает фото напрямую по presigned URL, минуя бэкенд-сервисы. CDN раздаёт фото из этого же хранилища. Закрывает: ФТ-001, НФТ-004 (репликация ≥ 2 AZ), НФТ-005 (линейное масштабирование до 48 TB). |

**Итого: 9 компонентов.**

---

## 3. Sequence Diagrams

### 3.1 Happy Path — Создание объявления с асинхронной модерацией

Диаграмма: [docs/diagrams/sequence-happy-path.puml](diagrams/sequence-happy-path.puml)

**Участники**: Пользователь, API Gateway, Listing Service, PostgreSQL, RabbitMQ, Object Storage

**Описание**: Пользователь создаёт объявление — получает немедленный ответ с ID (статус `pending`). После этого система асинхронно отправляет объявление на модерацию. По результату модерации статус меняется на `published` или `rejected`, пользователь получает уведомление.

**Ключевые моменты**:
1. Ответ пользователю (201) возвращается сразу после записи в PostgreSQL — без ожидания модерации.
2. Presigned URL для фото генерируется отдельным запросом; клиент загружает фото напрямую в Object Storage.
3. Листинг-сервис публикует событие в RabbitMQ очередь `moderation` после записи объявления.
4. Moderation Worker (часть Listing Service) консьюмит задачу и вызывает ML Moderation API.
5. По итогу модерации обновляется статус в PostgreSQL и инвалидируется кэш Redis.
6. Пользователь получает push/SMS уведомление через очередь `notifications`.

---

### 3.2 Error Scenario — Поиск при недоступности Elasticsearch (Graceful Degradation)

Диаграмма: [docs/diagrams/sequence-error.puml](diagrams/sequence-error.puml)

**Участники**: Пользователь, API Gateway, Search Service, Elasticsearch, Redis, PostgreSQL

**Описание**: Elasticsearch недоступен (таймаут 200 ms). Search Service не возвращает 503 — вместо этого деградирует на PostgreSQL full-text search и возвращает результаты с заголовком, сигнализирующим о деградации.

**Ключевые моменты**:
1. Search Service сначала проверяет Redis-кэш. При cache hit — ES вообще не нужен.
2. При cache miss — попытка запроса к ES с таймаутом 200 ms.
3. При таймауте/ошибке — fallback на `tsvector` полнотекстовый поиск в PostgreSQL.
4. Ответ возвращается с кодом 200 и заголовком `X-Search-Degraded: true`.
5. Результаты PostgreSQL не ранжированы по продвижению — это фиксируется в ответе.
6. Search Service логирует ошибку в мониторинг — инженеры получают алерт.

---

### 3.3 Async Scenario — Оплата продвижения через Webhook

Диаграмма: [docs/diagrams/sequence-async.puml](diagrams/sequence-async.puml)

**Участники**: Пользователь, API Gateway, Payment Service, Платёжный шлюз, PostgreSQL, RabbitMQ, Listing Service

**Описание**: Пользователь инициирует оплату продвижения объявления. Payment Service создаёт запись о платеже и перенаправляет на страницу шлюза. После оплаты шлюз присылает webhook. Payment Service идемпотентно обрабатывает webhook и активирует продвижение.

**Ключевые моменты**:
1. При инициации платежа создаётся запись `payments` со статусом `pending` и уникальным `external_payment_id`.
2. Шлюз может прислать webhook несколько раз (сеть нестабильна). Защита: `SELECT FOR UPDATE` по `external_payment_id` + `unique constraint` — повторный webhook идемпотентно обрабатывается.
3. При дублирующем webhook (статус уже `paid`) — Payment Service возвращает 200 OK без повторной обработки.
4. После подтверждения оплаты Payment Service публикует событие в RabbitMQ `promotion-activated`.
5. Listing Service консьюмит событие и обновляет статус продвижения объявления + TTL продвижения.
6. Инвалидируется кэш в Redis для данного объявления — следующий запрос вернёт актуальный статус продвижения.

---

## 4. API Design

Версионирование: URI-prefix `/api/v1/`. При breaking changes вводится `/api/v2/` с параллельной поддержкой `/api/v1/` не менее 3 месяцев. Устаревание анонсируется заголовком `Deprecation: true` и `Sunset: <дата>`.

---

### POST /api/v1/listings

**Описание**: Создание нового объявления. Объявление создаётся со статусом `pending` и не отображается в поиске до завершения модерации.

**Request:**
```json
{
  "title": "iPhone 15 Pro 256GB",
  "description": "Куплен в декабре 2024, в отличном состоянии",
  "price": 89000,
  "currency": "RUB",
  "category_id": "electronics",
  "location": {
    "city": "Москва",
    "lat": 55.7558,
    "lon": 37.6176
  }
}
```

**Response 201:**
```json
{
  "listing_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "photo_upload_quota": 10,
  "created_at": "2026-04-18T12:00:00Z"
}
```

**Errors:**
- `400` — невалидные данные (отсутствует title, price < 0, неизвестная category_id)
- `401` — пользователь не аутентифицирован
- `422` — бизнес-ошибка (превышен лимит активных объявлений для аккаунта)
- `429` — rate limit: не более 10 объявлений в час с одного аккаунта
- `503` — сервис временно недоступен

---

### POST /api/v1/listings/{listing_id}/photos/upload-url

**Описание**: Получение presigned URL для прямой загрузки фото клиентом в Object Storage. Бэкенд не участвует в передаче файла — клиент загружает напрямую.

**Request:**
```json
{
  "filename": "photo1.jpg",
  "content_type": "image/jpeg",
  "size_bytes": 3145728
}
```

**Response 200:**
```json
{
  "upload_url": "https://storage.yandexcloud.net/...",
  "photo_id": "a1b2c3d4-...",
  "expires_at": "2026-04-18T12:15:00Z"
}
```

**Errors:**
- `400` — неподдерживаемый content_type (допустимы: image/jpeg, image/png, image/webp)
- `400` — размер файла превышает 5 MB
- `401` — не аутентифицирован
- `403` — объявление не принадлежит пользователю
- `404` — listing_id не найден
- `409` — достигнут лимит 10 фото на объявление

---

### GET /api/v1/listings/search

**Описание**: Полнотекстовый поиск объявлений с фильтрацией. Результаты ранжированы по релевантности с учётом активного продвижения. При деградации поиска заголовок `X-Search-Degraded: true` сигнализирует об использовании fallback.

**Query params:**
- `q` — поисковая фраза (обязательный)
- `category_id` — фильтр по категории
- `price_min`, `price_max` — ценовой диапазон
- `city` — фильтр по городу
- `radius_km` — радиус от точки (требует `lat`, `lon`)
- `lat`, `lon` — координаты для геофильтра
- `sort` — `relevance` (по умолчанию) | `date_desc` | `price_asc` | `price_desc`
- `page`, `per_page` — пагинация (per_page ≤ 50)

**Response 200:**
```json
{
  "total": 1420,
  "page": 1,
  "per_page": 20,
  "items": [
    {
      "listing_id": "550e8400-...",
      "title": "iPhone 15 Pro 256GB",
      "price": 89000,
      "currency": "RUB",
      "city": "Москва",
      "preview_photo_url": "https://cdn.example.com/photos/...",
      "promoted": true,
      "published_at": "2026-04-17T10:00:00Z"
    }
  ]
}
```

**Errors:**
- `400` — отсутствует параметр `q`, невалидные значения фильтров
- `429` — rate limit поисковых запросов

**Headers:** `X-Search-Degraded: true` при fallback на PostgreSQL FTS.

---

### GET /api/v1/listings/{listing_id}

**Описание**: Получение полной карточки объявления. Ответ кэшируется в Redis с TTL 5 минут. Инвалидируется при редактировании или изменении статуса модерации.

**Response 200:**
```json
{
  "listing_id": "550e8400-...",
  "title": "iPhone 15 Pro 256GB",
  "description": "Куплен в декабре 2024, в отличном состоянии",
  "price": 89000,
  "currency": "RUB",
  "status": "published",
  "category": {"id": "electronics", "name": "Электроника"},
  "location": {"city": "Москва", "lat": 55.7558, "lon": 37.6176},
  "photos": [
    {"photo_id": "a1b2c3d4-...", "url": "https://cdn.example.com/photos/..."}
  ],
  "seller": {
    "user_id": "...",
    "name": "Иван",
    "rating": 4.8,
    "phone": "+7 (999) 123-45-67"
  },
  "promoted": false,
  "published_at": "2026-04-17T10:00:00Z",
  "expires_at": "2026-05-17T10:00:00Z"
}
```

**Errors:**
- `404` — объявление не найдено или снято с публикации
- `410` — объявление удалено

---

### POST /api/v1/listings/{listing_id}/promote

**Описание**: Инициация оплаты продвижения объявления. Возвращает URL платёжной страницы для редиректа пользователя. Итоговый статус приходит асинхронно через webhook от шлюза.

**Request:**
```json
{
  "plan": "top_7days",
  "payment_method": "spb"
}
```

**Response 200:**
```json
{
  "payment_id": "pay_abc123",
  "redirect_url": "https://payment-gateway.ru/pay/...",
  "amount": 149,
  "currency": "RUB",
  "expires_at": "2026-04-18T12:30:00Z"
}
```

**Errors:**
- `400` — неизвестный план продвижения или метод оплаты
- `403` — объявление не принадлежит пользователю
- `404` — объявление не найдено
- `409` — объявление уже имеет активное продвижение
- `422` — объявление не в статусе `published`
- `503` — платёжный шлюз временно недоступен

---

## 5. Выбор БД и модель данных

### 5.1 Выбор хранилищ

#### PostgreSQL 16 — основная транзакционная БД

**Паттерн доступа**: точечные чтения по PK (карточка объявления), запись одной строки при создании объявления, JOIN между listings и photos/promotions при отдаче карточки, UPDATE по PK при смене статуса.

**Почему подходит**: ACID-гарантии критичны для платежей (нельзя потерять транзакцию). За 3 года данных ~130 GB — легко помещается на одном узле с репликой. `tsvector` даёт FTS-fallback при недоступности Elasticsearch. Знакомый стек, хорошая экосистема.

**Альтернативы**: MongoDB — не даёт ACID по умолчанию для мультидокументных транзакций, нет смысла для нашей реляционной модели. Cassandra — wide-column, хороша для write-heavy append-only нагрузки, но у нас reads >> writes и нужны JOIN.

Подробнее: [ADR-002: Выбор основной БД](adr/002-primary-database.md)

#### Elasticsearch 8.x — поисковый индекс

**Паттерн доступа**: полнотекстовый поиск по нескольким полям (title, description) + фильтры (category, price_range, geo_distance) + ранжирование (relevance score + promotion boost) — 3 500–5 000 RPS пик.

**Почему подходит**: PostgreSQL `tsvector` при 3 500 RPS без кэша не уложится в 80 ms p50 (нет нативного geo + relevance ranking). Elasticsearch — отраслевой стандарт для этого паттерна. Индекс за 3 года ~100 GB с репликами — помещается в RAM нескольких нод.

**Допустимая потеря данных**: 30 секунд (восстановление из PostgreSQL). Индекс — производный, не источник истины.

#### Redis 7 — кэш

**Паттерн доступа**: key-value по `listing_id` (кэш карточки, TTL 5 мин), key-value по хэшу поискового запроса (кэш топ-запросов, TTL 30 сек), JWT-токены сессий.

**Почему подходит**: при 5 000 RPS на карточку и среднем времени жизни кэша 5 мин, hit rate ~80% — реальная нагрузка на PostgreSQL снижается до ~1 000 RPS. Без Redis невозможно выполнить SLO ≤ 250 ms p99 на карточку.

#### Yandex Object Storage (S3) — фотографии

**Паттерн доступа**: write-once при загрузке (presigned URL, клиент загружает напрямую), read-many через CDN (миллионы запросов в день). Объём: ~48 TB за 3 года.

**Почему подходит**: блочное или файловое хранилище не масштабируется линейно до такого объёма в облаке. S3-совместимый API — стандарт для объектного хранилища. Встроенная репликация ≥ 2 AZ закрывает НФТ-004.

---

### 5.2 Модель данных (PostgreSQL)

#### Таблица `users`

| Поле | Тип | Ограничения |
|---|---|---|
| `id` | UUID | PK |
| `email` | VARCHAR(255) | UNIQUE, NOT NULL |
| `phone` | VARCHAR(20) | UNIQUE |
| `name` | VARCHAR(255) | NOT NULL |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

**Индексы:**
- `UNIQUE (email)` — аутентификация по email
- `UNIQUE (phone)` — аутентификация по телефону

**Объём**: 20 млн пользователей × ~1 KB = ~20 GB

---

#### Таблица `listings`

| Поле | Тип | Ограничения |
|---|---|---|
| `id` | UUID | PK |
| `user_id` | UUID | FK → users.id, NOT NULL |
| `title` | VARCHAR(500) | NOT NULL |
| `description` | TEXT | |
| `price` | BIGINT | NOT NULL (в копейках) |
| `currency` | CHAR(3) | NOT NULL, DEFAULT 'RUB' |
| `category_id` | VARCHAR(100) | NOT NULL |
| `city` | VARCHAR(255) | NOT NULL |
| `lat` | DOUBLE PRECISION | |
| `lon` | DOUBLE PRECISION | |
| `status` | VARCHAR(20) | NOT NULL (draft/pending/published/rejected/archived) |
| `search_vector` | TSVECTOR | — для FTS fallback |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT now() |
| `updated_at` | TIMESTAMPTZ | |
| `expires_at` | TIMESTAMPTZ | — TTL объявления (30 дней по умолчанию) |

**Индексы:**
- `(status, created_at DESC)` — листинг активных объявлений в кабинете пользователя
- `(user_id, status)` — личный кабинет: объявления пользователя по статусу
- `(category_id, status, created_at DESC)` — фильтрация по категории
- `GIN (search_vector)` — FTS fallback при недоступности Elasticsearch
- `(expires_at)` — фоновая задача архивирования истёкших объявлений

**Объём**: ~10 млн строк за 3 года × ~2 KB = ~20 GB (с индексами ~40 GB)

---

#### Таблица `listing_photos`

| Поле | Тип | Ограничения |
|---|---|---|
| `id` | UUID | PK |
| `listing_id` | UUID | FK → listings.id ON DELETE CASCADE |
| `storage_key` | VARCHAR(512) | NOT NULL — путь в Object Storage |
| `position` | SMALLINT | NOT NULL — порядок фото (0–9) |
| `created_at` | TIMESTAMPTZ | NOT NULL |

**Индексы:**
- `(listing_id, position)` — загрузка фото карточки в порядке позиции

**Объём**: ~50 млн строк (5 фото × 10 млн объявлений) × ~0.5 KB = ~25 GB

---

#### Таблица `payments`

| Поле | Тип | Ограничения |
|---|---|---|
| `id` | UUID | PK |
| `listing_id` | UUID | FK → listings.id |
| `user_id` | UUID | FK → users.id |
| `external_payment_id` | VARCHAR(255) | UNIQUE — ID от платёжного шлюза |
| `amount` | BIGINT | NOT NULL (в копейках) |
| `currency` | CHAR(3) | NOT NULL |
| `status` | VARCHAR(20) | NOT NULL (pending/paid/failed/refunded) |
| `payment_method` | VARCHAR(50) | |
| `created_at` | TIMESTAMPTZ | NOT NULL |
| `updated_at` | TIMESTAMPTZ | |

**Индексы:**
- `UNIQUE (external_payment_id)` — идемпотентная обработка webhook (ключевой!)
- `(listing_id)` — история платежей по объявлению
- `(user_id, created_at DESC)` — история платежей пользователя

**Объём**: ~330 тыс. строк/год (900 платежей/день × 365) — пренебрежимо мало

---

#### Таблица `promotions`

| Поле | Тип | Ограничения |
|---|---|---|
| `id` | UUID | PK |
| `listing_id` | UUID | FK → listings.id |
| `payment_id` | UUID | FK → payments.id, UNIQUE |
| `plan` | VARCHAR(50) | NOT NULL (top_7days, top_30days, etc.) |
| `starts_at` | TIMESTAMPTZ | NOT NULL |
| `expires_at` | TIMESTAMPTZ | NOT NULL |
| `active` | BOOLEAN | NOT NULL, DEFAULT true |

**Индексы:**
- `(listing_id, active, expires_at)` — проверка активного продвижения при выдаче карточки и поиске
- `(expires_at) WHERE active = true` — частичный индекс для фоновой задачи деактивации истёкших продвижений

**Объём**: ~330 тыс. строк/год — пренебрежимо мало

---

## 6. Ссылки на ADR

| ADR | Тема | Решение |
|---|---|---|
| [ADR-001](adr/001-architecture-style.md) | Выбор архитектурного стиля | Service-Based Architecture с элементами Event-Driven |
| [ADR-002](adr/002-primary-database.md) | Выбор основной БД | PostgreSQL как единственная транзакционная БД |
| [ADR-003](adr/003-payment-processing.md) | Обработка платежей | Async через webhook с идемпотентной обработкой |
