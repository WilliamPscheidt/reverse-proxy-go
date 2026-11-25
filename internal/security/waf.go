package security

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
)

// WAFRule constants identify the violated security policy.
const (
	RuleSQLi          = "SQL_INJECTION"
	RuleXSS           = "CROSS_SITE_SCRIPTING"
	RulePathTraversal = "PATH_TRAVERSAL"
	RuleBadBot        = "BAD_BOT"
	RuleCountryBlock  = "GEO_BLOCKED"
	RuleCIDRBlock     = "CIDR_BLOCKED"
)

// WAFConfig defines the rule toggles and policies.
type WAFConfig struct {
	Enabled            bool     `json:"enabled" yaml:"enabled"`
	BlockSQLi          bool     `json:"block_sqli" yaml:"block_sqli"`
	BlockXSS           bool     `json:"block_xss" yaml:"block_xss"`
	BlockPathTraversal bool     `json:"block_path_traversal" yaml:"block_path_traversal"`
	BlockBadBots       bool     `json:"block_bad_bots" yaml:"block_bad_bots"`
	BlockedCountries   []string `json:"blocked_countries" yaml:"blocked_countries"`
	AllowedCountries   []string `json:"allowed_countries" yaml:"allowed_countries"`
	BlockedCIDRs       []string `json:"blocked_cidrs" yaml:"blocked_cidrs"`
}

// WAFStats provides real-time counts of intercepted threats.
type WAFStats struct {
	TotalBlocked  uint64            `json:"total_blocked"`
	BlockedByRule map[string]uint64 `json:"blocked_by_rule"`
}

// WAF represents the Web Application Firewall engine.
type WAF struct {
	mu           sync.RWMutex
	config       WAFConfig
	sqliRegexes  []*regexp.Regexp
	xssRegexes   []*regexp.Regexp
	pathRegexes  []*regexp.Regexp
	botRegexes   []*regexp.Regexp
	blockedCIDRs []*net.IPNet
	totalBlocked uint64
	ruleBlocked  map[string]uint64
}

// DefaultWAFConfig returns sensible production-ready WAF settings.
func DefaultWAFConfig() WAFConfig {
	return WAFConfig{
		Enabled:            true,
		BlockSQLi:          true,
		BlockXSS:           true,
		BlockPathTraversal: true,
		BlockBadBots:       true,
		BlockedCountries:   []string{},
		AllowedCountries:   []string{},
		BlockedCIDRs:       []string{},
	}
}

// NewWAF creates and compiles a high-performance regex-based WAF engine.
func NewWAF(cfg WAFConfig) *WAF {
	w := &WAF{
		config:      cfg,
		ruleBlocked: make(map[string]uint64),
	}

	// SQL Injection patterns (designed to minimize false positives on normal text, JSON, passwords)
	w.sqliRegexes = []*regexp.Regexp{
		// 1. Classical keywords and statements
		regexp.MustCompile(`(?i)\b(UNION\s+(ALL\s+)?SELECT|INSERT\s+INTO\s+[\w.]+|DELETE\s+FROM\s+[\w.]+|DROP\s+(TABLE|DATABASE|VIEW)|ALTER\s+TABLE)\b`),
		// 2. Select subqueries and queries against sensitive databases/tables
		regexp.MustCompile(`(?i)(\(\s*SELECT\b[\s\S]*?\bFROM\b|'\s*;\s*SELECT\b|\bSELECT\b[\s\S]+?\bFROM\s+(users|accounts|passwords|members|admin|information_schema|sys|pg_|mysql)\b)`),
		// 3. String concatenation and tautologies: '||'1'='1, ' OR true, ' OR 1=1
		regexp.MustCompile(`(?i)('\s*\|\|\s*'|'\s*OR\s+(true|false)\b|'\s*AND\s+(true|false)\b)`),
		regexp.MustCompile(`(?i)(['"]\s*(OR|AND)\s+['"]?\d+['"]?\s*=\s*['"]?\d+|['"]\s*(OR|AND)\s+['"][^'"]*['"]\s*=\s*['"][^'"]*['"])`),
		regexp.MustCompile(`(?i)\b(OR|AND)\s+\d+\s*[><=]\s*\d+\b`),
		// 4. Time-based blind SQLi functions
		regexp.MustCompile(`(?i)\b(AND|OR)?\s*(SLEEP|BENCHMARK|PG_SLEEP|WAITFOR\s+DELAY)\s*\(\d+\)`),
		// 5. Injected comments (must follow quotes, SQL keywords or semicolons)
		regexp.MustCompile(`(?i)('|\bOR\b|\bAND\b|\bSELECT\b|;)\s*(--|#|/\*)`),
	}

	// Cross-Site Scripting (XSS) patterns
	w.xssRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(<script[\s>]|javascript:|alert\(|confirm\(|prompt\(|<iframe[\s>]|<svg[\s>])`),
		// Matches HTML tags with event handlers (e.g. <img onerror=...>, <details ontoggle=...) even without trailing '>'
		regexp.MustCompile(`(?i)<[a-z0-9_-]+[^>]*?\b(on\w+\s*=|src\s*=\s*['"]?javascript:)`),
	}

	// Path Traversal / LFI patterns
	w.pathRegexes = []*regexp.Regexp{
		// Standard traversal: ../, ..\, and Tomcat traversal ..;/
		regexp.MustCompile(`(?i)(\.\.[/\\]|\.\.;[/\\])`),
		// Common sensitive system files and directories
		regexp.MustCompile(`(?i)(/etc/(passwd|shadow|group|issue|hosts)|/proc/(self|\d+)/|/windows/system32|boot\.ini|win\.ini)`),
	}

	// Bad Bots, Scanners & Exploiters
	w.botRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(sqlmap|nikto|masscan|zgrab|nmap|dirbuster|gobuster|wpscan|hydra|acunetix|nessus)`),
	}

	w.compileCIDRs(cfg.BlockedCIDRs)

	return w
}

func (w *WAF) compileCIDRs(cidrs []string) {
	w.blockedCIDRs = nil
	for _, raw := range cidrs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(raw)
		if err == nil {
			w.blockedCIDRs = append(w.blockedCIDRs, ipNet)
		}
	}
}

// Inspect checks an incoming HTTP request for malicious patterns and security policy violations.
func (w *WAF) Inspect(r *http.Request, clientIP string, country string) (blocked bool, rule string, reason string) {
	w.mu.RLock()
	cfg := w.config
	blockedCIDRs := make([]*net.IPNet, len(w.blockedCIDRs))
	copy(blockedCIDRs, w.blockedCIDRs)
	w.mu.RUnlock()

	if !cfg.Enabled {
		return false, "", ""
	}

	// 1. CIDR Subnet Block
	if len(blockedCIDRs) > 0 {
		parsedIP := net.ParseIP(SanitizeIP(clientIP))
		if parsedIP != nil {
			for _, cidr := range blockedCIDRs {
				if cidr.Contains(parsedIP) {
					w.recordBlock(RuleCIDRBlock)
					return true, RuleCIDRBlock, fmt.Sprintf("IP %s pertence à sub-rede bloqueada %s", clientIP, cidr.String())
				}
			}
		}
	}

	// 2. Geo-Blocking (Country Rules)
	if country != "" {
		countryUpper := strings.ToUpper(country)

		// Allowed countries whitelist (if configured, strictly block anything outside)
		if len(cfg.AllowedCountries) > 0 {
			allowed := false
			for _, c := range cfg.AllowedCountries {
				if strings.ToUpper(c) == countryUpper {
					allowed = true
					break
				}
			}
			if !allowed {
				w.recordBlock(RuleCountryBlock)
				return true, RuleCountryBlock, fmt.Sprintf("País %s não está na lista de países permitidos", countryUpper)
			}
		}

		// Blocked countries blacklist
		for _, c := range cfg.BlockedCountries {
			if strings.ToUpper(c) == countryUpper {
				w.recordBlock(RuleCountryBlock)
				return true, RuleCountryBlock, fmt.Sprintf("Acesso a partir do país %s foi bloqueado pelo administrador", countryUpper)
			}
		}
	}

	// 3. Bad Bots / User-Agent Scanner Detection
	if cfg.BlockBadBots {
		ua := r.UserAgent()
		for _, re := range w.botRegexes {
			if re.MatchString(ua) {
				w.recordBlock(RuleBadBot)
				return true, RuleBadBot, fmt.Sprintf("Ferramenta de scan / Bot não autorizado detectado no User-Agent: %s", ua)
			}
		}
	}

	// Collect string payloads to inspect (URI, query params, path, headers, body)
	// Apply recursive URL decoding (up to 3 levels) to defeat nested obfuscation (%252e%252e%252f -> ../)
	var testStrings []string
	addWithDecodings := func(raw string) {
		if raw == "" {
			return
		}
		testStrings = append(testStrings, raw)
		curr := raw
		for i := 0; i < 3; i++ {
			dec, err := url.QueryUnescape(curr)
			if err != nil || dec == curr {
				break
			}
			testStrings = append(testStrings, dec)
			curr = dec
		}
	}

	rawURI := r.URL.RequestURI()
	addWithDecodings(rawURI)
	addWithDecodings(r.URL.Path)
	addWithDecodings(r.URL.RawQuery)

	// Inspect HTTP Headers (Cookie, Referer, Origin, User-Agent, etc.)
	for _, h := range []string{"Cookie", "Referer", "Origin", "User-Agent"} {
		if val := r.Header.Get(h); val != "" {
			addWithDecodings(val)
		}
	}

	// 4. Request Body Inspection (Inspect all methods with a body, up to 512KB)
	if r.Body != nil {
		bodySample, err := io.ReadAll(io.LimitReader(r.Body, 512*1024))
		if err == nil && len(bodySample) > 0 {
			// Reconstitute original body: the buffered sample + the remainder of r.Body
			r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(bodySample), r.Body))
			addWithDecodings(string(bodySample))
		}
	}

	// 4. Path Traversal & LFI
	if cfg.BlockPathTraversal {
		for _, s := range testStrings {
			for _, re := range w.pathRegexes {
				if re.MatchString(s) {
					w.recordBlock(RulePathTraversal)
					return true, RulePathTraversal, "Tentativa de Path Traversal / LFI detectada"
				}
			}
		}
	}

	// 5. SQL Injection
	if cfg.BlockSQLi {
		for _, s := range testStrings {
			for _, re := range w.sqliRegexes {
				if re.MatchString(s) {
					w.recordBlock(RuleSQLi)
					return true, RuleSQLi, "Assinatura de ataque SQL Injection detectada"
				}
			}
		}
	}

	// 6. Cross-Site Scripting (XSS)
	if cfg.BlockXSS {
		for _, s := range testStrings {
			for _, re := range w.xssRegexes {
				if re.MatchString(s) {
					w.recordBlock(RuleXSS)
					return true, RuleXSS, "Tentativa de Cross-Site Scripting (XSS) detectada"
				}
			}
		}
	}

	return false, "", ""
}

func (w *WAF) recordBlock(rule string) {
	atomic.AddUint64(&w.totalBlocked, 1)
	w.mu.Lock()
	w.ruleBlocked[rule]++
	w.mu.Unlock()
}

// GetStats returns current aggregate stats.
func (w *WAF) GetStats() WAFStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	byRule := make(map[string]uint64, len(w.ruleBlocked))
	for k, v := range w.ruleBlocked {
		byRule[k] = v
	}

	return WAFStats{
		TotalBlocked:  atomic.LoadUint64(&w.totalBlocked),
		BlockedByRule: byRule,
	}
}

// GetConfig returns the active configuration.
func (w *WAF) GetConfig() WAFConfig {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.config
}

// UpdateConfig dynamically updates the WAF settings in memory.
func (w *WAF) UpdateConfig(cfg WAFConfig) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.config = cfg
	w.compileCIDRs(cfg.BlockedCIDRs)
}
