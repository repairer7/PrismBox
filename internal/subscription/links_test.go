package subscription

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestParseHysteria2_URLEncodedPassword(t *testing.T) {
	// URL-encoded password test (%21 -> !)
	link := "hysteria2://samplePassword%21@198.51.100.1:20443?sni=example.com&insecure=1&allowInsecure=1#Test-Node"
	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse hysteria2: %v", err)
	}

	if out.Type != "hysteria2" {
		t.Errorf("expected type hysteria2, got %s", out.Type)
	}
	if out.Tag != "Test-Node" {
		t.Errorf("expected tag Test-Node, got %s", out.Tag)
	}
	if out.Server != "198.51.100.1" {
		t.Errorf("expected server 198.51.100.1, got %s", out.Server)
	}
	if out.Port != 20443 {
		t.Errorf("expected port 20443, got %d", out.Port)
	}

	password, _ := out.Settings["password"].(string)
	if password != "samplePassword!" {
		t.Errorf("expected password 'samplePassword!', got %q", password)
	}

	tlsMap, ok := out.Settings["tls"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tls map in settings, got %v", out.Settings["tls"])
	}
	if tlsMap["enabled"] != true {
		t.Errorf("expected tls.enabled true, got %v", tlsMap["enabled"])
	}
	if tlsMap["server_name"] != "example.com" {
		t.Errorf("expected tls.server_name 'example.com', got %v", tlsMap["server_name"])
	}
	if tlsMap["insecure"] != true {
		t.Errorf("expected tls.insecure true, got %v", tlsMap["insecure"])
	}

	// 2. URL-encoded password with %21, %40, %3A
	link2 := "hysteria2://user%40domain%3Apass%21@1.2.3.4:443?sni=example.com#TestEncoded"
	out2, err := ParseLink(link2)
	if err != nil {
		t.Fatalf("failed to parse hysteria2 with encoded password: %v", err)
	}
	pass2, _ := out2.Settings["password"].(string)
	if pass2 != "user@domain:pass!" {
		t.Errorf("expected password 'user@domain:pass!', got %q", pass2)
	}
	if out2.Tag != "TestEncoded" {
		t.Errorf("expected tag 'TestEncoded', got %q", out2.Tag)
	}
}

func TestParseHysteria2_Hy2Alias(t *testing.T) {
	link := "hy2://simplepass@hy2.example.com:443#Hy2Node"
	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse hy2 alias: %v", err)
	}

	if out.Type != "hysteria2" {
		t.Errorf("expected type hysteria2, got %s", out.Type)
	}
	if out.Tag != "Hy2Node" {
		t.Errorf("expected tag Hy2Node, got %s", out.Tag)
	}
	if out.Server != "hy2.example.com" || out.Port != 443 {
		t.Errorf("server/port mismatch: %s:%d", out.Server, out.Port)
	}
	if out.Settings["password"] != "simplepass" {
		t.Errorf("password mismatch: %v", out.Settings["password"])
	}
}

func TestParseHysteria2_ObfsAndBandwidth(t *testing.T) {
	link := "hysteria2://pass@hy2.example.com:8443?obfs=salamander&obfs-password=obfspass123&up_mbps=100&down_mbps=200#ObfsNode"
	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse hysteria2 with obfs: %v", err)
	}

	obfsMap, ok := out.Settings["obfs"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected obfs map, got %v", out.Settings["obfs"])
	}
	if obfsMap["type"] != "salamander" || obfsMap["password"] != "obfspass123" {
		t.Errorf("obfs settings mismatch: %+v", obfsMap)
	}
	if out.Settings["up_mbps"] != 100 {
		t.Errorf("expected up_mbps 100, got %v", out.Settings["up_mbps"])
	}
	if out.Settings["down_mbps"] != 200 {
		t.Errorf("expected down_mbps 200, got %v", out.Settings["down_mbps"])
	}
}

func TestParseHysteria2_AllowInsecure_vs_Insecure(t *testing.T) {
	// Case A: allowInsecure=1
	linkA := "hysteria2://pass@hy2.example.com:443?allowInsecure=1#InsecureA"
	outA, err := ParseLink(linkA)
	if err != nil {
		t.Fatalf("failed to parse linkA: %v", err)
	}
	tlsA, _ := outA.Settings["tls"].(map[string]interface{})
	if tlsA["insecure"] != true {
		t.Errorf("expected insecure true for allowInsecure=1")
	}

	// Case B: insecure=1
	linkB := "hysteria2://pass@hy2.example.com:443?insecure=1#InsecureB"
	outB, err := ParseLink(linkB)
	if err != nil {
		t.Fatalf("failed to parse linkB: %v", err)
	}
	tlsB, _ := outB.Settings["tls"].(map[string]interface{})
	if tlsB["insecure"] != true {
		t.Errorf("expected insecure true for insecure=1")
	}

	// Case C: neither set
	linkC := "hysteria2://pass@hy2.example.com:443#SecureC"
	outC, err := ParseLink(linkC)
	if err != nil {
		t.Fatalf("failed to parse linkC: %v", err)
	}
	tlsC, _ := outC.Settings["tls"].(map[string]interface{})
	if tlsC["insecure"] == true {
		t.Errorf("expected insecure false/nil when not specified")
	}
}

func TestParseVLESS_Reality(t *testing.T) {
	link := "vless://11111111-2222-3333-4444-555555555555@reality.example.com:443?security=reality&pbk=pubkey123&sid=shortid123&sni=reality.example.com&fp=chrome#VLESSReality"
	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse vless reality: %v", err)
	}

	if out.Type != "vless" {
		t.Errorf("expected type vless, got %s", out.Type)
	}
	if out.Tag != "VLESSReality" {
		t.Errorf("expected tag VLESSReality, got %s", out.Tag)
	}
	if out.Server != "reality.example.com" || out.Port != 443 {
		t.Errorf("server/port mismatch: %s:%d", out.Server, out.Port)
	}
	if out.Settings["uuid"] != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("uuid mismatch: %v", out.Settings["uuid"])
	}

	tlsMap, ok := out.Settings["tls"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tls map, got %v", out.Settings["tls"])
	}
	if tlsMap["server_name"] != "reality.example.com" {
		t.Errorf("server_name mismatch: %v", tlsMap["server_name"])
	}

	realityMap, ok := tlsMap["reality"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected reality map, got %v", tlsMap["reality"])
	}
	if realityMap["enabled"] != true || realityMap["public_key"] != "pubkey123" || realityMap["short_id"] != "shortid123" {
		t.Errorf("reality config mismatch: %+v", realityMap)
	}

	utlsMap, ok := tlsMap["utls"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected utls map, got %v", tlsMap["utls"])
	}
	if utlsMap["fingerprint"] != "chrome" {
		t.Errorf("expected fingerprint chrome, got %v", utlsMap["fingerprint"])
	}
}

func TestParseVLESS_PlainTLS(t *testing.T) {
	link := "vless://22222222-3333-4444-5555-666666666666@tls.example.com:443?security=tls&sni=tls.example.com&alpn=h2,http/1.1&insecure=1#VLESSTLS"
	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse vless tls: %v", err)
	}

	tlsMap, ok := out.Settings["tls"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tls map, got %v", out.Settings["tls"])
	}
	if tlsMap["enabled"] != true || tlsMap["server_name"] != "tls.example.com" || tlsMap["insecure"] != true {
		t.Errorf("tls settings mismatch: %+v", tlsMap)
	}
	alpn, _ := tlsMap["alpn"].([]string)
	if len(alpn) != 2 || alpn[0] != "h2" || alpn[1] != "http/1.1" {
		t.Errorf("alpn mismatch: %v", alpn)
	}
}

func TestParseVLESS_GRPCTransport(t *testing.T) {
	link := "vless://uuid123@grpc.example.com:443?security=tls&sni=grpc.example.com&type=grpc&serviceName=MyVlessService#GRPCNode"
	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse vless grpc: %v", err)
	}

	transport, ok := out.Settings["transport"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected transport map, got %v", out.Settings["transport"])
	}
	if transport["type"] != "grpc" {
		t.Errorf("expected type grpc, got %v", transport["type"])
	}
	if transport["service_name"] != "MyVlessService" {
		t.Errorf("expected service_name MyVlessService, got %v", transport["service_name"])
	}
}

func TestParseVLESS_WSTransport(t *testing.T) {
	link := "vless://uuid123@ws.example.com:443?security=tls&sni=ws.example.com&type=ws&path=/vless-ws&host=ws.example.com#WSNode"
	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse vless ws: %v", err)
	}

	transport, ok := out.Settings["transport"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected transport map, got %v", out.Settings["transport"])
	}
	if transport["type"] != "ws" || transport["path"] != "/vless-ws" {
		t.Errorf("transport mismatch: %+v", transport)
	}
	headers, _ := transport["headers"].(map[string]string)
	if headers["Host"] != "ws.example.com" {
		t.Errorf("expected Host header ws.example.com, got %v", headers["Host"])
	}
}

func TestParseVLESS_FlowVision(t *testing.T) {
	link := "vless://uuid123@flow.example.com:443?security=reality&pbk=pbk&sid=sid&flow=xtls-rprx-vision#VisionNode"
	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse vless vision: %v", err)
	}

	if out.Settings["flow"] != "xtls-rprx-vision" {
		t.Errorf("expected flow xtls-rprx-vision, got %v", out.Settings["flow"])
	}
}

func TestParseVMess_Base64Encodings(t *testing.T) {
	jsonPayload := `{
		"v": "2",
		"ps": "VMessStandard",
		"add": "vmess.example.com",
		"port": 443,
		"id": "e44c2079-d2b3-4f96-8ac0-4e3e3b3e3e3e",
		"aid": 0,
		"scy": "auto",
		"net": "ws",
		"tls": "tls",
		"path": "/ws",
		"host": "vmess.example.com"
	}`

	// 1. Standard base64 with padding
	b64Std := base64.StdEncoding.EncodeToString([]byte(jsonPayload))
	outStd, err := ParseLink("vmess://" + b64Std)
	if err != nil {
		t.Fatalf("failed to parse standard base64 vmess: %v", err)
	}
	if outStd.Tag != "VMessStandard" || outStd.Server != "vmess.example.com" || outStd.Port != 443 {
		t.Errorf("vmess standard mismatch: %+v", outStd)
	}

	// 2. URL-safe base64
	b64URL := base64.URLEncoding.EncodeToString([]byte(jsonPayload))
	outURL, err := ParseLink("vmess://" + b64URL)
	if err != nil {
		t.Fatalf("failed to parse urlsafe base64 vmess: %v", err)
	}
	if outURL.Tag != "VMessStandard" || outURL.Server != "vmess.example.com" {
		t.Errorf("vmess urlsafe mismatch: %+v", outURL)
	}
}

func TestParseVMess_MissingPadding(t *testing.T) {
	jsonPayload := `{"v":"2","ps":"NoPadNode","add":"1.2.3.4","port":8443,"id":"my-uuid","net":"tcp"}`
	b64 := base64.StdEncoding.EncodeToString([]byte(jsonPayload))
	unpadded := strings.TrimRight(b64, "=")

	out, err := ParseLink("vmess://" + unpadded)
	if err != nil {
		t.Fatalf("failed to parse unpadded vmess base64: %v", err)
	}
	if out.Tag != "NoPadNode" || out.Server != "1.2.3.4" || out.Port != 8443 {
		t.Errorf("unpadded vmess mismatch: %+v", out)
	}
}

func TestParseVMess_StringPort(t *testing.T) {
	jsonPayload := `{"v":"2","ps":"StringPortNode","add":"1.2.3.4","port":"8443","aid":"16","id":"my-uuid"}`
	b64 := base64.StdEncoding.EncodeToString([]byte(jsonPayload))

	out, err := ParseLink("vmess://" + b64)
	if err != nil {
		t.Fatalf("failed to parse string port vmess: %v", err)
	}
	if out.Port != 8443 {
		t.Errorf("expected port 8443, got %d", out.Port)
	}
	if out.Settings["alter_id"] != 16 {
		t.Errorf("expected alter_id 16, got %v", out.Settings["alter_id"])
	}
}

func TestParseTrojan_Basic(t *testing.T) {
	link := "trojan://mypassword%21@trojan.example.com:443?sni=trojan.example.com#TrojanBasic"
	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse trojan basic: %v", err)
	}

	if out.Type != "trojan" {
		t.Errorf("expected type trojan, got %s", out.Type)
	}
	if out.Tag != "TrojanBasic" {
		t.Errorf("expected tag TrojanBasic, got %s", out.Tag)
	}
	if out.Server != "trojan.example.com" || out.Port != 443 {
		t.Errorf("server/port mismatch: %s:%d", out.Server, out.Port)
	}
	if out.Settings["password"] != "mypassword!" {
		t.Errorf("password mismatch: %v", out.Settings["password"])
	}
	tlsMap, _ := out.Settings["tls"].(map[string]interface{})
	if tlsMap["enabled"] != true || tlsMap["server_name"] != "trojan.example.com" {
		t.Errorf("tls mismatch: %+v", tlsMap)
	}
}

func TestParseTrojan_WebSocket(t *testing.T) {
	link := "trojan://password123@trojan.example.com:443?type=ws&host=trojan.example.com&path=/trojan-ws#TrojanWS"
	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse trojan ws: %v", err)
	}

	transport, ok := out.Settings["transport"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected transport map, got %v", out.Settings["transport"])
	}
	if transport["type"] != "ws" || transport["path"] != "/trojan-ws" {
		t.Errorf("transport mismatch: %+v", transport)
	}
	headers, _ := transport["headers"].(map[string]string)
	if headers["Host"] != "trojan.example.com" {
		t.Errorf("expected Host trojan.example.com, got %v", headers["Host"])
	}
}

func TestParseShadowsocks_SIP002(t *testing.T) {
	userinfo := base64.URLEncoding.EncodeToString([]byte("aes-256-gcm:mysecretpassword"))
	link := "ss://" + userinfo + "@ss.example.com:8388/?plugin=obfs-local%3Bobfs%3Dhttp%3Bobfs-host%3Dexample.com#SIP002Node"

	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse ss sip002: %v", err)
	}

	if out.Type != "shadowsocks" {
		t.Errorf("expected shadowsocks, got %s", out.Type)
	}
	if out.Tag != "SIP002Node" {
		t.Errorf("expected SIP002Node, got %s", out.Tag)
	}
	if out.Server != "ss.example.com" || out.Port != 8388 {
		t.Errorf("server/port mismatch: %s:%d", out.Server, out.Port)
	}
	if out.Settings["method"] != "aes-256-gcm" || out.Settings["password"] != "mysecretpassword" {
		t.Errorf("settings mismatch: %+v", out.Settings)
	}
	if out.Settings["plugin"] != "obfs-local" {
		t.Errorf("expected plugin obfs-local, got %v", out.Settings["plugin"])
	}
	if out.Settings["plugin_opts"] != "obfs=http;obfs-host=example.com" {
		t.Errorf("expected plugin_opts 'obfs=http;obfs-host=example.com', got %v", out.Settings["plugin_opts"])
	}
}

func TestParseShadowsocks_Legacy(t *testing.T) {
	legacyStr := "chacha20-ietf-poly1305:mypassword@legacy.example.com:8389"
	b64 := base64.StdEncoding.EncodeToString([]byte(legacyStr))
	link := "ss://" + b64 + "#LegacySS"

	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse legacy shadowsocks: %v", err)
	}

	if out.Type != "shadowsocks" || out.Tag != "LegacySS" {
		t.Errorf("type/tag mismatch: %s / %s", out.Type, out.Tag)
	}
	if out.Server != "legacy.example.com" || out.Port != 8389 {
		t.Errorf("server/port mismatch: %s:%d", out.Server, out.Port)
	}
	if out.Settings["method"] != "chacha20-ietf-poly1305" || out.Settings["password"] != "mypassword" {
		t.Errorf("settings mismatch: %+v", out.Settings)
	}
}

func TestParseTUIC_AllParams(t *testing.T) {
	link := "tuic://my-uuid-1234:my-password-5678@tuic.example.com:8443?congestion_controller=bbr&alpn=h3&sni=tuic.example.com&reduce_rtt=1&disable_sni=1#TUICNode"
	out, err := ParseLink(link)
	if err != nil {
		t.Fatalf("failed to parse tuic: %v", err)
	}

	if out.Type != "tuic" {
		t.Errorf("expected type tuic, got %s", out.Type)
	}
	if out.Tag != "TUICNode" {
		t.Errorf("expected tag TUICNode, got %s", out.Tag)
	}
	if out.Server != "tuic.example.com" || out.Port != 8443 {
		t.Errorf("server/port mismatch: %s:%d", out.Server, out.Port)
	}
	if out.Settings["uuid"] != "my-uuid-1234" || out.Settings["password"] != "my-password-5678" {
		t.Errorf("uuid/password mismatch: %+v", out.Settings)
	}
	if out.Settings["congestion_control"] != "bbr" {
		t.Errorf("congestion_control mismatch: %v", out.Settings["congestion_control"])
	}
	if out.Settings["reduce_rtt"] != true {
		t.Errorf("expected reduce_rtt true, got %v", out.Settings["reduce_rtt"])
	}

	tlsMap, ok := out.Settings["tls"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tls map, got %v", out.Settings["tls"])
	}
	if tlsMap["enabled"] != true || tlsMap["server_name"] != "tuic.example.com" || tlsMap["disable_sni"] != true {
		t.Errorf("tls mismatch: %+v", tlsMap)
	}
	alpn, _ := tlsMap["alpn"].([]string)
	if len(alpn) != 1 || alpn[0] != "h3" {
		t.Errorf("expected alpn [h3], got %v", alpn)
	}
}

func TestParseMultipleLinks_Mixed(t *testing.T) {
	text := `
# Commentary line
// Another comment

hysteria2://pass1@hy2.com:443#Hy2Valid
invalid-scheme://something
trojan://pass2@trojan.com:443#TrojanValid

vless://uuid3@vless.com:443#VlessValid
`
	outbounds, errs := ParseMultipleLinks(text)
	if len(outbounds) != 3 {
		t.Fatalf("expected 3 valid outbounds, got %d", len(outbounds))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error for invalid scheme, got %d: %v", len(errs), errs)
	}

	if outbounds[0].Tag != "Hy2Valid" || outbounds[1].Tag != "TrojanValid" || outbounds[2].Tag != "VlessValid" {
		t.Errorf("outbounds tag mismatch: %+v", outbounds)
	}
}

func TestParseLink_EmptyWhitespace(t *testing.T) {
	cases := []string{"", "   ", "\t\r\n"}
	for _, c := range cases {
		_, err := ParseLink(c)
		if err == nil {
			t.Errorf("expected error for empty/whitespace %q, got nil", c)
		}
	}
}

func TestParseLink_InvalidScheme(t *testing.T) {
	cases := []string{
		"http://example.com:80",
		"ftp://example.com:21",
		"unknown://foobar@host:123",
	}
	for _, c := range cases {
		_, err := ParseLink(c)
		if err == nil {
			t.Errorf("expected error for unsupported scheme %q, got nil", c)
		}
	}
}
