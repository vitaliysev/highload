CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================
-- Таблица пользователей
-- ============================================================
CREATE TABLE users (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT        NOT NULL,
    email        TEXT        NOT NULL UNIQUE,
    phone        TEXT,
    city         TEXT,
    avatar_url   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Таблица объявлений
-- ============================================================
CREATE TYPE listing_status AS ENUM ('pending', 'published', 'rejected', 'archived');

CREATE TABLE listings (
    id            UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID           NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Контент
    title         TEXT           NOT NULL CHECK (char_length(title) BETWEEN 3 AND 200),
    description   TEXT           CHECK (char_length(description) <= 5000),
    price         NUMERIC(12, 2) CHECK (price >= 0),
    category      TEXT           NOT NULL,
    location      TEXT           NOT NULL,     -- город/район, текст

    -- Состояние
    status        listing_status NOT NULL DEFAULT 'pending',
    is_promoted   BOOLEAN        NOT NULL DEFAULT false,
    promoted_until TIMESTAMPTZ,               -- до когда активно продвижение

    -- Статистика
    views_count   INT            NOT NULL DEFAULT 0,

    -- Временны́е метки
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ    NOT NULL DEFAULT now(),

    -- Полнотекстовый поиск: title + description + location на русском
    search_vector TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('russian',
            coalesce(title, '')       || ' ' ||
            coalesce(description, '') || ' ' ||
            coalesce(location, '')
        )
    ) STORED
);

-- ============================================================
-- Индексы
-- ============================================================

-- FTS — основной поиск
CREATE INDEX idx_listings_search_vector ON listings USING GIN (search_vector);

-- Фильтрация по статусу + сортировка по дате (главная выборка published)
CREATE INDEX idx_listings_status_created ON listings (status, created_at DESC);

-- Фильтрация по категории и цене
CREATE INDEX idx_listings_category_price ON listings (category, price);

-- Ранжирование: продвинутые объявления сначала, затем по дате
CREATE INDEX idx_listings_promoted_created ON listings (is_promoted DESC, created_at DESC)
    WHERE status = 'published';

-- Личный кабинет: объявления конкретного пользователя
CREATE INDEX idx_listings_user_id ON listings (user_id, status, created_at DESC);

-- ============================================================
-- Автообновление updated_at
-- ============================================================
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_listings_updated_at
    BEFORE UPDATE ON listings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
