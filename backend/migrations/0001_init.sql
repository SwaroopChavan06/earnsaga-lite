-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- for gen_random_uuid()

CREATE TABLE users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_sub   TEXT UNIQUE NOT NULL,
    email        TEXT UNIQUE NOT NULL,
    name         TEXT NOT NULL DEFAULT '',
    avatar_url   TEXT NOT NULL DEFAULT '',
    is_admin     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE offers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pubscale_id   TEXT UNIQUE NOT NULL, -- prevents duplicates on re-sync
    name          TEXT NOT NULL,
    icon_url      TEXT NOT NULL DEFAULT '',
    description   TEXT NOT NULL DEFAULT '',
    total_payout  NUMERIC(12,2) NOT NULL DEFAULT 0,
    tracking_url  TEXT NOT NULL DEFAULT '',
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_offers_name ON offers USING gin (to_tsvector('english', name));
CREATE INDEX idx_offers_active ON offers (is_active);

CREATE TABLE offer_goals (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offer_id      UUID NOT NULL REFERENCES offers(id) ON DELETE CASCADE,
    pubscale_goal_id TEXT,
    title         TEXT NOT NULL,
    instructions  TEXT NOT NULL DEFAULT '',
    reward        NUMERIC(12,2) NOT NULL DEFAULT 0,
    sort_order    INT NOT NULL DEFAULT 0
);
CREATE INDEX idx_offer_goals_offer_id ON offer_goals (offer_id);

CREATE TABLE user_offers (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    offer_id     UUID NOT NULL REFERENCES offers(id) ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'in_progress', -- in_progress | completed
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    UNIQUE (user_id, offer_id) -- enforces "only one start per user per offer"
);

CREATE TABLE user_offer_goals (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_offer_id  UUID NOT NULL REFERENCES user_offers(id) ON DELETE CASCADE,
    goal_id        UUID NOT NULL REFERENCES offer_goals(id) ON DELETE CASCADE,
    completed_at   TIMESTAMPTZ,
    UNIQUE (user_offer_id, goal_id)
);

CREATE TABLE wallet_transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount          NUMERIC(12,2) NOT NULL,
    offer_id        UUID REFERENCES offers(id) ON DELETE SET NULL,
    goal_id         UUID REFERENCES offer_goals(id) ON DELETE SET NULL,
    type            TEXT NOT NULL DEFAULT 'credit', -- credit | debit
    callback_token  TEXT UNIQUE, -- enforces idempotency on S2S callbacks
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_wallet_tx_user_id ON wallet_transactions (user_id);
CREATE INDEX idx_wallet_tx_created_at ON wallet_transactions (created_at);

CREATE TABLE events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type       TEXT NOT NULL, -- impression | click
    offer_id   UUID REFERENCES offers(id) ON DELETE SET NULL,
    user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_events_type_date ON events (type, created_at);
CREATE INDEX idx_events_offer_id ON events (offer_id);

-- +goose Down
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS wallet_transactions;
DROP TABLE IF EXISTS user_offer_goals;
DROP TABLE IF EXISTS user_offers;
DROP TABLE IF EXISTS offer_goals;
DROP TABLE IF EXISTS offers;
DROP TABLE IF EXISTS users;
