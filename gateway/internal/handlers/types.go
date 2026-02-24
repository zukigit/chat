package handlers

// loginRequest struct for /login
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// signupRequest struct for /signup
type signupRequest struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"passwd,omitempty"` // required for email signup
	Code     string `json:"code,omitempty"`   // required for google signup
}
