package admin

import (
	"net/http"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/admin" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Reverse Proxy & Load Balancer | Console</title>
  
  <!-- FontAwesome 6 -->
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css" integrity="sha512-DTOQO9RWCH3ppGqcWaEA1BIZOC6xxalwEsw9c2QQeAIftl+Vegovlnee1c9QX4TctnWMn13TZye+giMm8e2LwA==" crossorigin="anonymous" referrerpolicy="no-referrer" />

  <style>
    :root {
      --bg: #090d16;
      --card-bg: #111827;
      --card-sub: #1f2937;
      --border: #1f2937;
      --border-focus: #374151;
      --accent: #2563eb;
      --accent-hover: #1d4ed8;
      --accent-light: rgba(37, 99, 235, 0.12);
      --green: #10b981;
      --green-bg: rgba(16, 185, 129, 0.1);
      --red: #ef4444;
      --red-bg: rgba(239, 68, 68, 0.1);
      --text: #f3f4f6;
      --text-muted: #9ca3af;
      --text-sub: #6b7280;
    }

    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background-color: var(--bg);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
      min-height: 100vh;
      line-height: 1.5;
      font-size: 14px;
    }

    .container {
      max-width: 1240px;
      margin: 0 auto;
      padding: 24px 20px;
    }

    /* Top Navigation */
    header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding-bottom: 20px;
      margin-bottom: 24px;
      border-bottom: 1px solid var(--border);
      flex-wrap: wrap;
      gap: 16px;
    }

    .brand-group {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .brand-icon {
      width: 36px;
      height: 36px;
      background: var(--accent-light);
      color: #60a5fa;
      border: 1px solid rgba(96, 165, 250, 0.2);
      border-radius: 8px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 16px;
    }

    .brand-title {
      font-size: 16px;
      font-weight: 600;
      color: #fff;
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .brand-subtitle {
      font-size: 12px;
      color: var(--text-muted);
    }

    .header-actions {
      display: flex;
      align-items: center;
      gap: 10px;
    }

    /* Status Pill */
    .status-pill {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      background: var(--card-sub);
      border: 1px solid var(--border);
      padding: 4px 10px;
      border-radius: 6px;
      font-size: 12px;
      color: var(--text-muted);
      font-family: ui-monospace, monospace;
    }

    .status-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: var(--green);
      display: inline-block;
      box-shadow: 0 0 8px rgba(16, 185, 129, 0.6);
    }

    /* Buttons */
    .btn {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      background: var(--accent);
      color: #fff;
      padding: 8px 14px;
      border-radius: 6px;
      font-weight: 500;
      font-size: 13px;
      border: 1px solid transparent;
      cursor: pointer;
      text-decoration: none;
      transition: all 0.15s ease-in-out;
    }

    .btn:hover {
      background: var(--accent-hover);
    }

    .btn-secondary {
      background: var(--card-bg);
      color: var(--text);
      border-color: var(--border);
    }

    .btn-secondary:hover {
      background: var(--card-sub);
      border-color: var(--border-focus);
    }

    .btn-danger {
      background: var(--red-bg);
      color: #f87171;
      border-color: rgba(239, 68, 68, 0.2);
    }

    .btn-danger:hover {
      background: var(--red);
      color: #fff;
    }

    .btn-sm {
      padding: 4px 8px;
      font-size: 12px;
      border-radius: 4px;
    }

    /* Stat Cards */
    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 16px;
      margin-bottom: 24px;
    }

    .stat-card {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 18px 20px;
      position: relative;
    }

    .stat-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 8px;
    }

    .stat-label {
      font-size: 12px;
      font-weight: 500;
      color: var(--text-muted);
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }

    .stat-icon {
      color: var(--text-sub);
      font-size: 14px;
    }

    .stat-value {
      font-size: 24px;
      font-weight: 600;
      color: #fff;
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    }

    .stat-sub {
      font-size: 12px;
      color: var(--text-muted);
      margin-top: 4px;
    }

    /* Sections */
    .section-box {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 8px;
      margin-bottom: 24px;
      overflow: hidden;
    }

    .section-header {
      padding: 16px 20px;
      border-bottom: 1px solid var(--border);
      display: flex;
      align-items: center;
      justify-content: space-between;
    }

    .section-title {
      font-size: 14px;
      font-weight: 600;
      color: #fff;
      display: flex;
      align-items: center;
      gap: 10px;
    }

    .section-body {
      padding: 20px;
    }

    /* Algorithm Cards */
    .algo-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
      gap: 14px;
    }

    .algo-card {
      background: var(--bg);
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 16px;
      cursor: pointer;
      transition: all 0.15s ease-in-out;
      position: relative;
    }

    .algo-card:hover {
      border-color: var(--border-focus);
    }

    .algo-card.active {
      border-color: #3b82f6;
      background: var(--accent-light);
    }

    .algo-title {
      font-size: 14px;
      font-weight: 600;
      color: #fff;
      margin-bottom: 6px;
      display: flex;
      align-items: center;
      justify-content: space-between;
    }

    .algo-desc {
      font-size: 12px;
      color: var(--text-muted);
      line-height: 1.4;
    }

    .algo-indicator {
      font-size: 12px;
      color: #60a5fa;
      display: none;
    }

    .algo-card.active .algo-indicator {
      display: inline-block;
    }

    /* Tables */
    .table-wrapper {
      overflow-x: auto;
    }

    table {
      width: 100%;
      border-collapse: collapse;
      text-align: left;
      font-size: 13px;
    }

    th {
      padding: 12px 16px;
      background: rgba(17, 24, 39, 0.6);
      color: var(--text-muted);
      font-weight: 500;
      border-bottom: 1px solid var(--border);
      text-transform: uppercase;
      font-size: 11px;
      letter-spacing: 0.5px;
    }

    td {
      padding: 14px 16px;
      border-bottom: 1px solid var(--border);
      vertical-align: middle;
    }

    tr:last-child td {
      border-bottom: none;
    }

    tr:hover td {
      background: rgba(255, 255, 255, 0.015);
    }

    .mono {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    }

    /* Badges */
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 2px 8px;
      border-radius: 4px;
      font-size: 11px;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }

    .badge-protocol {
      background: var(--card-sub);
      color: #93c5fd;
      border: 1px solid rgba(147, 197, 253, 0.2);
    }

    .badge-online {
      background: var(--green-bg);
      color: #34d399;
      border: 1px solid rgba(52, 211, 153, 0.2);
    }

    .badge-offline {
      background: var(--red-bg);
      color: #f87171;
      border: 1px solid rgba(248, 113, 113, 0.2);
    }

    /* Form */
    .form-grid {
      display: grid;
      grid-template-columns: 140px 1fr 100px auto;
      gap: 12px;
      align-items: center;
    }

    select, input {
      background: var(--bg);
      border: 1px solid var(--border);
      color: #fff;
      padding: 9px 12px;
      border-radius: 6px;
      font-size: 13px;
      width: 100%;
      outline: none;
      transition: border-color 0.15s;
    }

    select:focus, input:focus {
      border-color: #3b82f6;
    }

    /* Config Cards Grid */
    .config-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 16px;
    }

    .config-box {
      background: var(--bg);
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 16px;
      font-size: 13px;
    }

    .config-box-title {
      font-weight: 600;
      color: #fff;
      margin-bottom: 12px;
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 13px;
    }

    .config-row {
      display: flex;
      justify-content: space-between;
      padding: 6px 0;
      border-bottom: 1px solid rgba(255, 255, 255, 0.04);
      color: var(--text-muted);
    }

    .config-row:last-child {
      border-bottom: none;
    }

    .config-val {
      color: #fff;
      font-family: ui-monospace, monospace;
    }

    /* Toast */
    #toast {
      position: fixed;
      bottom: 24px;
      right: 24px;
      background: var(--card-bg);
      border: 1px solid var(--border-focus);
      color: #fff;
      padding: 12px 18px;
      border-radius: 6px;
      font-size: 13px;
      display: none;
      align-items: center;
      gap: 10px;
      box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
      z-index: 9999;
    }
  </style>
</head>
<body>

  <div class="container">
    <!-- Header -->
    <header>
      <div class="brand-group">
        <div class="brand-icon">
          <i class="fa-solid fa-server"></i>
        </div>
        <div>
          <div class="brand-title">
            Reverse Proxy & Load Balancer
            <span class="badge badge-protocol" style="font-size: 10px;">L4 / L7</span>
          </div>
          <div class="brand-subtitle">Console de Gerenciamento e Controle de Roteamento</div>
        </div>
      </div>

      <div class="header-actions">
        <div class="status-pill">
          <span class="status-dot"></span>
          <span id="uptimeBadge">Uptime: --</span>
        </div>
        <a href="http://localhost:2112/metrics" target="_blank" class="btn btn-secondary">
          <i class="fa-solid fa-chart-line"></i>
          Métricas
        </a>
        <button onclick="fetchStatus()" class="btn btn-secondary">
          <i class="fa-solid fa-arrows-rotate"></i>
          Atualizar
        </button>
      </div>
    </header>

    <!-- Metrics Cards -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-label">Nós L7 (HTTP)</span>
          <i class="fa-solid fa-globe stat-icon"></i>
        </div>
        <div class="stat-value" id="l7Nodes">-- / --</div>
        <div class="stat-sub" id="l7Reqs">0 requisições atendidas</div>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-label">Nós L4 (TCP)</span>
          <i class="fa-solid fa-network-wired stat-icon"></i>
        </div>
        <div class="stat-value" id="l4Nodes">-- / --</div>
        <div class="stat-sub" id="l4Reqs">0 conexões ativas</div>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-label">Algoritmo</span>
          <i class="fa-solid fa-sliders stat-icon"></i>
        </div>
        <div class="stat-value" id="currentAlgo" style="font-size: 20px; color: #60a5fa;">--</div>
        <div class="stat-sub">Roteamento em execução</div>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-label">Rate Limiter</span>
          <i class="fa-solid fa-shield-halved stat-icon"></i>
        </div>
        <div class="stat-value" id="rateLimitVal" style="font-size: 20px; color: #34d399;">--</div>
        <div class="stat-sub" id="rateLimitStrategy">Carregando...</div>
      </div>
    </div>

    <!-- Algorithm Selection -->
    <div class="section-box">
      <div class="section-header">
        <div class="section-title">
          <i class="fa-solid fa-scale-balanced" style="color: #60a5fa;"></i>
          Algoritmo de Balanceamento
        </div>
        <span style="font-size: 12px; color: var(--text-muted);">Hot-swap em tempo de execução</span>
      </div>
      <div class="section-body">
        <div class="algo-grid">
          <div class="algo-card" id="btn-round_robin" onclick="setAlgorithm('round_robin')">
            <div class="algo-title">
              Round-Robin
              <i class="fa-solid fa-check algo-indicator"></i>
            </div>
            <div class="algo-desc">Alternância sequencial atômica sem locks globais (~35ns/op). Distribuição equilibrada para clusters homogêneos.</div>
          </div>

          <div class="algo-card" id="btn-least_conn" onclick="setAlgorithm('least_conn')">
            <div class="algo-title">
              Least Connections
              <i class="fa-solid fa-check algo-indicator"></i>
            </div>
            <div class="algo-desc">Avalia em tempo real as conexões ativas em cada nó e direciona o fluxo para o servidor mais ocioso.</div>
          </div>

          <div class="algo-card" id="btn-ip_hash" onclick="setAlgorithm('ip_hash')">
            <div class="algo-title">
              IP Hash
              <i class="fa-solid fa-check algo-indicator"></i>
            </div>
            <div class="algo-desc">Hashing determinístico via xxhash do IP do cliente para persistência de sessão com failover dinâmico.</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Backends Table -->
    <div class="section-box">
      <div class="section-header">
        <div class="section-title">
          <i class="fa-solid fa-list-check" style="color: #60a5fa;"></i>
          Servidores Upstream Configurados
        </div>
        <span style="font-size: 12px; color: var(--text-muted);" id="nodesCount">Total: 0</span>
      </div>

      <div class="table-wrapper">
        <table>
          <thead>
            <tr>
              <th>Protocolo</th>
              <th>Endereço / URL Alvo</th>
              <th>Peso</th>
              <th>Status</th>
              <th>Conexões</th>
              <th>Total Reqs</th>
              <th>Falhas / Sucessos</th>
              <th style="text-align: right;">Ações</th>
            </tr>
          </thead>
          <tbody id="backendsTable">
            <tr>
              <td colspan="8" style="text-align: center; color: var(--text-muted); padding: 24px;">Carregando destinos...</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Add Backend Form -->
      <div style="padding: 16px 20px; background: rgba(17, 24, 39, 0.4); border-top: 1px solid var(--border);">
        <div style="font-size: 12px; font-weight: 600; text-transform: uppercase; color: var(--text-muted); margin-bottom: 10px; letter-spacing: 0.5px;">
          Adicionar Servidor Upstream
        </div>
        <form onsubmit="addBackend(event)" class="form-grid">
          <select id="newType" required>
            <option value="l7">HTTP (L7)</option>
            <option value="l4">TCP (L4)</option>
          </select>
          <input type="text" id="newTarget" placeholder="Ex: http://127.0.0.1:8084 ou 127.0.0.1:9084" required />
          <input type="number" id="newWeight" placeholder="Peso" value="1" min="1" max="100" required />
          <button type="submit" class="btn">
            <i class="fa-solid fa-plus"></i>
            Adicionar
          </button>
        </form>
      </div>
    </div>

    <!-- Health Check & HTTP Details -->
    <div class="section-box">
      <div class="section-header">
        <div class="section-title">
          <i class="fa-solid fa-sliders" style="color: #60a5fa;"></i>
          Configurações Operacionais
        </div>
        <span style="font-size: 12px; color: var(--text-muted);">Definições ativas de rede e observabilidade</span>
      </div>
      <div class="section-body">
        <div class="config-grid">
          <div class="config-box">
            <div class="config-box-title">
              <i class="fa-solid fa-heart-pulse" style="color: #ec4899;"></i>
              Active Health Check
            </div>
            <div class="config-row">
              <span>Intervalo de verificação</span>
              <span class="config-val" id="hcInterval">--</span>
            </div>
            <div class="config-row">
              <span>Timeout por ping</span>
              <span class="config-val" id="hcTimeout">--</span>
            </div>
            <div class="config-row">
              <span>Endpoint de inspeção</span>
              <span class="config-val" id="hcPath">--</span>
            </div>
            <div class="config-row">
              <span>Critérios de transição</span>
              <span class="config-val" id="hcThresholds">--</span>
            </div>
          </div>

          <div class="config-box">
            <div class="config-box-title">
              <i class="fa-solid fa-shield-halved" style="color: #3b82f6;"></i>
              Rate Limiting & Conexões
            </div>
            <div class="config-row">
              <span>Estratégia de Limite</span>
              <span class="config-val" id="rlStrategy">--</span>
            </div>
            <div class="config-row">
              <span>Status do Redis</span>
              <span class="config-val" id="redisStatus">--</span>
            </div>
            <div class="config-row">
              <span>Capacidade Burst</span>
              <span class="config-val" id="rlBurst">--</span>
            </div>
            <div class="config-row">
              <span>Buffer Pooling L4</span>
              <span class="config-val">32 KB (sync.Pool)</span>
            </div>
            <div style="margin-top: 12px; padding-top: 10px; border-top: 1px solid rgba(255, 255, 255, 0.05);">
              <div style="font-size: 11px; text-transform: uppercase; color: var(--text-muted); margin-bottom: 6px; font-weight: 500;">Ajustar Limite por IP</div>
              <div style="display: flex; gap: 8px; align-items: center;">
                <input type="number" id="inputRate" placeholder="Rate (req/s)" style="padding: 6px 8px; font-size: 12px; width: 90px;" value="5" min="1" />
                <input type="number" id="inputBurst" placeholder="Burst" style="padding: 6px 8px; font-size: 12px; width: 90px;" value="10" min="1" />
                <button onclick="updateRateLimit()" class="btn btn-sm" style="white-space: nowrap;"><i class="fa-solid fa-check"></i> Salvar</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <div id="toast">
    <i id="toastIcon" class="fa-solid fa-circle-check" style="color: var(--green);"></i>
    <span id="toastMsg">Mensagem</span>
  </div>

  <script>
    function showToast(msg, isError = false) {
      const t = document.getElementById('toast');
      const icon = document.getElementById('toastIcon');
      const text = document.getElementById('toastMsg');

      text.innerText = msg;
      if (isError) {
        icon.className = 'fa-solid fa-circle-exclamation';
        icon.style.color = 'var(--red)';
        t.style.borderColor = 'rgba(239, 68, 68, 0.4)';
      } else {
        icon.className = 'fa-solid fa-circle-check';
        icon.style.color = 'var(--green)';
        t.style.borderColor = 'rgba(16, 185, 129, 0.4)';
      }

      t.style.display = 'flex';
      setTimeout(() => { t.style.display = 'none'; }, 3000);
    }

    async function fetchStatus() {
      try {
        const res = await fetch('/api/status');
        if (!res.ok) return;
        const data = await res.json();

        document.getElementById('uptimeBadge').innerText = 'Uptime: ' + data.uptime;
        
        if (data.l7) {
          document.getElementById('l7Nodes').innerText = data.l7.healthy_nodes + ' / ' + data.l7.total_nodes;
          document.getElementById('l7Reqs').innerText = data.l7.total_requests.toLocaleString() + ' reqs (' + data.l7.active_conns + ' ativas)';
          document.getElementById('currentAlgo').innerText = data.l7.algorithm.toUpperCase().replace('_', '-');
          highlightAlgoBtn(data.l7.algorithm);
        }

        if (data.l4) {
          document.getElementById('l4Nodes').innerText = data.l4.healthy_nodes + ' / ' + data.l4.total_nodes;
          document.getElementById('l4Reqs').innerText = data.l4.active_conns + ' conexões ativas';
        }

        if (data.rate_limit) {
          document.getElementById('rateLimitVal').innerText = data.rate_limit.enabled 
            ? (data.rate_limit.rate.toLocaleString() + ' req/s') 
            : 'Desativado';

          let stratText = data.rate_limit.strategy ? data.rate_limit.strategy.toUpperCase() : 'MEMORY';
          if (data.rate_limit.strategy && data.rate_limit.strategy.includes('redis')) {
            if (data.rate_limit.healthy) {
              stratText = 'Redis (' + (data.rate_limit.redis_addr || 'Conectado') + ')';
              document.getElementById('redisStatus').innerHTML = '<span style="color: var(--green);"><i class="fa-solid fa-circle" style="font-size: 6px;"></i> Conectado</span>';
            } else {
              stratText = 'Redis [Fallback Local]';
              document.getElementById('redisStatus').innerHTML = '<span style="color: var(--yellow);"><i class="fa-solid fa-circle-exclamation" style="font-size: 9px;"></i> Offline (Fallback Ativo)</span>';
            }
          } else {
            document.getElementById('redisStatus').innerText = 'N/A (Modo In-Memory)';
          }

          document.getElementById('rateLimitStrategy').innerText = stratText;
          document.getElementById('rlStrategy').innerText = stratText;
          document.getElementById('rlBurst').innerText = (data.rate_limit.burst || 0).toLocaleString() + ' tokens';
        }

        if (data.health_check) {
          document.getElementById('hcInterval').innerText = (data.health_check.interval / 1e9) + 's';
          document.getElementById('hcTimeout').innerText = (data.health_check.timeout / 1e9) + 's';
          document.getElementById('hcPath').innerText = data.health_check.path;
          document.getElementById('hcThresholds').innerText = 'Falha: ' + data.health_check.unhealthy_threshold + 'x / Ok: ' + data.health_check.healthy_threshold + 'x';
        }

        await fetchBackends();
      } catch (err) {
        console.error('Fetch status error:', err);
      }
    }

    function highlightAlgoBtn(algo) {
      document.querySelectorAll('.algo-card').forEach(b => b.classList.remove('active'));
      const activeBtn = document.getElementById('btn-' + algo.toLowerCase());
      if (activeBtn) activeBtn.classList.add('active');
    }

    async function setAlgorithm(algo) {
      try {
        const res = await fetch('/api/algorithm', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ type: 'both', algorithm: algo })
        });
        if (res.ok) {
          showToast('Algoritmo atualizado: ' + algo.toUpperCase().replace('_', '-'));
          fetchStatus();
        } else {
          showToast('Falha ao atualizar algoritmo', true);
        }
      } catch (err) {
        showToast('Erro de conexão ao alterar algoritmo', true);
      }
    }

    async function fetchBackends() {
      try {
        const res = await fetch('/api/backends');
        if (!res.ok) return;
        const list = await res.json();
        const tbody = document.getElementById('backendsTable');
        document.getElementById('nodesCount').innerText = 'Total: ' + (list ? list.length : 0);

        if (!list || list.length === 0) {
          tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--text-muted); padding: 24px;">Nenhum destino configurado.</td></tr>';
          return;
        }

        let html = '';
        list.forEach(b => {
          const statusBadge = b.alive 
            ? '<span class="badge badge-online"><i class="fa-solid fa-circle" style="font-size: 6px;"></i> Online</span>' 
            : '<span class="badge badge-offline"><i class="fa-solid fa-circle" style="font-size: 6px;"></i> Offline</span>';

          const toggleAction = b.alive 
            ? '<button onclick="toggleBackend(\'' + b.type + '\', \'' + b.target + '\', false)" class="btn btn-secondary btn-sm" title="Drenar tráfego"><i class="fa-solid fa-pause"></i> Drenar</button>'
            : '<button onclick="toggleBackend(\'' + b.type + '\', \'' + b.target + '\', true)" class="btn btn-secondary btn-sm" title="Reativar"><i class="fa-solid fa-play"></i> Reativar</button>';

          html += '<tr>' +
            '<td><span class="badge badge-protocol">' + b.type.toUpperCase() + '</span></td>' +
            '<td><strong class="mono" style="color: #fff;">' + b.target + '</strong></td>' +
            '<td class="mono">' + b.weight + '</td>' +
            '<td>' + statusBadge + '</td>' +
            '<td class="mono">' + b.active_conns + '</td>' +
            '<td class="mono">' + b.total_requests.toLocaleString() + '</td>' +
            '<td class="mono" style="color: var(--text-muted);">' + b.consecutive_failures + ' err / ' + b.consecutive_successes + ' ok</td>' +
            '<td style="text-align: right; display: flex; gap: 6px; justify-content: flex-end;">' +
              toggleAction +
              '<button onclick="removeBackend(\'' + b.type + '\', \'' + b.target + '\')" class="btn btn-danger btn-sm" title="Excluir"><i class="fa-regular fa-trash-can"></i></button>' +
            '</td>' +
          '</tr>';
        });

        tbody.innerHTML = html;
      } catch (err) {
        console.error('Fetch backends error:', err);
      }
    }

    async function addBackend(e) {
      e.preventDefault();
      const type = document.getElementById('newType').value;
      const target = document.getElementById('newTarget').value.trim();
      const weight = parseInt(document.getElementById('newWeight').value) || 1;

      try {
        const res = await fetch('/api/backends', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ type, target, weight })
        });
        if (res.ok) {
          showToast('Servidor ' + target + ' adicionado com sucesso');
          document.getElementById('newTarget').value = '';
          fetchStatus();
        } else {
          const msg = await res.text();
          showToast('Erro ao adicionar: ' + msg, true);
        }
      } catch (err) {
        showToast('Erro de rede ao adicionar servidor', true);
      }
    }

    async function removeBackend(type, target) {
      if (!confirm('Deseja realmente remover o servidor ' + target + '?')) return;
      try {
        const res = await fetch('/api/backends', {
          method: 'DELETE',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ type, target })
        });
        if (res.ok) {
          showToast('Servidor removido do pool');
          fetchStatus();
        } else {
          showToast('Falha ao remover servidor', true);
        }
      } catch (err) {
        showToast('Erro de rede ao remover servidor', true);
      }
    }

    async function toggleBackend(type, target, alive) {
      try {
        const res = await fetch('/api/backends/toggle', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ type, target, alive })
        });
        if (res.ok) {
          showToast('Status do nó atualizado');
          fetchStatus();
        } else {
          showToast('Falha ao atualizar status', true);
        }
      } catch (err) {
        showToast('Erro ao atualizar status', true);
      }
    }

    async function updateRateLimit() {
      const rate = parseFloat(document.getElementById('inputRate').value);
      const burst = parseInt(document.getElementById('inputBurst').value);
      if (!rate || !burst) return;
      try {
        const res = await fetch('/api/ratelimit', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ rate, burst, enabled: true })
        });
        if (res.ok) {
          showToast('Limite atualizado: ' + rate + ' req/s (burst: ' + burst + ')');
          fetchStatus();
        } else {
          showToast('Erro ao atualizar rate limit', true);
        }
      } catch(err) {
        showToast('Erro de rede', true);
      }
    }

    // Polling regular a cada 2.5s
    fetchStatus();
    setInterval(fetchStatus, 2500);
  </script>
</body>
</html>
`
