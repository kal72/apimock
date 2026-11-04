package domain

type WorkspaceMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}
