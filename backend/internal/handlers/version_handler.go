package handlers

import (
	"net/http"

	"github.com/zukigit/chat/backend/internal/lib"
)

// GetVersion handles GET /version and returns the current chat protocol version.
func GetVersion(w http.ResponseWriter, r *http.Request) {
	lib.WriteJSON(w, http.StatusOK, lib.Response{
		Success: true,
		Data:    map[string]int{"version": lib.ChatResponseEnvelopeVersion},
	})
}
