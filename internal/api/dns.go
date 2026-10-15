package api

import (
	"encoding/json"
	"net/http"

	"github.com/prismbox/prismbox/internal/config"
)

// HandleGetDNS returns an HTTP handler that fetches the current DNS configuration.
func HandleGetDNS(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dns := store.GetDNS()
		writeJSON(w, http.StatusOK, dns)
	}
}

// HandleUpdateDNS returns an HTTP handler that updates the DNS configuration.
// Supports setting enabled=false to leave the DNS section completely empty in sing-box config.
func HandleUpdateDNS(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var dns config.DNSConfig
		if err := json.NewDecoder(r.Body).Decode(&dns); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if err := store.SetDNS(dns); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update DNS config: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, dns)
	}
}
