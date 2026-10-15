package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/prismbox/prismbox/internal/config"
)

// ==================== ROUTING CONFIG ====================

// HandleGetRouting returns an HTTP handler that fetches the current routing configuration.
func HandleGetRouting(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routing := store.GetRouting()
		writeJSON(w, http.StatusOK, routing)
	}
}

// HandleUpdateRouting returns an HTTP handler that updates the entire routing configuration.
func HandleUpdateRouting(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var routing config.Routing
		if err := json.NewDecoder(r.Body).Decode(&routing); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if err := store.SetRouting(routing); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update routing: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, routing)
	}
}

// ==================== ROUTING RULES ====================

// HandleGetRules returns an HTTP handler that lists all routing rules.
func HandleGetRules(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routing := store.GetRouting()
		writeJSON(w, http.StatusOK, routing.Rules)
	}
}

// HandleCreateRule returns an HTTP handler that adds a new routing rule.
func HandleCreateRule(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rule config.Rule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if rule.Outbound == "" {
			writeError(w, http.StatusBadRequest, "rule outbound action is required")
			return
		}

		if rule.ID == "" {
			rule.ID = uuid.New().String()
		}

		routing := store.GetRouting()
		routing.Rules = append(routing.Rules, rule)
		if err := store.SetRouting(routing); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to add rule: "+err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, rule)
	}
}

// HandleUpdateRule returns an HTTP handler that updates a routing rule by ID.
func HandleUpdateRule(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := getIDParam(r)
		if id == "" {
			writeError(w, http.StatusBadRequest, "rule id is required")
			return
		}

		var rule config.Rule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		routing := store.GetRouting()
		found := false
		for i, item := range routing.Rules {
			if item.ID == id {
				rule.ID = id
				routing.Rules[i] = rule
				found = true
				break
			}
		}

		if !found {
			writeError(w, http.StatusNotFound, fmt.Sprintf("rule %s not found", id))
			return
		}

		if err := store.SetRouting(routing); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update rule: "+err.Error())
			return
		}

		rule.ID = id
		writeJSON(w, http.StatusOK, rule)
	}
}

// HandleDeleteRule returns an HTTP handler that deletes a routing rule by ID.
func HandleDeleteRule(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := getIDParam(r)
		if id == "" {
			writeError(w, http.StatusBadRequest, "rule id is required")
			return
		}

		routing := store.GetRouting()
		found := false
		for i, item := range routing.Rules {
			if item.ID == id {
				routing.Rules = append(routing.Rules[:i], routing.Rules[i+1:]...)
				found = true
				break
			}
		}

		if !found {
			writeError(w, http.StatusNotFound, fmt.Sprintf("rule %s not found", id))
			return
		}

		if err := store.SetRouting(routing); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete rule: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"id":      id,
		})
	}
}

// ==================== RULE SETS ====================

// HandleGetRuleSets returns an HTTP handler that lists all rule sets.
func HandleGetRuleSets(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ruleSets := store.GetRuleSets()
		writeJSON(w, http.StatusOK, ruleSets)
	}
}

// HandleCreateRuleSet returns an HTTP handler that adds a new rule set.
func HandleCreateRuleSet(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rs config.RuleSet
		if err := json.NewDecoder(r.Body).Decode(&rs); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if rs.Tag == "" {
			writeError(w, http.StatusBadRequest, "rule set tag is required")
			return
		}
		if rs.URL == "" {
			writeError(w, http.StatusBadRequest, "rule set url is required")
			return
		}
		if rs.Type == "" {
			rs.Type = "remote"
		}
		if rs.Format == "" {
			rs.Format = "binary"
		}

		created, err := store.AddRuleSet(rs)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to add rule set: "+err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, created)
	}
}

// HandleUpdateRuleSet returns an HTTP handler that updates an existing rule set by ID.
func HandleUpdateRuleSet(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := getIDParam(r)
		if id == "" {
			writeError(w, http.StatusBadRequest, "rule set id is required")
			return
		}

		var rs config.RuleSet
		if err := json.NewDecoder(r.Body).Decode(&rs); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if err := store.UpdateRuleSet(id, rs); err != nil {
			if errors.Is(err, config.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update rule set: "+err.Error())
			return
		}

		rs.ID = id
		writeJSON(w, http.StatusOK, rs)
	}
}

// HandleDeleteRuleSet returns an HTTP handler that deletes a rule set by ID.
func HandleDeleteRuleSet(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := getIDParam(r)
		if id == "" {
			writeError(w, http.StatusBadRequest, "rule set id is required")
			return
		}

		if err := store.DeleteRuleSet(id); err != nil {
			if errors.Is(err, config.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to delete rule set: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"id":      id,
		})
	}
}
