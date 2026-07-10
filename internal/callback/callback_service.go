package callback

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
)

type Service struct {
	Repo      *Repository
	SecretKey string
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
// before. Returns whether this call actually credited anything (false
// means it was a harmless replay).
func (s *Service) HandleCallback(ctx context.Context, userID string, value float64, token string) (credited bool, err error) {
	return s.Repo.CreditWalletIdempotent(ctx, userID, value, token)
}
