package boxmain

import (
	"testing"
	"time"
)

// replicates the nekoray core path: Create with the config shapes nekoray generates
func TestNekoCreate(t *testing.T) {
	const config = `{
		"log": {"level": "warn"},
		"dns": {
			"servers": [
				{"tag": "dns-remote", "type": "https", "server": "dns.google", "domain_resolver": "dns-local"},
				{"tag": "dns-direct", "type": "udp", "server": "77.88.8.8"},
				{"tag": "dns-fake", "type": "fakeip", "inet4_range": "198.18.0.0/15", "inet6_range": "fc00::/18"},
				{"tag": "dns-local", "type": "local"}
			],
			"rules": [
				{"outbound": "any", "action": "route", "server": "dns-direct"},
				{"query_type": [32, 33], "action": "predefined", "rcode": 0},
				{"inbound": "tun-in", "action": "route", "server": "dns-fake"}
			],
			"strategy": "prefer_ipv4"
		},
		"inbounds": [{"tag": "mixed-in", "type": "mixed", "listen": "127.0.0.1", "listen_port": 21892}],
		"outbounds": [
			{"type": "direct", "tag": "direct"},
			{"type": "block", "tag": "block"}
		],
		"route": {
			"rules": [
				{"action": "sniff", "inbound": ["mixed-in"], "timeout": "3s"},
				{"protocol": "dns", "action": "hijack-dns"}
			],
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
