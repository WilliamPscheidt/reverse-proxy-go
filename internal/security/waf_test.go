package security

import (
	"net/http"
	"net/url"
	"testing"
)

func TestWAF_SQLInjection(t *testing.T) {
	waf := NewWAF(DefaultWAFConfig())

	maliciousURIs := []string{
		"/users?id=1%20UNION%20SELECT%20username,%20password%20FROM%20users",
		"/login?user=admin'%20OR%20'1'='1",
		"/api/data?q=1;%20DROP%20TABLE%20clients;--",
	}

	for _, uri := range maliciousURIs {
		u, _ := url.Parse(uri)
		req := &http.Request{
			Method: "GET",
			URL:    u,
			Header: make(http.Header),
		}
		blocked, rule, _ := waf.Inspect(req, "1.2.3.4", "BR")
		if !blocked || rule != RuleSQLi {
			t.Fatalf("Expected SQLi block for URI %s, got blocked=%v, rule=%s", uri, blocked, rule)
		}
	}
}

func TestWAF_XSS(t *testing.T) {
	waf := NewWAF(DefaultWAFConfig())

	maliciousURIs := []string{
		"/search?q=<script>alert('pwned')</script>",
		"/profile?bio=<img%20src=x%20onerror=alert(1)>",
		"/redirect?url=javascript:alert(document.cookie)",
	}

	for _, uri := range maliciousURIs {
		u, _ := url.Parse(uri)
		req := &http.Request{
			Method: "GET",
			URL:    u,
			Header: make(http.Header),
		}
		blocked, rule, _ := waf.Inspect(req, "1.2.3.4", "BR")
		if !blocked || rule != RuleXSS {
			t.Fatalf("Expected XSS block for URI %s, got blocked=%v, rule=%s", uri, blocked, rule)
		}
	}
}

func TestWAF_PathTraversal(t *testing.T) {
	waf := NewWAF(DefaultWAFConfig())

	maliciousURIs := []string{
		"/static/../../etc/passwd",
		"/download?file=..%2F..%2Fwindows%2Fsystem32",
	}

	for _, uri := range maliciousURIs {
		u, _ := url.Parse(uri)
		req := &http.Request{
			Method: "GET",
			URL:    u,
			Header: make(http.Header),
		}
		blocked, rule, _ := waf.Inspect(req, "1.2.3.4", "BR")
		if !blocked || rule != RulePathTraversal {
			t.Fatalf("Expected Path Traversal block for URI %s, got blocked=%v, rule=%s", uri, blocked, rule)
		}
	}
}

func TestWAF_BadBots(t *testing.T) {
	waf := NewWAF(DefaultWAFConfig())

	u, _ := url.Parse("/api/status")
	req := &http.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header{
			"User-Agent": []string{"sqlmap/1.6#stable (https://sqlmap.org)"},
		},
	}

	blocked, rule, _ := waf.Inspect(req, "1.2.3.4", "BR")
	if !blocked || rule != RuleBadBot {
		t.Fatalf("Expected BadBot block, got blocked=%v, rule=%s", blocked, rule)
	}
}

func TestWAF_GeoBlocking(t *testing.T) {
	cfg := DefaultWAFConfig()
	cfg.BlockedCountries = []string{"CN", "RU"}
	waf := NewWAF(cfg)

	u, _ := url.Parse("/api/status")
	req := &http.Request{Method: "GET", URL: u, Header: make(http.Header)}

	// Allowed country
	blocked, _, _ := waf.Inspect(req, "1.2.3.4", "BR")
	if blocked {
		t.Fatal("Expected BR to be allowed")
	}

	// Blocked country
	blocked, rule, _ := waf.Inspect(req, "1.2.3.4", "RU")
	if !blocked || rule != RuleCountryBlock {
		t.Fatalf("Expected RU to be blocked, got blocked=%v, rule=%s", blocked, rule)
	}
}

func TestWAF_CIDRBlocking(t *testing.T) {
	cfg := DefaultWAFConfig()
	cfg.BlockedCIDRs = []string{"10.0.0.0/8", "192.168.100.0/24"}
	waf := NewWAF(cfg)

	u, _ := url.Parse("/index.html")
	req := &http.Request{Method: "GET", URL: u, Header: make(http.Header)}

	// Within blocked CIDR
	blocked, rule, _ := waf.Inspect(req, "10.254.1.2:54321", "US")
	if !blocked || rule != RuleCIDRBlock {
		t.Fatalf("Expected 10.x.x.x to be blocked by CIDR, got blocked=%v, rule=%s", blocked, rule)
	}

	// Outside CIDR
	blocked, _, _ = waf.Inspect(req, "172.16.0.5", "US")
	if blocked {
		t.Fatal("Expected 172.16.0.5 to be allowed")
	}
}
