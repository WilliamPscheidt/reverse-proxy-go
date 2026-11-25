package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

// SessionData holds in-memory authenticated session details.
type SessionData struct {
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Server) createSession(username string) string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)

	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	s.sessions[token] = SessionData{
		Username:  username,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	return token
}

func (s *Server) validateSession(token string) (SessionData, bool) {
	if token == "" {
		return SessionData{}, false
	}
	s.sessionsMu.RLock()
	data, exists := s.sessions[token]
	s.sessionsMu.RUnlock()

	if !exists {
		return SessionData{}, false
	}
	if time.Now().After(data.ExpiresAt) {
		s.sessionsMu.Lock()
		delete(s.sessions, token)
		s.sessionsMu.Unlock()
		return SessionData{}, false
	}
	return data, true
}

func (s *Server) deleteSession(token string) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	delete(s.sessions, token)
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	// If auth is completely disabled in config, redirect directly to dashboard
	if !s.cfg.Admin.AuthEnabled && s.cfg.Admin.Password == "" && s.cfg.Admin.APIKey == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// If already authenticated via session cookie, redirect directly to dashboard
	if cookie, err := r.Cookie("proxygo_session"); err == nil && cookie.Value != "" {
		if _, ok := s.validateSession(cookie.Value); ok {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(loginHTML))
}

type loginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleAPILogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var payload loginPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "Payload JSON inválido"})
		return
	}

	expectedUser := s.cfg.Admin.Username
	if expectedUser == "" {
		expectedUser = "admin"
	}
	expectedPass := s.cfg.Admin.Password
	if expectedPass == "" {
		expectedPass = "admin123"
	}

	if payload.Username != expectedUser || payload.Password != expectedPass {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "Usuário ou senha incorretos"})
		return
	}

	token := s.createSession(payload.Username)
	http.SetCookie(w, &http.Cookie{
		Name:     "proxygo_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 hours
	})

	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":  true,
		"username": payload.Username,
		"redirect": "/",
	})
}

func (s *Server) handleAPILogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("proxygo_session"); err == nil && cookie.Value != "" {
		s.deleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "proxygo_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":  true,
		"redirect": "/login",
	})
}

func (s *Server) handleAPIMe(w http.ResponseWriter, r *http.Request) {
	username := "admin"
	if cookie, err := r.Cookie("proxygo_session"); err == nil && cookie.Value != "" {
		if sess, ok := s.validateSession(cookie.Value); ok {
			username = sess.Username
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"authenticated": true,
		"username":      username,
	})
}

const loginHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Login | ProxyGO Enterprise Console</title>
  
  <!-- FontAwesome 6 -->
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css" integrity="sha512-DTOQO9RWCH3ppGqcWaEA1BIZOC6xxalwEsw9c2QQeAIftl+Vegovlnee1c9QX4TctnWMn13TZye+giMm8e2LwA==" crossorigin="anonymous" referrerpolicy="no-referrer" />

  <style>
    :root {
      --bg: #090d16;
      --card-bg: #111827;
      --card-sub: #1f2937;
      --border: #1f2937;
      --border-focus: #3b82f6;
      --accent: #2563eb;
      --accent-hover: #1d4ed8;
      --text: #f3f4f6;
      --text-muted: #9ca3af;
      --red: #ef4444;
      --green: #10b981;
    }

    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background-color: var(--bg);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 13px;
      line-height: 1.5;
      position: relative;
      overflow: hidden;
    }

    /* Ambient background glows */
    .glow-bg-1 {
      position: absolute;
      width: 450px;
      height: 450px;
      border-radius: 50%;
      background: radial-gradient(circle, rgba(37, 99, 235, 0.15) 0%, rgba(37, 99, 235, 0) 70%);
      top: -100px;
      left: 10%;
      pointer-events: none;
      filter: blur(40px);
    }
    .glow-bg-2 {
      position: absolute;
      width: 450px;
      height: 450px;
      border-radius: 50%;
      background: radial-gradient(circle, rgba(99, 102, 241, 0.12) 0%, rgba(99, 102, 241, 0) 70%);
      bottom: -100px;
      right: 10%;
      pointer-events: none;
      filter: blur(40px);
    }

    .login-container {
      width: 100%;
      max-width: 420px;
      padding: 24px;
      position: relative;
      z-index: 10;
    }

    .login-card {
      background: rgba(17, 24, 39, 0.85);
      border: 1px solid rgba(255, 255, 255, 0.08);
      backdrop-filter: blur(16px);
      border-radius: 12px;
      padding: 36px 32px;
      box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5), 0 0 1px 1px rgba(255, 255, 255, 0.05);
    }

    .brand-header {
      text-align: center;
      margin-bottom: 28px;
    }

    .brand-icon-box {
      width: 48px;
      height: 48px;
      background: linear-gradient(135deg, rgba(37, 99, 235, 0.25), rgba(99, 102, 241, 0.25));
      border: 1px solid rgba(96, 165, 250, 0.3);
      border-radius: 12px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      font-size: 22px;
      color: #60a5fa;
      margin-bottom: 12px;
      box-shadow: 0 0 20px rgba(37, 99, 235, 0.2);
    }

    .brand-title {
      font-size: 20px;
      font-weight: 700;
      color: #fff;
      letter-spacing: -0.3px;
    }

    .brand-subtitle {
      font-size: 12px;
      color: var(--text-muted);
      margin-top: 4px;
    }

    .alert-error {
      background: rgba(239, 68, 68, 0.12);
      border: 1px solid rgba(239, 68, 68, 0.3);
      color: #f87171;
      padding: 10px 14px;
      border-radius: 6px;
      font-size: 12px;
      margin-bottom: 20px;
      display: none;
      align-items: center;
      gap: 10px;
    }

    .form-group {
      margin-bottom: 18px;
    }

    .form-label {
      display: block;
      font-size: 11px;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.6px;
      color: var(--text-muted);
      margin-bottom: 8px;
    }

    .input-wrapper {
      position: relative;
      display: flex;
      align-items: center;
    }

    .input-icon {
      position: absolute;
      left: 14px;
      color: #6b7280;
      font-size: 13px;
    }

    .input-field {
      width: 100%;
      height: 42px;
      background: rgba(15, 23, 42, 0.6);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 0 40px 0 38px;
      color: #fff;
      font-size: 13px;
      transition: all 0.2s ease;
      font-family: inherit;
    }

    .input-field:focus {
      outline: none;
      border-color: var(--border-focus);
      background: rgba(15, 23, 42, 0.9);
      box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.15);
    }

    .toggle-pass {
      position: absolute;
      right: 14px;
      color: #6b7280;
      cursor: pointer;
      font-size: 13px;
      transition: color 0.15s;
    }
    .toggle-pass:hover { color: #fff; }

    .btn-submit {
      width: 100%;
      height: 42px;
      background: linear-gradient(135deg, #2563eb, #1d4ed8);
      border: none;
      border-radius: 8px;
      color: #fff;
      font-size: 13px;
      font-weight: 600;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      transition: all 0.2s ease;
      margin-top: 24px;
      box-shadow: 0 4px 12px rgba(37, 99, 235, 0.25);
    }

    .btn-submit:hover {
      background: linear-gradient(135deg, #1d4ed8, #1e40af);
      box-shadow: 0 6px 16px rgba(37, 99, 235, 0.35);
      transform: translateY(-1px);
    }

    .btn-submit:active {
      transform: translateY(0);
    }

    .footer-note {
      text-align: center;
      margin-top: 24px;
      font-size: 11px;
      color: #6b7280;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 6px;
    }
  </style>
</head>
<body>
  <div class="glow-bg-1"></div>
  <div class="glow-bg-2"></div>

  <div class="login-container">
    <div class="login-card">
      <div class="brand-header">
        <div class="brand-icon-box">
          <i class="fa-solid fa-server"></i>
        </div>
        <div class="brand-title">ProxyGO</div>
        <div class="brand-subtitle">Console de Controle Administrativo</div>
      </div>

      <div id="loginError" class="alert-error">
        <i class="fa-solid fa-circle-exclamation"></i>
        <span id="errorMsg">Usuário ou senha incorretos.</span>
      </div>

      <form id="loginForm" onsubmit="handleLoginSubmit(event)">
        <div class="form-group">
          <label class="form-label" for="username">Usuário Administrador</label>
          <div class="input-wrapper">
            <i class="fa-solid fa-user input-icon"></i>
            <input type="text" id="username" class="input-field" placeholder="Ex: admin" required autocomplete="username" autofocus />
          </div>
        </div>

        <div class="form-group">
          <label class="form-label" for="password">Senha de Acesso</label>
          <div class="input-wrapper">
            <i class="fa-solid fa-lock input-icon"></i>
            <input type="password" id="password" class="input-field" placeholder="••••••••" required autocomplete="current-password" />
            <i id="togglePass" class="fa-solid fa-eye toggle-pass" onclick="togglePasswordVisibility()"></i>
          </div>
        </div>

        <button type="submit" id="btnSubmit" class="btn-submit">
          <i class="fa-solid fa-arrow-right-to-bracket"></i>
          <span>Entrar no ProxyGO</span>
        </button>
      </form>

      <div class="footer-note">
        <i class="fa-solid fa-shield-halved" style="color: var(--green);"></i>
        <span>Sessão protegida por cookie HttpOnly & WAF</span>
      </div>
    </div>
  </div>

  <script>
    function togglePasswordVisibility() {
      const pass = document.getElementById('password');
      const icon = document.getElementById('togglePass');
      if (pass.type === 'password') {
        pass.type = 'text';
        icon.classList.remove('fa-eye');
        icon.classList.add('fa-eye-slash');
      } else {
        pass.type = 'password';
        icon.classList.remove('fa-eye-slash');
        icon.classList.add('fa-eye');
      }
    }

    async function handleLoginSubmit(e) {
      e.preventDefault();
      const user = document.getElementById('username').value.trim();
      const pass = document.getElementById('password').value;
      const errBox = document.getElementById('loginError');
      const errMsg = document.getElementById('errorMsg');
      const btn = document.getElementById('btnSubmit');

      errBox.style.display = 'none';
      btn.disabled = true;
      btn.innerHTML = '<i class="fa-solid fa-circle-notch fa-spin"></i> <span>Autenticando...</span>';

      try {
        const res = await fetch('/api/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username: user, password: pass })
        });

        const data = await res.json();
        if (res.ok && data.success) {
          btn.innerHTML = '<i class="fa-solid fa-check"></i> <span>Sucesso! Redirecionando...</span>';
          btn.style.background = '#10b981';
          setTimeout(() => {
            window.location.href = data.redirect || '/';
          }, 300);
        } else {
          errMsg.innerText = data.error || 'Credenciais inválidas. Verifique seu usuário e senha.';
          errBox.style.display = 'flex';
          btn.disabled = false;
          btn.innerHTML = '<i class="fa-solid fa-arrow-right-to-bracket"></i> <span>Entrar no ProxyGO</span>';
        }
      } catch (err) {
        errMsg.innerText = 'Erro ao conectar ao servidor. Verifique a rede.';
        errBox.style.display = 'flex';
        btn.disabled = false;
        btn.innerHTML = '<i class="fa-solid fa-arrow-right-to-bracket"></i> <span>Entrar no ProxyGO</span>';
      }
    }
  </script>
</body>
</html>
`
