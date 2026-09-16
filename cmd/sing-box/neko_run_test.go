package boxmain

import (
	"testing"
	"time"
)

// replicates the nekoray core path: Create with legacy-style JSON config
func TestNekoCreate(t *testing.T) {
	const config = `{
		"log": {"level": "warn"},
		"dns": {
			"servers": [
				{"tag": "dns-remote", "address": "https://dns.google/dns-query", "address_resolver": "dns-local", "detour": "direct"},
				{"tag": "dns-direct", "address": "77.88.8.8", "detour": "direct"},
				{"tag": "dns-block", "address": "rcode://success"},
				{"tag": "dns-local", "address": "local", "detour": "direct"}
			],
			"rules": [
				{"outbound": "any", "server": "dns-direct"},
				{"query_type": [32, 33], "server": "dns-block"}
			],
			"independent_cache": true
		},
		"inbounds": [{"tag": "mixed-in", "type": "mixed", "listen": "127.0.0.1", "listen_port": 21891}],
		"outbounds": [
			{"type": "direct", "tag": "direct"},
			{"type": "block", "tag": "block"}
		],
		"route": {
			"rules": [{"protocol": "dns", "action": "hijack-dns"}],
			"default_domain_resolver": {"server": "dns-direct"}
		}
	}`
	instance, cancel, err := Create([]byte(config))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	time.Sleep(700 * time.Millisecond)
	cancel()
	if err := instance.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
