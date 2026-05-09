package scriptengine

import (
	"context"
	"testing"
)

func TestRunPostFetch(t *testing.T) {
	script := `function main(config) {
		config.dns = { enable: true };
		config.proxies = config.proxies.filter(function(p) { return p.name !== 'bad'; });
		return config;
	}`

	config := map[string]interface{}{
		"proxies": []interface{}{
			map[string]interface{}{"name": "good", "type": "ss"},
			map[string]interface{}{"name": "bad", "type": "ss"},
		},
	}

	result, err := RunPostFetch(context.Background(), script, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dns, ok := result["dns"].(map[string]interface{})
	if !ok {
		t.Fatal("dns should be a map")
	}
	if dns["enable"] != true {
		t.Errorf("dns.enable = %v, want true", dns["enable"])
	}

	proxies, ok := result["proxies"].([]interface{})
	if !ok {
		t.Fatalf("proxies should be an array, got %T", result["proxies"])
	}
	if len(proxies) != 1 {
		t.Errorf("len(proxies) = %d, want 1", len(proxies))
	}
}

func TestRunPreSaveNodes(t *testing.T) {
	script := `function main(proxies) {
		return proxies.map(function(p) {
			p.name = 'prefix_' + p.name;
			p['skip-cert-verify'] = 'true';
			return p;
		});
	}`

	proxies := []map[string]interface{}{
		{"name": "node1", "type": "ss", "server": "1.1.1.1"},
		{"name": "node2", "type": "vmess", "server": "2.2.2.2"},
	}

	result, err := RunPreSaveNodes(context.Background(), script, proxies)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if result[0]["name"] != "prefix_node1" {
		t.Errorf("result[0].name = %v, want prefix_node1", result[0]["name"])
	}
	if result[1]["name"] != "prefix_node2" {
		t.Errorf("result[1].name = %v, want prefix_node2", result[1]["name"])
	}
	if result[0]["skip-cert-verify"] != "true" {
		t.Errorf("result[0].skip-cert-verify = %v, want true", result[0]["skip-cert-verify"])
	}
}

func TestRunPostFetch_Timeout(t *testing.T) {
	script := `function main(config) {
		while(true) {}
		return config;
	}`

	config := map[string]interface{}{}
	_, err := RunPostFetch(context.Background(), script, config)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestRunPostFetch_SyntaxError(t *testing.T) {
	script := `function main(config { return config; }`

	config := map[string]interface{}{}
	_, err := RunPostFetch(context.Background(), script, config)
	if err == nil {
		t.Fatal("expected syntax error")
	}
}
