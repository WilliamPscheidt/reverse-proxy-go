package security_test

// Bateria de ataques e falsos positivos contra o WAF.
// Roda só sob demanda, para não quebrar o `go test ./...` normal:
//
//	SECURITY_AUDIT=1 go test ./internal/security -run TestWAFAttackSuite -v
//
// Cada bypass ou falso positivo vira uma falha do teste. Depois de corrigir o WAF,
// rode de novo para conferir o que ainda passa.

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/security"
)

type wafCase struct {
	name      string
	method    string
	target    string
	body      string
	headers   map[string]string
	malicious bool
}

func TestWAFAttackSuite(t *testing.T) {
	if os.Getenv("SECURITY_AUDIT") != "1" {
		t.Skip("defina SECURITY_AUDIT=1 para rodar a bateria de ataques")
	}
	w := security.NewWAF(security.DefaultWAFConfig())
	big := strings.Repeat("a", 128*1024)

	cases := []wafCase{
		// Ataques (devem ser bloqueados)
		{"SQLi UNION SELECT na query", "GET", "/p?id=1%20UNION%20SELECT%20pass%20FROM%20users", "", nil, true},
		{"SQLi com ||", "GET", "/p?id=1'||'1'='1", "", nil, true},
		{"SQLi OR true", "GET", "/p?id=1'%20OR%20true", "", nil, true},
		{"SQLi OR 2>1", "GET", "/p?id=1%20OR%202>1", "", nil, true},
		{"SQLi SLEEP()", "GET", "/p?id=1%20AND%20SLEEP(5)", "", nil, true},
		{"SQLi no Cookie", "GET", "/p", "", map[string]string{"Cookie": "id=1 UNION SELECT pass FROM users"}, true},
		{"XSS <script>", "GET", "/s?q=%3Cscript%3Ealert(1)%3C/script%3E", "", nil, true},
		{"XSS <img onerror>", "GET", "/s?q=%3Cimg%20src=x%20onerror=alert(1)%3E", "", nil, true},
		{"XSS <details ontoggle> sem fechar", "GET", "/s?q=%3Cdetails%20open%20ontoggle=confirm(1)", "", nil, true},
		{"XSS no Referer", "GET", "/", "", map[string]string{"Referer": "<script>confirm(1)</script>"}, true},
		{"XSS depois de 128KB de body", "POST", "/api", "x=" + big + "<script>confirm(1)</script>", nil, true},
		{"XSS em GET com body", "GET", "/api", "<script>confirm(1)</script>", nil, true},
		{"Traversal ..%2f", "GET", "/f?p=..%2f..%2fetc%2fpasswd", "", nil, true},
		{"Traversal com encode duplo", "GET", "/f?p=%252e%252e%252f%252e%252e%252fetc%252fshadow", "", nil, true},
		{"Traversal ..;/ (Tomcat)", "GET", "/app/..;/manager/html", "", nil, true},
		{"LFI /etc/shadow", "GET", "/f?p=/etc/shadow", "", nil, true},
		{"LFI /proc/self/environ", "GET", "/f?p=/proc/self/environ", "", nil, true},
		{"Bot sqlmap", "GET", "/", "", map[string]string{"User-Agent": "sqlmap/1.7"}, true},

		// Tráfego legítimo (deve passar)
		{"Legítimo: senha com #", "POST", "/login", "user=ana&pass=Minha#Senha1", nil, false},
		{"Legítimo: texto com --", "POST", "/comment", "msg=Ok -- combinado", nil, false},
		{"Legítimo: cor CSS em JSON", "POST", "/theme", `{"color":"#ff0000"}`, map[string]string{"Content-Type": "application/json"}, false},
		{"Legítimo: busca 'select items from menu'", "GET", "/search?q=select%20items%20from%20menu", "", nil, false},
		{"Legítimo: 'this or that=fine'", "POST", "/comment", "msg=either this or that=fine", nil, false},
		{"Legítimo: comentário /* */", "POST", "/doc", "code=/* comentario */", nil, false},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			var r *http.Request
			if c.body != "" {
				r = httptest.NewRequest(c.method, c.target, strings.NewReader(c.body))
			} else {
				r = httptest.NewRequest(c.method, c.target, nil)
			}
			for k, v := range c.headers {
				r.Header.Set(k, v)
			}
			blocked, rule, _ := w.Inspect(r, "8.8.8.8:1234", "US")
			if c.malicious && !blocked {
				t.Errorf("BYPASS: ataque passou pelo WAF")
			}
			if !c.malicious && blocked {
				t.Errorf("FALSO POSITIVO: requisição legítima bloqueada (%s)", rule)
			}
		})
	}
}
