package handlers

// loginRequest holds the expected JSON body for POST /login.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
