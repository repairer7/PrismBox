package config

import (
	"encoding/json"
)

// PrismBoxConfig represents PrismBox's internal configuration edited via WebUI.
type PrismBoxConfig struct {
	Inbounds  []Inbound  `json:"inbounds"`
	Outbounds []Outbound `json:"outbounds"`
	Groups    []Group    `json:"groups"`
	Routing   Routing    `json:"routing"`
	DNS       DNSConfig  `json:"dns"`
}

// Inbound defines an incoming traffic interface (tproxy, tun, mixed, etc.).
type Inbound struct {
	ID      string `json:"id"`
	Tag     string `json:"tag"`
	Type    string `json:"type"` // "tproxy", "tun", "mixed"
	Enabled bool   `json:"enabled"`

	// Common
	Listen string `json:"listen,omitempty"`
	Port   int    `json:"port,omitempty"`

	// TProxy specific
	Network string `json:"network,omitempty"` // "tcp", "udp", "tcp,udp"

	// TUN specific
	InterfaceName string   `json:"interface_name,omitempty"`
	MTU           int      `json:"mtu,omitempty"`
	AutoRoute     bool     `json:"auto_route,omitempty"`
	StrictRoute   bool     `json:"strict_route,omitempty"`
	Stack         string   `json:"stack,omitempty"` // "system", "gvisor", "mixed"
	Inet4Address  []string `json:"inet4_address,omitempty"`
	Inet6Address  []string `json:"inet6_address,omitempty"`

	// Mixed specific
	SetSystemProxy bool `json:"set_system_proxy,omitempty"`

	// Sniff settings
	Sniff                    bool `json:"sniff,omitempty"`
	SniffOverrideDestination bool `json:"sniff_override_destination,omitempty"`
}

// UnmarshalJSON handles both port and listen_port, as well as string/slice address formats.
func (i *Inbound) UnmarshalJSON(data []byte) error {
	type Alias Inbound
	aux := struct {
		ListenPort int             `json:"listen_port,omitempty"`
		RawInet4   json.RawMessage `json:"inet4_address,omitempty"`
		RawInet6   json.RawMessage `json:"inet6_address,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(i),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if i.Port == 0 && aux.ListenPort != 0 {
		i.Port = aux.ListenPort
	}

	if len(aux.RawInet4) > 0 {
		var s string
		if err := json.Unmarshal(aux.RawInet4, &s); err == nil && s != "" {
			i.Inet4Address = []string{s}
		} else {
			var slice []string
			if err := json.Unmarshal(aux.RawInet4, &slice); err == nil {
				i.Inet4Address = slice
			}
		}
	}

	if len(aux.RawInet6) > 0 {
		var s string
		if err := json.Unmarshal(aux.RawInet6, &s); err == nil && s != "" {
			i.Inet6Address = []string{s}
		} else {
			var slice []string
			if err := json.Unmarshal(aux.RawInet6, &slice); err == nil {
				i.Inet6Address = slice
			}
		}
	}

	return nil
}

// MarshalJSON provides both port and listen_port for web UI compatibility.
func (i Inbound) MarshalJSON() ([]byte, error) {
	type Alias Inbound
	return json.Marshal(&struct {
		Alias
		ListenPort int `json:"listen_port,omitempty"`
	}{
		Alias:      Alias(i),
		ListenPort: i.Port,
	})
}

// Outbound defines an outbound proxy node.
type Outbound struct {
	ID       string `json:"id"`
	Tag      string `json:"tag"`
	Type     string `json:"type"` // "vless","vmess","trojan","shadowsocks","hysteria2","tuic","direct","block","dns"
	Enabled  bool   `json:"enabled"`
	Server   string `json:"server,omitempty"`
	Port     int    `json:"server_port,omitempty"`

	// Protocol-specific fields
	Settings map[string]interface{} `json:"settings,omitempty"`

	// Subscription metadata
	SubscriptionURL string `json:"subscription_url,omitempty"`
	SubscriptionTag string `json:"subscription_tag,omitempty"`

	// Speed test result in milliseconds
	Latency int `json:"latency,omitempty"`
}

// UnmarshalJSON handles both port and server_port field names.
func (o *Outbound) UnmarshalJSON(data []byte) error {
	type Alias Outbound
	aux := struct {
		AltPort int `json:"port,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(o),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if o.Port == 0 && aux.AltPort != 0 {
		o.Port = aux.AltPort
	}
	return nil
}

// MarshalJSON provides both server_port and port for compatibility.
func (o Outbound) MarshalJSON() ([]byte, error) {
	type Alias Outbound
	return json.Marshal(&struct {
		Alias
		Port int `json:"port,omitempty"`
	}{
		Alias: Alias(o),
		Port:  o.Port,
	})
}

// Group defines an outbound group (selector or urltest).
type Group struct {
	ID        string   `json:"id"`
	Tag       string   `json:"tag"`
	Type      string   `json:"type"` // "selector", "urltest"
	Outbounds []string `json:"outbounds"`
	URL       string   `json:"url,omitempty"`
	Interval  string   `json:"interval,omitempty"`
	Tolerance int      `json:"tolerance,omitempty"`
	Selected  string   `json:"selected,omitempty"`
}

// Routing defines traffic routing and rule settings.
type Routing struct {
	Sniff                    bool      `json:"sniff"`
	SniffOverrideDestination bool      `json:"sniff_override_destination"`
	DomainStrategy           string    `json:"domain_strategy,omitempty"`
	Rules                    []Rule    `json:"rules"`
	RuleSets                 []RuleSet `json:"rule_sets"`
}

// Rule defines a routing match condition and destination outbound.
type Rule struct {
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`

	// Match conditions
	Domain        []string `json:"domain,omitempty"`
	DomainSuffix  []string `json:"domain_suffix,omitempty"`
	DomainKeyword []string `json:"domain_keyword,omitempty"`
	DomainRegex   []string `json:"domain_regex,omitempty"`
	IPCidr        []string `json:"ip_cidr,omitempty"`
	GeoIP         []string `json:"geoip,omitempty"`
	GeoSite       []string `json:"geosite,omitempty"`
	RuleSet       []string `json:"rule_set,omitempty"`
	Protocol      []string `json:"protocol,omitempty"`
	Port          []int    `json:"port,omitempty"`
	PortRange     []string `json:"port_range,omitempty"`
	Network       []string `json:"network,omitempty"`
	SourceIPCidr  []string `json:"source_ip_cidr,omitempty"`

	// Action
	Outbound string `json:"outbound"`
}

// UnmarshalJSON permits flexible parsing of network/protocol as string or array.
func (r *Rule) UnmarshalJSON(data []byte) error {
	type Alias Rule
	aux := struct {
		RawNetwork  json.RawMessage `json:"network,omitempty"`
		RawProtocol json.RawMessage `json:"protocol,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.RawNetwork) > 0 {
		var s string
		if err := json.Unmarshal(aux.RawNetwork, &s); err == nil && s != "" {
			r.Network = []string{s}
		} else {
			var slice []string
			if err := json.Unmarshal(aux.RawNetwork, &slice); err == nil {
				r.Network = slice
			}
		}
	}
	if len(aux.RawProtocol) > 0 {
		var s string
		if err := json.Unmarshal(aux.RawProtocol, &s); err == nil && s != "" {
			r.Protocol = []string{s}
		} else {
			var slice []string
			if err := json.Unmarshal(aux.RawProtocol, &slice); err == nil {
				r.Protocol = slice
			}
		}
	}
	return nil
}

// RuleSet defines a remote or local rule set definition.
type RuleSet struct {
	ID             string `json:"id"`
	Tag            string `json:"tag"`
	Type           string `json:"type"`   // "remote"
	Format         string `json:"format"` // "binary" (.srs) or "source" (.json)
	URL            string `json:"url,omitempty"`
	DownloadDetour string `json:"download_detour,omitempty"`
	UpdateInterval string `json:"update_interval,omitempty"`
	Interval       string `json:"interval,omitempty"` // alias for compatibility
}

// DNSConfig defines DNS resolver settings.
type DNSConfig struct {
	Enabled          bool        `json:"enabled"`
	Servers          []DNSServer `json:"servers,omitempty"`
	Rules            []DNSRule   `json:"rules,omitempty"`
	Strategy         string      `json:"strategy,omitempty"`
	DisableCache     bool        `json:"disable_cache,omitempty"`
	DisableExpire    bool        `json:"disable_expire,omitempty"`
	IndependentCache bool        `json:"independent_cache,omitempty"`
}

type dnsSettingsPayload struct {
	Strategy         string `json:"strategy,omitempty"`
	DisableCache     bool   `json:"disable_cache,omitempty"`
	DisableExpire    bool   `json:"disable_expire,omitempty"`
	IndependentCache bool   `json:"independent_cache,omitempty"`
}

// UnmarshalJSON extracts settings from nested settings object or top-level.
func (d *DNSConfig) UnmarshalJSON(data []byte) error {
	type Alias DNSConfig
	aux := struct {
		Settings *dnsSettingsPayload `json:"settings,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Settings != nil {
		if d.Strategy == "" && aux.Settings.Strategy != "" {
			d.Strategy = aux.Settings.Strategy
		}
		if !d.DisableCache && aux.Settings.DisableCache {
			d.DisableCache = true
		}
		if !d.DisableExpire && aux.Settings.DisableExpire {
			d.DisableExpire = true
		}
		if !d.IndependentCache && aux.Settings.IndependentCache {
			d.IndependentCache = true
		}
	}
	return nil
}

// MarshalJSON includes nested settings object for UI compatibility.
func (d DNSConfig) MarshalJSON() ([]byte, error) {
	type Alias DNSConfig
	return json.Marshal(&struct {
		Alias
		Settings dnsSettingsPayload `json:"settings"`
	}{
		Alias: Alias(d),
		Settings: dnsSettingsPayload{
			Strategy:         d.Strategy,
			DisableCache:     d.DisableCache,
			DisableExpire:    d.DisableExpire,
			IndependentCache: d.IndependentCache,
		},
	})
}

// DNSServer defines an upstream DNS server.
type DNSServer struct {
	Tag             string `json:"tag"`
	Address         string `json:"address"`
	AddressResolver string `json:"address_resolver,omitempty"`
	Detour          string `json:"detour,omitempty"`
	Strategy        string `json:"strategy,omitempty"`
}

// DNSRule defines a rule mapping domain/geosite criteria to a DNS server tag.
type DNSRule struct {
	ID            string   `json:"id,omitempty"`
	Domain        []string `json:"domain,omitempty"`
	DomainSuffix  []string `json:"domain_suffix,omitempty"`
	DomainKeyword []string `json:"domain_keyword,omitempty"`
	GeoSite       []string `json:"geosite,omitempty"`
	RuleSet       []string `json:"rule_set,omitempty"`
	Outbound      string   `json:"outbound,omitempty"`
	Server        string   `json:"server"`
	ClashMode     string   `json:"clash_mode,omitempty"`
}
