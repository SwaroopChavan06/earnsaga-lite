package models

import "time"

type User struct {
	ID        string    `json:"id"`
	GoogleSub string    `json:"-"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

type Offer struct {
	ID          string    `json:"id"`
	PubScaleID  string    `json:"pubscale_id"`
	Name        string    `json:"name"`
	IconURL     string    `json:"icon_url"`
	Description string    `json:"description"`
	TotalPayout float64   `json:"total_payout"`
	TrackingURL string    `json:"-"` // raw trk_url template, not exposed as-is to client
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OfferGoal struct {
	ID           string  `json:"id"`
	OfferID      string  `json:"offer_id"`
	Title        string  `json:"title"`
	Instructions string  `json:"instructions"`
	Reward       float64 `json:"reward"`
	SortOrder    int     `json:"sort_order"`
}

type UserOfferStatus string

const (
	StatusNotStarted UserOfferStatus = "not_started"
	StatusInProgress UserOfferStatus = "in_progress"
	StatusCompleted  UserOfferStatus = "completed"
)

type UserOffer struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	OfferID   string          `json:"offer_id"`
	Status    UserOfferStatus `json:"status"`
	StartedAt time.Time       `json:"started_at"`
}

type WalletTransaction struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Amount      float64   `json:"amount"`
	OfferID     *string   `json:"offer_id,omitempty"`
	GoalID      *string   `json:"goal_id,omitempty"`
	Type        string    `json:"type"` // credit | debit
	CallbackToken *string `json:"-"`     // used for idempotency, not exposed
	CreatedAt   time.Time `json:"created_at"`
}

type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // impression | click
	OfferID   string    `json:"offer_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
