package user

type UserProfileResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	IsAdmin   bool   `json:"is_admin"`
}
