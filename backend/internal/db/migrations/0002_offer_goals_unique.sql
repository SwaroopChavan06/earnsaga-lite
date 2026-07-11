-- +goose Up
ALTER TABLE offer_goals
    ADD CONSTRAINT offer_goals_offer_pubscale_goal_unique UNIQUE (offer_id, pubscale_goal_id);

-- +goose Down
ALTER TABLE offer_goals DROP CONSTRAINT IF EXISTS offer_goals_offer_pubscale_goal_unique;
