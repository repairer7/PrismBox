package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrNotFound is returned when an item cannot be found by ID.
	ErrNotFound = errors.New("item not found")
)

// Store provides thread-safe persistent configuration storage with file locking.
type Store struct {
	filePath string
	lockPath string
	mu       sync.RWMutex
	config   PrismBoxConfig
}

// NewStore loads existing config or creates default config in dataDir/prismbox-config.json.
func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	filePath := filepath.Join(dataDir, "prismbox-config.json")
	lockPath := filepath.Join(dataDir, "prismbox-config.json.lock")

	store := &Store{
		filePath: filePath,
		lockPath: lockPath,
		config:   defaultConfig(),
	}

	// Check if primary config exists
	if _, err := os.Stat(filePath); err == nil {
		if err := store.Load(); err != nil {
			return nil, fmt.Errorf("failed to load existing config: %w", err)
		}
		return store, nil
	}

	// Check legacy prismbox.json migration
	legacyPath := filepath.Join(dataDir, "prismbox.json")
	if _, err := os.Stat(legacyPath); err == nil {
		data, readErr := os.ReadFile(legacyPath)
		if readErr == nil {
			var legacyCfg PrismBoxConfig
			if unmarshalErr := json.Unmarshal(data, &legacyCfg); unmarshalErr == nil {
				store.config = legacyCfg
				if saveErr := store.Save(); saveErr == nil {
					return store, nil
				}
			}
		}
	}

	// Initialize with baseline default configuration
	if err := store.Save(); err != nil {
		return nil, fmt.Errorf("failed to save initial config: %w", err)
	}

	return store, nil
}

// defaultConfig returns the baseline configuration for PrismBox.
func defaultConfig() PrismBoxConfig {
	return PrismBoxConfig{
		Inbounds: []Inbound{
			{
				ID:                       uuid.New().String(),
				Tag:                      "tproxy-in",
				Type:                     "tproxy",
				Enabled:                  true,
				Listen:                   "::",
				Port:                     12345,
				Network:                  "tcp,udp",
				Sniff:                    true,
				SniffOverrideDestination: true,
			},
		},
		Outbounds: []Outbound{
			{
				ID:      uuid.New().String(),
				Tag:     "direct",
				Type:    "direct",
				Enabled: true,
			},
			{
				ID:      uuid.New().String(),
				Tag:     "block",
				Type:    "block",
				Enabled: true,
			},
		},
		Groups: make([]Group, 0),
		Routing: Routing{
			Sniff:                    true,
			SniffOverrideDestination: true,
			DomainStrategy:           "prefer_ipv4",
			Rules:                    make([]Rule, 0),
			RuleSets:                 make([]RuleSet, 0),
		},
		DNS: DNSConfig{
			Enabled:          false,
			Servers:          make([]DNSServer, 0),
			Rules:            make([]DNSRule, 0),
			Strategy:         "prefer_ipv4",
			DisableCache:     false,
			DisableExpire:    false,
			IndependentCache: false,
		},
	}
}

// acquireFileLock obtains a cross-process lockfile, handling stale locks if needed.
func (s *Store) acquireFileLock() (func(), error) {
	deadline := time.Now().Add(5 * time.Second)
	staleThreshold := 10 * time.Second

	for {
		f, err := os.OpenFile(s.lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d %d\n", os.Getpid(), time.Now().UnixNano())
			_ = f.Sync()
			_ = f.Close()

			return func() {
				_ = os.Remove(s.lockPath)
			}, nil
		}

		if os.IsExist(err) {
			if fi, statErr := os.Stat(s.lockPath); statErr == nil {
				if time.Since(fi.ModTime()) > staleThreshold {
					_ = os.Remove(s.lockPath)
					continue
				}
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout acquiring file lock on %s", s.lockPath)
		}

		time.Sleep(25 * time.Millisecond)
	}
}

// Save persists current in-memory configuration to disk with file locking.
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saveLocked()
}

// saveLocked writes the config to disk without acquiring s.mu.
func (s *Store) saveLocked() error {
	unlock, err := s.acquireFileLock()
	if err != nil {
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	defer unlock()

	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config temp file: %w", err)
	}

	if err := os.Rename(tmpFile, s.filePath); err != nil {
		_ = os.Remove(s.filePath)
		if err := os.Rename(tmpFile, s.filePath); err != nil {
			// Fallback direct write
			if writeErr := os.WriteFile(s.filePath, data, 0644); writeErr != nil {
				return fmt.Errorf("failed to write config file: %w", writeErr)
			}
			_ = os.Remove(tmpFile)
		}
	}

	return nil
}

// Load reads the configuration from disk into memory with file locking.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	unlock, err := s.acquireFileLock()
	if err != nil {
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	defer unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg PrismBoxConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Ensure slice fields are non-nil
	if cfg.Inbounds == nil {
		cfg.Inbounds = make([]Inbound, 0)
	}
	if cfg.Outbounds == nil {
		cfg.Outbounds = make([]Outbound, 0)
	}
	if cfg.Groups == nil {
		cfg.Groups = make([]Group, 0)
	}
	if cfg.Routing.Rules == nil {
		cfg.Routing.Rules = make([]Rule, 0)
	}
	if cfg.Routing.RuleSets == nil {
		cfg.Routing.RuleSets = make([]RuleSet, 0)
	}
	if cfg.DNS.Servers == nil {
		cfg.DNS.Servers = make([]DNSServer, 0)
	}
	if cfg.DNS.Rules == nil {
		cfg.DNS.Rules = make([]DNSRule, 0)
	}

	s.config = cfg
	return nil
}

// GetConfig returns a copy of the entire configuration.
func (s *Store) GetConfig() PrismBoxConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// SaveConfig replaces the entire configuration and persists it.
func (s *Store) SaveConfig(cfg PrismBoxConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = cfg
	return s.saveLocked()
}

// SetConfig is an alias for SaveConfig.
func (s *Store) SetConfig(cfg PrismBoxConfig) error {
	return s.SaveConfig(cfg)
}

// ==================== INBOUNDS ====================

// UpdateInbounds replaces all inbounds and saves.
func (s *Store) UpdateInbounds(inbounds []Inbound) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range inbounds {
		if inbounds[i].ID == "" {
			inbounds[i].ID = uuid.New().String()
		}
	}
	s.config.Inbounds = inbounds
	return s.saveLocked()
}

// SetInbounds is an alias for UpdateInbounds.
func (s *Store) SetInbounds(inbounds []Inbound) error {
	return s.UpdateInbounds(inbounds)
}

// GetInbounds returns a copy of all configured inbounds.
func (s *Store) GetInbounds() []Inbound {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Inbound, len(s.config.Inbounds))
	copy(result, s.config.Inbounds)
	return result
}

// AddInbound adds a new inbound, generating an ID if missing.
func (s *Store) AddInbound(inbound Inbound) (Inbound, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if inbound.ID == "" {
		inbound.ID = uuid.New().String()
	}
	s.config.Inbounds = append(s.config.Inbounds, inbound)
	if err := s.saveLocked(); err != nil {
		return inbound, err
	}
	return inbound, nil
}

// UpdateInbound updates an existing inbound by ID.
func (s *Store) UpdateInbound(id string, inbound Inbound) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.config.Inbounds {
		if item.ID == id {
			inbound.ID = id
			s.config.Inbounds[i] = inbound
			return s.saveLocked()
		}
	}
	return fmt.Errorf("%w: inbound %s", ErrNotFound, id)
}

// DeleteInbound deletes an inbound by ID.
func (s *Store) DeleteInbound(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.config.Inbounds {
		if item.ID == id {
			s.config.Inbounds = append(s.config.Inbounds[:i], s.config.Inbounds[i+1:]...)
			return s.saveLocked()
		}
	}
	return fmt.Errorf("%w: inbound %s", ErrNotFound, id)
}

// ==================== OUTBOUNDS ====================

// UpdateOutbounds replaces all outbounds and saves.
func (s *Store) UpdateOutbounds(outbounds []Outbound) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range outbounds {
		if outbounds[i].ID == "" {
			outbounds[i].ID = uuid.New().String()
		}
	}
	s.config.Outbounds = outbounds
	return s.saveLocked()
}

// SetOutbounds is an alias for UpdateOutbounds.
func (s *Store) SetOutbounds(outbounds []Outbound) error {
	return s.UpdateOutbounds(outbounds)
}

// GetOutbounds returns a copy of all configured outbounds.
func (s *Store) GetOutbounds() []Outbound {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Outbound, len(s.config.Outbounds))
	copy(result, s.config.Outbounds)
	return result
}

// AddOutbound adds a new outbound, generating an ID if missing.
func (s *Store) AddOutbound(outbound Outbound) (Outbound, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if outbound.ID == "" {
		outbound.ID = uuid.New().String()
	}
	s.config.Outbounds = append(s.config.Outbounds, outbound)
	if err := s.saveLocked(); err != nil {
		return outbound, err
	}
	return outbound, nil
}

// UpdateOutbound updates an existing outbound by ID.
func (s *Store) UpdateOutbound(id string, outbound Outbound) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.config.Outbounds {
		if item.ID == id {
			outbound.ID = id
			s.config.Outbounds[i] = outbound
			return s.saveLocked()
		}
	}
	return fmt.Errorf("%w: outbound %s", ErrNotFound, id)
}

// DeleteOutbound deletes an outbound by ID.
func (s *Store) DeleteOutbound(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.config.Outbounds {
		if item.ID == id {
			s.config.Outbounds = append(s.config.Outbounds[:i], s.config.Outbounds[i+1:]...)
			return s.saveLocked()
		}
	}
	return fmt.Errorf("%w: outbound %s", ErrNotFound, id)
}

// UpdateOutboundLatency updates latency for an outbound by ID.
func (s *Store) UpdateOutboundLatency(id string, latency int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.config.Outbounds {
		if item.ID == id {
			s.config.Outbounds[i].Latency = latency
			return s.saveLocked()
		}
	}
	return fmt.Errorf("%w: outbound %s", ErrNotFound, id)
}

// ==================== GROUPS ====================

// UpdateGroups replaces all groups and saves.
func (s *Store) UpdateGroups(groups []Group) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range groups {
		if groups[i].ID == "" {
			groups[i].ID = uuid.New().String()
		}
	}
	s.config.Groups = groups
	return s.saveLocked()
}

// SetGroups is an alias for UpdateGroups.
func (s *Store) SetGroups(groups []Group) error {
	return s.UpdateGroups(groups)
}

// GetGroups returns a copy of all configured groups.
func (s *Store) GetGroups() []Group {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Group, len(s.config.Groups))
	copy(result, s.config.Groups)
	return result
}

// AddGroup adds a new group, generating an ID if missing.
func (s *Store) AddGroup(group Group) (Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	s.config.Groups = append(s.config.Groups, group)
	if err := s.saveLocked(); err != nil {
		return group, err
	}
	return group, nil
}

// UpdateGroup updates an existing group by ID.
func (s *Store) UpdateGroup(id string, group Group) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.config.Groups {
		if item.ID == id {
			group.ID = id
			s.config.Groups[i] = group
			return s.saveLocked()
		}
	}
	return fmt.Errorf("%w: group %s", ErrNotFound, id)
}

// DeleteGroup deletes a group by ID.
func (s *Store) DeleteGroup(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.config.Groups {
		if item.ID == id {
			s.config.Groups = append(s.config.Groups[:i], s.config.Groups[i+1:]...)
			return s.saveLocked()
		}
	}
	return fmt.Errorf("%w: group %s", ErrNotFound, id)
}

// ==================== ROUTING & RULESETS ====================

// UpdateRouting replaces the routing configuration and saves.
func (s *Store) UpdateRouting(routing Routing) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range routing.Rules {
		if routing.Rules[i].ID == "" {
			routing.Rules[i].ID = uuid.New().String()
		}
	}
	for i := range routing.RuleSets {
		if routing.RuleSets[i].ID == "" {
			routing.RuleSets[i].ID = uuid.New().String()
		}
	}
	s.config.Routing = routing
	return s.saveLocked()
}

// SetRouting is an alias for UpdateRouting.
func (s *Store) SetRouting(routing Routing) error {
	return s.UpdateRouting(routing)
}

// GetRouting returns the current routing configuration.
func (s *Store) GetRouting() Routing {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.config.Routing
}

// GetRuleSets returns a copy of all configured rule sets.
func (s *Store) GetRuleSets() []RuleSet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]RuleSet, len(s.config.Routing.RuleSets))
	copy(result, s.config.Routing.RuleSets)
	return result
}

// AddRuleSet adds a new rule set.
func (s *Store) AddRuleSet(rs RuleSet) (RuleSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rs.ID == "" {
		rs.ID = uuid.New().String()
	}
	s.config.Routing.RuleSets = append(s.config.Routing.RuleSets, rs)
	if err := s.saveLocked(); err != nil {
		return rs, err
	}
	return rs, nil
}

// UpdateRuleSet updates an existing rule set by ID.
func (s *Store) UpdateRuleSet(id string, rs RuleSet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.config.Routing.RuleSets {
		if item.ID == id {
			rs.ID = id
			s.config.Routing.RuleSets[i] = rs
			return s.saveLocked()
		}
	}
	return fmt.Errorf("%w: rule set %s", ErrNotFound, id)
}

// DeleteRuleSet deletes a rule set by ID.
func (s *Store) DeleteRuleSet(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.config.Routing.RuleSets {
		if item.ID == id {
			s.config.Routing.RuleSets = append(s.config.Routing.RuleSets[:i], s.config.Routing.RuleSets[i+1:]...)
			return s.saveLocked()
		}
	}
	return fmt.Errorf("%w: rule set %s", ErrNotFound, id)
}

// ==================== DNS ====================

// UpdateDNS replaces the DNS configuration and saves.
func (s *Store) UpdateDNS(dns DNSConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range dns.Rules {
		if dns.Rules[i].ID == "" {
			dns.Rules[i].ID = uuid.New().String()
		}
	}
	s.config.DNS = dns
	return s.saveLocked()
}

// SetDNS is an alias for UpdateDNS.
func (s *Store) SetDNS(dns DNSConfig) error {
	return s.UpdateDNS(dns)
}

// GetDNS returns the current DNS configuration.
func (s *Store) GetDNS() DNSConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.config.DNS
}

// SelectGroupOutbound selects an outbound tag for a selector group.
func (s *Store) SelectGroupOutbound(groupID string, selectedTag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, g := range s.config.Groups {
		if g.ID == groupID || g.Tag == groupID {
			return s.saveLocked()
		}
	}
	return fmt.Errorf("%w: group %s", ErrNotFound, groupID)
}

// GetRules returns a copy of all configured routing rules.
func (s *Store) GetRules() []Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Rule, len(s.config.Routing.Rules))
	copy(result, s.config.Routing.Rules)
	return result
}

// AddRule adds a new routing rule.
func (s *Store) AddRule(rule Rule) (Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	s.config.Routing.Rules = append(s.config.Routing.Rules, rule)
	if err := s.saveLocked(); err != nil {
		return rule, err
	}
	return rule, nil
}

// UpdateRule updates an existing routing rule by ID.
func (s *Store) UpdateRule(id string, rule Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.config.Routing.Rules {
		if item.ID == id {
			rule.ID = id
			s.config.Routing.Rules[i] = rule
			return s.saveLocked()
		}
	}
	return fmt.Errorf("%w: rule %s", ErrNotFound, id)
}

// DeleteRule deletes a routing rule by ID.
func (s *Store) DeleteRule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.config.Routing.Rules {
		if item.ID == id {
			s.config.Routing.Rules = append(s.config.Routing.Rules[:i], s.config.Routing.Rules[i+1:]...)
			return s.saveLocked()
		}
	}
	return fmt.Errorf("%w: rule %s", ErrNotFound, id)
}

