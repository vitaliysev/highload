CREATE TABLE listing_photos (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id  UUID         NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    storage_key VARCHAR(512) NOT NULL,
    position    SMALLINT     NOT NULL CHECK (position BETWEEN 0 AND 9),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (listing_id, position)
);

CREATE INDEX idx_listing_photos_listing_position ON listing_photos (listing_id, position);

CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'failed', 'refunded');

CREATE TABLE payments (
    id                  UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id          UUID           NOT NULL REFERENCES listings(id),
    user_id             UUID           NOT NULL REFERENCES users(id),
    external_payment_id VARCHAR(255)   UNIQUE,
    amount              BIGINT         NOT NULL,
    currency            CHAR(3)        NOT NULL DEFAULT 'RUB',
    status              payment_status NOT NULL DEFAULT 'pending',
    payment_method      VARCHAR(50),
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX idx_payments_listing_id ON payments (listing_id);
CREATE INDEX idx_payments_user_created ON payments (user_id, created_at DESC);

CREATE TRIGGER trg_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE promotions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id  UUID        NOT NULL REFERENCES listings(id),
    payment_id  UUID        NOT NULL UNIQUE REFERENCES payments(id),
    plan        VARCHAR(50) NOT NULL,
    starts_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    active      BOOLEAN     NOT NULL DEFAULT true
);

CREATE INDEX idx_promotions_listing_active ON promotions (listing_id, active, expires_at);
CREATE INDEX idx_promotions_expires_active ON promotions (expires_at) WHERE active = true;
