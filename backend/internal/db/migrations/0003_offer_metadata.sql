-- +goose Up
-- PubScale already sends category (ctg), platform (os), and offer type
-- (off_type) in the offers sync payload — these were previously parsed by
-- the client and discarded. Persisting them lets the offer detail page
-- show more of the data we already have instead of just name/icon/payout.
ALTER TABLE offers
    ADD COLUMN category   TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN platform   TEXT NOT NULL DEFAULT '',
    ADD COLUMN offer_type TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE offers
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS platform,
    DROP COLUMN IF EXISTS offer_type;
