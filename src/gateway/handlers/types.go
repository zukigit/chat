package handlers

// loginRequest struct for /login
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
