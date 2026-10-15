package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/prismbox/prismbox/internal/config"
)

// HandleExportConfig returns the generated native sing-box 1.14.3 JSON configuration.
func HandleExportConfig(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := config.GenerateSingBoxConfig(store)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate sing-box config: "+err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}

// Validator provides config validation using sing-box check.
type Validator interface {
	ValidateConfigData(data []byte) error
}

// HandleValidateConfig validates that the configuration can be successfully transformed
// into sing-box config and checked by sing-box check -c.
func HandleValidateConfig(store *config.Store, validator ...Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var activeValidator Validator
		if len(validator) > 0 {
			activeValidator = validator[0]
		}

		var data []byte
		var err error

		if r.ContentLength > 0 {
			bodyBytes, readErr := io.ReadAll(r.Body)
			if readErr == nil && len(bodyBytes) > 0 {
				var rawMap map[string]interface{}
				if jsonErr := json.Unmarshal(bodyBytes, &rawMap); jsonErr != nil {
					writeJSON(w, http.StatusOK, map[string]interface{}{
						"valid": false,
						"error": "invalid JSON: " + jsonErr.Error(),
					})
					return
				}

				// Check if it's native sing-box config directly (has route or inbounds array with sing-box fields)
				_, hasRoute := rawMap["route"]
				_, hasOutbounds := rawMap["outbounds"]
				_, hasInbounds := rawMap["inbounds"]

				// Try to unmarshal into PrismBoxConfig
				var customCfg config.PrismBoxConfig
				unmarshalErr := json.Unmarshal(bodyBytes, &customCfg)

				if hasRoute || (unmarshalErr != nil && (hasOutbounds || hasInbounds)) {
					// Native sing-box configuration passed directly
					data = bodyBytes
				} else if unmarshalErr == nil && (len(customCfg.Inbounds) > 0 || len(customCfg.Outbounds) > 0 || len(customCfg.Groups) > 0) {
					data, err = config.GenerateSingBoxConfigFromModel(customCfg)
					if err != nil {
						writeJSON(w, http.StatusOK, map[string]interface{}{
							"valid": false,
							"error": "failed to generate sing-box config: " + err.Error(),
						})
						return
					}
				}
			}
		}

		if len(data) == 0 {
			data, err = config.GenerateSingBoxConfig(store)
			if err != nil {
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"valid": false,
					"error": "failed to generate sing-box config: " + err.Error(),
				})
				return
			}
		}

		// Generic JSON validation
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"valid": false,
				"error": "generated config is not valid JSON: " + err.Error(),
			})
			return
		}

		// If external kernel validator is provided, invoke `sing-box check -c`
		if activeValidator != nil {
			if err := activeValidator.ValidateConfigData(data); err != nil {
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"valid": false,
					"error": err.Error(),
				})
				return
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid":   true,
			"message": "Configuration successfully validated for sing-box",
		})
	}
}

// HandleImportConfig imports raw sing-box JSON or PrismBoxConfig JSON (advanced mode).
func HandleImportConfig(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil || len(bodyBytes) == 0 {
			writeError(w, http.StatusBadRequest, "empty or unreadable request body")
			return
		}

		var rawMap map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
			return
		}

		// If payload is wrapped in {"config": ...}
		if nested, ok := rawMap["config"]; ok {
			if nestedMap, ok := nested.(map[string]interface{}); ok {
				rawMap = nestedMap
				if nestedBytes, err := json.Marshal(nestedMap); err == nil {
					bodyBytes = nestedBytes
				}
			}
		}

		// Check if payload directly matches PrismBoxConfig schema (has groups or routing object)
		var pCfg config.PrismBoxConfig
		if err := json.Unmarshal(bodyBytes, &pCfg); err == nil && len(pCfg.Groups) > 0 {
			if err := store.SaveConfig(pCfg); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to save imported config: "+err.Error())
				return
			}
			writeSuccess(w, map[string]interface{}{
				"message": "PrismBox configuration successfully imported",
				"config":  pCfg,
			})
			return
		}

		// Otherwise, parse native sing-box config structure
		newCfg := config.PrismBoxConfig{
			Inbounds:  make([]config.Inbound, 0),
			Outbounds: make([]config.Outbound, 0),
			Groups:    make([]config.Group, 0),
			Routing: config.Routing{
				Sniff:                    true,
				SniffOverrideDestination: true,
				Rules:                    make([]config.Rule, 0),
				RuleSets:                 make([]config.RuleSet, 0),
			},
			DNS: config.DNSConfig{
				Servers: make([]config.DNSServer, 0),
				Rules:   make([]config.DNSRule, 0),
			},
		}

		// Extract inbounds
		if inboundsRaw, ok := rawMap["inbounds"].([]interface{}); ok {
			for _, item := range inboundsRaw {
				if itemMap, ok := item.(map[string]interface{}); ok {
					b, _ := json.Marshal(itemMap)
					var in config.Inbound
					if err := json.Unmarshal(b, &in); err == nil {
						if in.ID == "" {
							in.ID = uuid.New().String()
						}
						newCfg.Inbounds = append(newCfg.Inbounds, in)
					}
				}
			}
		}

		// Extract outbounds and groups
		if outboundsRaw, ok := rawMap["outbounds"].([]interface{}); ok {
			for _, item := range outboundsRaw {
				if itemMap, ok := item.(map[string]interface{}); ok {
					oType, _ := itemMap["type"].(string)
					tag, _ := itemMap["tag"].(string)

					if oType == "selector" || oType == "urltest" {
						var grp config.Group
						b, _ := json.Marshal(itemMap)
						if err := json.Unmarshal(b, &grp); err == nil {
							if grp.ID == "" {
								grp.ID = uuid.New().String()
							}
							newCfg.Groups = append(newCfg.Groups, grp)
						}
					} else {
						var out config.Outbound
						b, _ := json.Marshal(itemMap)
						if err := json.Unmarshal(b, &out); err == nil {
							if out.ID == "" {
								out.ID = uuid.New().String()
							}
							if out.Tag == "" {
								out.Tag = tag
							}
							newCfg.Outbounds = append(newCfg.Outbounds, out)
						}
					}
				}
			}
		}

		// Extract DNS
		if dnsRaw, ok := rawMap["dns"].(map[string]interface{}); ok {
			newCfg.DNS.Enabled = true
			b, _ := json.Marshal(dnsRaw)
			_ = json.Unmarshal(b, &newCfg.DNS)
		}

		// Save imported config
		if err := store.SaveConfig(newCfg); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save converted config: "+err.Error())
			return
		}

		writeSuccess(w, map[string]interface{}{
			"message": "Sing-box configuration successfully imported",
			"config":  newCfg,
		})
	}
}

// HandleGetRawConfig returns the raw internal PrismBoxConfig JSON.
func HandleGetRawConfig(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := store.GetConfig()
		writeJSON(w, http.StatusOK, cfg)
	}
}

// HandleSetRawConfig replaces the raw internal PrismBoxConfig JSON.
func HandleSetRawConfig(store *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var cfg config.PrismBoxConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if err := store.SaveConfig(cfg); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save raw config: "+err.Error())
			return
		}

		writeSuccess(w, cfg)
	}
}
