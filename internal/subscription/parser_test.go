package subscription

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseClash_AllProtocols(t *testing.T) {
	yamlContent := `
proxies:
  - name: "Clash SS"
    type: ss
    server: ss.example.com
    port: 8388
    cipher: aes-256-gcm
    password: sspassword
    plugin: obfs-local
    plugin-opts:
      mode: http
      host: clash.example.com

  - name: "Clash VMess"
    type: vmess
    server: vmess.example.com
    port: 443
    uuid: 11111111-2222-3333-4444-555555555555
    alterId: 0
    cipher: auto
    tls: true
    servername: vmess.example.com
    skip-cert-verify: true
    client-fingerprint: chrome
    alpn:
      - h2
      - http/1.1
    network: ws
    ws-opts:
      path: /ws
      headers:
        Host: vmess.example.com

  - name: "Clash VLESS Reality"
    type: vless
    server: vless.example.com
    port: 443
    uuid: 22222222-3333-4444-5555-666666666666
    flow: xtls-rprx-vision
    servername: vless.example.com
    reality-opts:
      public-key: mypubkey
      short-id: myshortid
    network: grpc
    grpc-opts:
      grpc-service-name: MyGrpcService

  - name: "Clash Trojan"
    type: trojan
    server: trojan.example.com
    port: 443
    password: trojanpassword
    sni: trojan.example.com
    skip-cert-verify: true
    network: h2
    h2-opts:
      path: /h2path
      host:
        - trojan.example.com

  - name: "Clash Hysteria2"
    type: hysteria2
    server: hy2.example.com
    port: 8443
    auth: hy2password
    sni: hy2.example.com
    skip-cert-verify: true
    obfs: salamander
    obfs-password: obfspassword
    up: "100 Mbps"
    down: "200 Mbps"

  - name: "Clash TUIC"
    type: tuic
    server: tuic.example.com
    port: 8443
    uuid: 33333333-4444-5555-6666-777777777777
    password: tuicpassword
    congestion-controller: bbr
    reduce-rtt: true
    disable-sni: true
`
	outbounds, err := ParseClash([]byte(yamlContent))
	if err != nil {
		t.Fatalf("failed to parse clash YAML: %v", err)
	}

	if len(outbounds) != 6 {
		t.Fatalf("expected 6 outbounds, got %d", len(outbounds))
	}

	// 1. Shadowsocks
	ss := outbounds[0]
	if ss.Type != "shadowsocks" || ss.Tag != "Clash SS" || ss.Server != "ss.example.com" || ss.Port != 8388 {
		t.Errorf("ss mismatch: %+v", ss)
	}
	if ss.Settings["method"] != "aes-256-gcm" || ss.Settings["password"] != "sspassword" || ss.Settings["plugin"] != "obfs-local" {
		t.Errorf("ss settings mismatch: %+v", ss.Settings)
	}

	// 2. VMess
	vmess := outbounds[1]
	if vmess.Type != "vmess" || vmess.Tag != "Clash VMess" || vmess.Server != "vmess.example.com" || vmess.Port != 443 {
		t.Errorf("vmess mismatch: %+v", vmess)
	}
	tlsVmess, _ := vmess.Settings["tls"].(map[string]interface{})
	if tlsVmess["insecure"] != true || tlsVmess["server_name"] != "vmess.example.com" {
		t.Errorf("vmess tls mismatch: %+v", tlsVmess)
	}
	transportVmess, _ := vmess.Settings["transport"].(map[string]interface{})
	if transportVmess["type"] != "ws" || transportVmess["path"] != "/ws" {
		t.Errorf("vmess transport mismatch: %+v", transportVmess)
	}

	// 3. VLESS Reality
	vless := outbounds[2]
	if vless.Type != "vless" || vless.Tag != "Clash VLESS Reality" {
		t.Errorf("vless mismatch: %+v", vless)
	}
	if vless.Settings["flow"] != "xtls-rprx-vision" {
		t.Errorf("expected flow xtls-rprx-vision, got %v", vless.Settings["flow"])
	}
	tlsVless, _ := vless.Settings["tls"].(map[string]interface{})
	reality, _ := tlsVless["reality"].(map[string]interface{})
	if reality["public_key"] != "mypubkey" || reality["short_id"] != "myshortid" {
		t.Errorf("vless reality mismatch: %+v", reality)
	}
	transportVless, _ := vless.Settings["transport"].(map[string]interface{})
	if transportVless["type"] != "grpc" || transportVless["service_name"] != "MyGrpcService" {
		t.Errorf("vless transport mismatch: %+v", transportVless)
	}

	// 4. Trojan
	trojan := outbounds[3]
	if trojan.Type != "trojan" || trojan.Tag != "Clash Trojan" {
		t.Errorf("trojan mismatch: %+v", trojan)
	}
	if trojan.Settings["password"] != "trojanpassword" {
		t.Errorf("trojan password mismatch: %v", trojan.Settings["password"])
	}
	transportTrojan, _ := trojan.Settings["transport"].(map[string]interface{})
	if transportTrojan["type"] != "h2" || transportTrojan["path"] != "/h2path" {
		t.Errorf("trojan transport mismatch: %+v", transportTrojan)
	}

	// 5. Hysteria2
	hy2 := outbounds[4]
	if hy2.Type != "hysteria2" || hy2.Tag != "Clash Hysteria2" {
		t.Errorf("hy2 mismatch: %+v", hy2)
	}
	if hy2.Settings["password"] != "hy2password" {
		t.Errorf("hy2 password mismatch: %v", hy2.Settings["password"])
	}
	obfs, _ := hy2.Settings["obfs"].(map[string]interface{})
	if obfs["type"] != "salamander" || obfs["password"] != "obfspassword" {
		t.Errorf("hy2 obfs mismatch: %+v", obfs)
	}
	if hy2.Settings["up_mbps"] != 100 || hy2.Settings["down_mbps"] != 200 {
		t.Errorf("hy2 bandwidth mismatch: up=%v down=%v", hy2.Settings["up_mbps"], hy2.Settings["down_mbps"])
	}

	// 6. TUIC
	tuic := outbounds[5]
	if tuic.Type != "tuic" || tuic.Tag != "Clash TUIC" {
		t.Errorf("tuic mismatch: %+v", tuic)
	}
	if tuic.Settings["uuid"] != "33333333-4444-5555-6666-777777777777" || tuic.Settings["password"] != "tuicpassword" {
		t.Errorf("tuic credentials mismatch: %+v", tuic.Settings)
	}
	if tuic.Settings["congestion_control"] != "bbr" || tuic.Settings["reduce_rtt"] != true {
		t.Errorf("tuic options mismatch: %+v", tuic.Settings)
	}
	tlsTuic, _ := tuic.Settings["tls"].(map[string]interface{})
	if tlsTuic["disable_sni"] != true {
		t.Errorf("expected disable_sni true, got %v", tlsTuic["disable_sni"])
	}
}

func TestParseClash_Errors(t *testing.T) {
	// Empty content
	_, err := ParseClash([]byte(""))
	if err == nil {
		t.Errorf("expected error on empty clash data")
	}

	// No proxies key
	_, err = ParseClash([]byte("port: 7890\nsocks-port: 7891\n"))
	if err == nil {
		t.Errorf("expected error when proxies key is missing")
	}

	// Invalid YAML
	_, err = ParseClash([]byte("proxies: [invalid yaml {"))
	if err == nil {
		t.Errorf("expected error on invalid yaml")
	}
}

func TestParseBase64_Success(t *testing.T) {
	links := "trojan://pass1@example1.com:443#Node1\ntrojan://pass2@example2.com:443#Node2"
	b64 := base64.StdEncoding.EncodeToString([]byte(links))

	outbounds, err := ParseBase64([]byte(b64))
	if err != nil {
		t.Fatalf("failed to parse base64: %v", err)
	}
	if len(outbounds) != 2 {
		t.Fatalf("expected 2 outbounds, got %d", len(outbounds))
	}
	if outbounds[0].Tag != "Node1" || outbounds[1].Tag != "Node2" {
		t.Errorf("tags mismatch: %+v", outbounds)
	}

	// Plain text without base64
	outbounds2, err := ParseBase64([]byte(links))
	if err != nil {
		t.Fatalf("failed to parse plain text links: %v", err)
	}
	if len(outbounds2) != 2 {
		t.Fatalf("expected 2 outbounds, got %d", len(outbounds2))
	}
}

func TestParseSubscriptionData_AutoDetect(t *testing.T) {
	// Base64 containing hysteria2 link
	hy2Link := "hysteria2://pass@hy2.com:443?sni=hy2.com#Hy2SubNode"
	b64 := base64.StdEncoding.EncodeToString([]byte(hy2Link))

	outbounds, err := ParseSubscriptionData([]byte(b64))
	if err != nil {
		t.Fatalf("failed to autodetect base64: %v", err)
	}
	if len(outbounds) != 1 || outbounds[0].Tag != "Hy2SubNode" {
		t.Errorf("outbound mismatch: %+v", outbounds)
	}

	// Clash YAML
	yamlContent := `
proxies:
  - name: "Clash Node"
    type: trojan
    server: trojan.com
    port: 443
    password: pass
`
	outbounds2, err := ParseSubscriptionData([]byte(yamlContent))
	if err != nil {
		t.Fatalf("failed to autodetect clash: %v", err)
	}
	if len(outbounds2) != 1 || outbounds2[0].Tag != "Clash Node" {
		t.Errorf("clash auto detect mismatch: %+v", outbounds2)
	}
}

func TestFetchSubscription_HTTPServer(t *testing.T) {
	receivedUA := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")

		switch r.URL.Path {
		case "/base64":
			link := "trojan://mypass@trojan.com:443#HTTPNode"
			b64 := base64.StdEncoding.EncodeToString([]byte(link))
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(b64))

		case "/clash":
			yamlData := `
proxies:
  - name: "HTTP Clash Node"
    type: trojan
    server: httpclash.com
    port: 443
    password: pass
`
			w.Header().Set("Content-Type", "application/x-yaml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(yamlData))

		case "/error404":
			http.NotFound(w, r)

		case "/error500":
			w.WriteHeader(http.StatusInternalServerError)

		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	// 1. Base64 endpoint
	outbounds, err := FetchSubscription(ts.URL + "/base64")
	if err != nil {
		t.Fatalf("FetchSubscription /base64 failed: %v", err)
	}
	if len(outbounds) != 1 || outbounds[0].Tag != "HTTPNode" {
		t.Fatalf("expected HTTPNode, got %+v", outbounds)
	}
	if outbounds[0].SubscriptionURL != ts.URL+"/base64" {
		t.Errorf("expected subscription URL set, got %s", outbounds[0].SubscriptionURL)
	}
	if receivedUA != "clash-verge/v1.7.5 ClashMeta/1.18.0 PrismBox/1.0" {
		t.Errorf("User-Agent mismatch: got %q", receivedUA)
	}

	// 2. Clash YAML endpoint
	outbounds2, err := ParseSubscription(ts.URL + "/clash")
	if err != nil {
		t.Fatalf("ParseSubscription /clash failed: %v", err)
	}
	if len(outbounds2) != 1 || outbounds2[0].Tag != "HTTP Clash Node" {
		t.Fatalf("expected HTTP Clash Node, got %+v", outbounds2)
	}

	// 3. Error 404
	_, err = FetchSubscription(ts.URL + "/error404")
	if err == nil {
		t.Errorf("expected error for 404 response")
	}

	// 4. Error 500
	_, err = FetchSubscription(ts.URL + "/error500")
	if err == nil {
		t.Errorf("expected error for 500 response")
	}

	// 5. Empty URL
	_, err = FetchSubscription("")
	if err == nil {
		t.Errorf("expected error for empty URL")
	}
}
