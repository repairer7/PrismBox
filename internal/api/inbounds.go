package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/prismbox/prismbox/internal/config"
)

// HandleGetInbounds returns an HTTP handler that lists all inbounds.
func HandleGetInbounds(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inbounds := store.GetInbounds()
		writeJSON(w, http.StatusOK, inbounds)
	}
}

// HandleCreateInbound returns an HTTP handler that creates a new inbound.
func HandleCreateInbound(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var inbound config.Inbound
		if err := json.NewDecoder(r.Body).Decode(&inbound); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if inbound.Type == "" {
			writeError(w, http.StatusBadRequest, "inbound type is required")
			return
		}
		if inbound.Tag == "" {
			inbound.Tag = inbound.Type + "-in"
		}

		created, err := store.AddInbound(inbound)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to add inbound: "+err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, created)
	}
}

// HandleUpdateInbound returns an HTTP handler that updates an inbound by ID.
func HandleUpdateInbound(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := getIDParam(r)
		if id == "" {
			writeError(w, http.StatusBadRequest, "inbound id is required")
			return
		}

		var inbound config.Inbound
		if err := json.NewDecoder(r.Body).Decode(&inbound); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if err := store.UpdateInbound(id, inbound); err != nil {
			if errors.Is(err, config.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update inbound: "+err.Error())
			return
		}

		inbound.ID = id
		writeJSON(w, http.StatusOK, inbound)
	}
}

// HandleDeleteInbound returns an HTTP handler that deletes an inbound by ID.
func HandleDeleteInbound(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := getIDParam(r)
		if id == "" {
			writeError(w, http.StatusBadRequest, "inbound id is required")
			return
		}

		if err := store.DeleteInbound(id); err != nil {
			if errors.Is(err, config.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to delete inbound: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"id":      id,
		})
	}
}
