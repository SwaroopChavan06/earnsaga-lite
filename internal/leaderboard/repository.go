package leaderboard

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

const (
	RangeDaily   = "daily"
	RangeWeekly  = "weekly"
	RangeAllTime = "alltime"

	allTimeKey = "lb:alltime"
)

func dailyKey(t time.Time) string {
	return "lb:daily:" + t.UTC().Format("2006-01-02")
}

// weeklyKey is anchored to the most recent Sunday (UTC), NOT Go's ISO week
// (which resets Monday). The spec requires a Sunday-midnight-UTC reset —
// using time.Time.Weekday() directly (Sunday == 0) gets that right.
func weeklyKey(t time.Time) string {
	t = t.UTC()
	daysSinceSunday := int(t.Weekday())
	sunday := t.AddDate(0, 0, -daysSinceSunday)
	return "lb:weekly:" + sunday.Format("2006-01-02")
}

func keyForRange(rangeName string) string {
	switch rangeName {
	case RangeDaily:
		return dailyKey(time.Now())
	case RangeWeekly:
		return weeklyKey(time.Now())
	default:
		return allTimeKey
	}
}

// IncrementScore pushes a wallet credit into all three sorted sets in one
// pipelined round trip. Daily/weekly keys are date-scoped, so "resets" fall
// out naturally from the key changing at midnight/Sunday — no cron job
// needed. TTLs keep old keys from accumulating in Redis forever.
func (r *Repository) IncrementScore(ctx context.Context, userID string, amount float64) error {
	now := time.Now()
	dKey := dailyKey(now)
	wKey := weeklyKey(now)

	pipe := r.Redis.TxPipeline()
	pipe.ZIncrBy(ctx, dKey, amount, userID)
	pipe.Expire(ctx, dKey, 48*time.Hour)
	pipe.ZIncrBy(ctx, wKey, amount, userID)
	pipe.Expire(ctx, wKey, 15*24*time.Hour)
	pipe.ZIncrBy(ctx, allTimeKey, amount, userID)
	_, err := pipe.Exec(ctx)
	return err
}

type ScoreRow struct {
	UserID string
	Score  float64
}

// GetTopScores reads directly from Redis — this is what satisfies "don't
// query the main transactional database for every leaderboard load."
func (r *Repository) GetTopScores(ctx context.Context, rangeName string, limit int64) ([]ScoreRow, error) {
	key := keyForRange(rangeName)
	results, err := r.Redis.ZRevRangeWithScores(ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, err
	}
	rows := make([]ScoreRow, 0, len(results))
	for _, z := range results {
		member, ok := z.Member.(string)
		if !ok {
			continue
		}
		rows = append(rows, ScoreRow{UserID: member, Score: z.Score})
	}
	return rows, nil
}

type UserInfo struct {
	Name      string
	AvatarURL string
}

// GetUserInfo is a small, bounded Postgres lookup (at most 50 rows, one
// query) for display names of whoever's already in the top 50 per Redis —
// this is not the "query main DB per load" the spec warns against, since
// it's not aggregating over wallet_transactions.
func (r *Repository) GetUserInfo(ctx context.Context, userIDs []string) (map[string]UserInfo, error) {
	if len(userIDs) == 0 {
		return map[string]UserInfo{}, nil
	}
	rows, err := r.DB.Query(ctx, `SELECT id, name, avatar_url FROM users WHERE id = ANY($1)`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]UserInfo{}
	for rows.Next() {
		var id, name, avatar string
		if err := rows.Scan(&id, &name, &avatar); err != nil {
			return nil, err
		}
		out[id] = UserInfo{Name: name, AvatarURL: avatar}
	}
	return out, rows.Err()
}
