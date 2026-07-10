package callback

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"

	"earnsaga-lite/internal/leaderboard"
)

// repository is the seam callback.Service depends on. Unexported since it
// only exists to let tests substitute a fake — the concrete *Repository
// (in repository.go) satisfies it implicitly, so main.go wiring is
// untouched.
type repository interface {
	CreditWalletIdempotent(ctx context.Context, userID string, value float64, token string) (credited bool, err error)
}

type Service struct {
	Repo        repository
	SecretKey   string
	Leaderboard *leaderboard.Service // optional-ish: nil-checked before use, so callback still works without it wired
}

// VerifySignature checks PubScale's S2S signature per their documented
// formula: MD5(secret_key.user_id.int(value).token)
func (s *Service) VerifySignature(userID string, value float64, token, signature string) bool {
	template := fmt.Sprintf("%s.%s.%d.%s", s.SecretKey, userID, int64(value), token)
	sum := md5.Sum([]byte(template))
	expected := hex.EncodeToString(sum[:])
	return expected == signature
}

// HandleCallback credits the wallet if the token hasn't been processed
// before, then records the earning on the leaderboard. Returns whether
// this call actually credited anything (false means it was a harmless
// replay, and we deliberately skip the leaderboard update in that case —
// a replayed callback must not double-count someone's ranking either).
func (s *Service) HandleCallback(ctx context.Context, userID string, value float64, token string) (credited bool, err error) {
	credited, err = s.Repo.CreditWalletIdempotent(ctx, userID, value, token)
	if err != nil || !credited {
		return credited, err
	}

	if s.Leaderboard != nil {
		if lbErr := s.Leaderboard.RecordEarning(ctx, userID, value); lbErr != nil {
			// The wallet credit already committed successfully — a
			// leaderboard update failure shouldn't fail the whole callback
			// (PubScale would retry and we'd double-credit the wallet).
			// Log it as a known inconsistency to investigate instead.
			log.Printf("leaderboard update failed for user_id=%s after successful credit: %v", userID, lbErr)
		}
	}

	return credited, nil
}
