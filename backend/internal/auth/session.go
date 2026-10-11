package auth

import (
	"crypto/sha256"
	"net/http"
)

// SessionHash extracts a session credential, never an identity supplied by the client.
// Resolve it with Store.UserBySession inside the domain operation's transaction.
func SessionHash(r *http.Request) []byte {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil || len(cookie.Value) != 43 {
		return nil
	}
	hash := sha256.Sum256([]byte(cookie.Value))
	return hash[:]
}
