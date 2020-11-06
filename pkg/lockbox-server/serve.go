package server

import (
	"net/http"

	"github.com/kevinburke/nacl"
)

// PublicKey creates an HTTP handler that responses with the specified public key
// as binary data.
func PublicKey(pubKey nacl.Key) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(pubKey[:])
	})
}
