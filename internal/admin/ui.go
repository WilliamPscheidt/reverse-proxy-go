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
  <title>Reverse Proxy & Load Balancer | Enterprise Console</title>
  
  <!-- FontAwesome 6 -->
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css" integrity="sha512-DTOQO9RWCH3ppGqcWaEA1BIZOC6xxalwEsw9c2QQeAIftl+Vegovlnee1c9QX4TctnWMn13TZye+giMm8e2LwA==" crossorigin="anonymous" referrerpolicy="no-referrer" />

  <!-- Leaflet Maps CSS -->
  <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" integrity="sha256-p4NxAoJBhIIN+hmNHrzRCf9tD/miZyoHS5obTRR9BMY=" crossorigin="" />

  <style>
    :root {
      --bg: #090d16;
      --sidebar-bg: #0d121f;
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
      --yellow: #f59e0b;
      --yellow-bg: rgba(245, 158, 11, 0.1);
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
      display: flex;
      font-size: 13px;
      line-height: 1.5;
    }

    /* Layout: Sidebar + Main Content */
    .sidebar {
      width: 250px;
      background: var(--sidebar-bg);
      border-right: 1px solid var(--border);
      display: flex;
      flex-direction: column;
      position: fixed;
      top: 0;
      bottom: 0;
      left: 0;
      z-index: 100;
    }

    .main-wrapper {
      margin-left: 250px;
      flex: 1;
      display: flex;
      flex-direction: column;
      min-height: 100vh;
    }

    /* Sidebar Brand */
    .sidebar-brand {
      padding: 20px 18px;
      border-bottom: 1px solid var(--border);
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .brand-icon {
      width: 34px;
      height: 34px;
      background: var(--accent-light);
      color: #60a5fa;
      border: 1px solid rgba(96, 165, 250, 0.2);
      border-radius: 8px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 16px;
    }

    .brand-name {
      font-size: 15px;
      font-weight: 600;
      color: #fff;
      letter-spacing: -0.2px;
    }

    .brand-mode {
      font-size: 11px;
      color: var(--text-muted);
    }

    /* Sidebar Navigation */
    .sidebar-menu {
      padding: 16px 10px;
      flex: 1;
      overflow-y: auto;
    }

    .nav-label {
      font-size: 10px;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.8px;
      color: var(--text-sub);
      padding: 8px 12px 4px;
      margin-top: 10px;
    }

    .nav-label:first-child { margin-top: 0; }

    .nav-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 10px 14px;
      color: var(--text-muted);
      border-radius: 6px;
      cursor: pointer;
      font-size: 13px;
      font-weight: 500;
      transition: all 0.15s ease-in-out;
      margin-bottom: 2px;
      text-decoration: none;
    }

    .nav-item i {
      width: 16px;
      text-align: center;
      font-size: 14px;
    }

    .nav-item:hover {
      color: #fff;
      background: rgba(255, 255, 255, 0.04);
    }

    .nav-item.active {
      color: #fff;
      background: var(--accent);
    }

    .sidebar-footer {
      padding: 16px;
      border-top: 1px solid var(--border);
      font-size: 11px;
      color: var(--text-muted);
      background: rgba(0, 0, 0, 0.15);
    }

    /* Topbar */
    .topbar {
      height: 60px;
      border-bottom: 1px solid var(--border);
      padding: 0 28px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      background: rgba(9, 13, 22, 0.8);
      backdrop-filter: blur(8px);
      position: sticky;
      top: 0;
      z-index: 90;
    }

    .page-title {
      font-size: 16px;
      font-weight: 600;
      color: #fff;
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .topbar-actions {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    /* Content Area */
    .content-area {
      padding: 24px 28px;
      flex: 1;
    }

    .tab-content {
      display: none;
    }

    .tab-content.active {
      display: block;
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

    .btn:hover { background: var(--accent-hover); }

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

    /* Cards */
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
      font-size: 11px;
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
      font-size: 26px;
      font-weight: 600;
      color: #fff;
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    }

    .stat-sub {
      font-size: 12px;
      color: var(--text-muted);
      margin-top: 4px;
    }

    /* Section Boxes */
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

    /* Map Box */
    #worldMap {
      height: 320px;
      width: 100%;
      border-radius: 6px;
      background: #0d121f;
    }

    /* Tables */
    .table-wrapper { overflow-x: auto; }
    table { width: 100%; border-collapse: collapse; text-align: left; font-size: 13px; }
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
    td { padding: 14px 16px; border-bottom: 1px solid var(--border); vertical-align: middle; }
    tr:last-child td { border-bottom: none; }
    tr:hover td { background: rgba(255, 255, 255, 0.015); }

    .mono { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }

    /* Badges */
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 5px;
      padding: 2px 8px;
      border-radius: 4px;
      font-size: 11px;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }
    .badge-protocol { background: var(--card-sub); color: #93c5fd; border: 1px solid rgba(147, 197, 253, 0.2); }
    .badge-online { background: var(--green-bg); color: #34d399; border: 1px solid rgba(52, 211, 153, 0.2); }
    .badge-offline { background: var(--red-bg); color: #f87171; border: 1px solid rgba(248, 113, 113, 0.2); }
    .badge-warn { background: var(--yellow-bg); color: #fbbf24; border: 1px solid rgba(251, 191, 36, 0.2); }

    /* Forms */
    .form-grid {
      display: grid;
      grid-template-columns: 140px 1fr 100px auto;
      gap: 12px;
      align-items: center;
    }

    .settings-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
      gap: 20px;
    }

    .form-group {
      margin-bottom: 14px;
    }

    .form-group label {
      display: block;
      font-size: 12px;
      font-weight: 500;
      color: var(--text-muted);
      margin-bottom: 6px;
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

    select:focus, input:focus { border-color: #3b82f6; }

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
    }

    .algo-card:hover { border-color: var(--border-focus); }
    .algo-card.active { border-color: #3b82f6; background: var(--accent-light); }

    .algo-title {
      font-size: 14px;
      font-weight: 600;
      color: #fff;
      margin-bottom: 6px;
      display: flex;
      align-items: center;
      justify-content: space-between;
    }

    .algo-desc { font-size: 12px; color: var(--text-muted); line-height: 1.4; }
    .algo-indicator { font-size: 12px; color: #60a5fa; display: none; }
    .algo-card.active .algo-indicator { display: inline-block; }

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

  <!-- Left Sidebar -->
  <aside class="sidebar">
    <div class="sidebar-brand">
      <div class="brand-icon">
        <i class="fa-solid fa-server"></i>
      </div>
      <div>
        <div class="brand-name">Reverse Proxy</div>
        <div class="brand-mode" id="sidebarMode">L4 / L7 High-Perf</div>
      </div>
    </div>

    <div class="sidebar-menu">
      <div class="nav-label">Navegação</div>
      <a class="nav-item active" onclick="switchTab('overview')">
        <i class="fa-solid fa-chart-pie"></i>
        <span>Overview & Tráfego</span>
      </a>
      <a class="nav-item" onclick="switchTab('backends')">
        <i class="fa-solid fa-network-wired"></i>
        <span>Upstream Backends</span>
      </a>
      <a class="nav-item" onclick="switchTab('algorithms')">
        <i class="fa-solid fa-scale-balanced"></i>
        <span>Balanceamento</span>
      </a>
      <a class="nav-item" onclick="switchTab('ratelimit')">
        <i class="fa-solid fa-shield-halved"></i>
        <span>Rate Limit & Redis</span>
      </a>

      <div class="nav-label">Sistema</div>
      <a class="nav-item" onclick="switchTab('settings')">
        <i class="fa-solid fa-sliders"></i>
        <span>Configurações Gerais</span>
      </a>
      <a href="http://localhost:2112/metrics" target="_blank" class="nav-item">
        <i class="fa-solid fa-chart-line"></i>
        <span>Prometheus Metrics</span>
      </a>
    </div>

    <div class="sidebar-footer">
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px;">
        <span style="font-weight: 600; color: #fff;">Status</span>
        <span class="status-dot"></span>
      </div>
      <div>Porta HTTP: <span class="mono" style="color: #fff;">:8080</span></div>
      <div>Porta TCP: <span class="mono" style="color: #fff;">:9000</span></div>
    </div>
  </aside>

  <!-- Main Content Wrapper -->
  <div class="main-wrapper">
    <!-- Topbar -->
    <header class="topbar">
      <div class="page-title" id="pageTitle">
        <i class="fa-solid fa-chart-pie" style="color: #60a5fa;"></i>
        <span>Visão Geral do Tráfego</span>
      </div>

      <div class="topbar-actions">
        <div class="status-pill">
          <span class="status-dot"></span>
          <span id="uptimeBadge">Uptime: --</span>
        </div>
        <button onclick="fetchStatus()" class="btn btn-secondary">
          <i class="fa-solid fa-arrows-rotate"></i>
          Atualizar
        </button>
      </div>
    </header>

    <!-- Content Area with Tabs -->
    <main class="content-area">

      <!-- ============================================================== -->
      <!-- TAB 1: OVERVIEW & TRAFFIC (HOMEPAGE) -->
      <!-- ============================================================== -->
      <section id="tab-overview" class="tab-content active">
        <!-- Requests Time Window Cards -->
        <div class="stats-grid">
          <div class="stat-card">
            <div class="stat-header">
              <span class="stat-label">Último Minuto</span>
              <i class="fa-solid fa-stopwatch stat-icon"></i>
            </div>
            <div class="stat-value" id="reqLastMin">0</div>
            <div class="stat-sub">Requisições nos últimos 60s</div>
          </div>

          <div class="stat-card">
            <div class="stat-header">
              <span class="stat-label">Última Hora</span>
              <i class="fa-solid fa-clock stat-icon"></i>
            </div>
            <div class="stat-value" id="reqLastHour">0</div>
            <div class="stat-sub">Requisições nos últimos 60min</div>
          </div>

          <div class="stat-card">
            <div class="stat-header">
              <span class="stat-label">Últimas 24 Horas</span>
              <i class="fa-solid fa-calendar-day stat-icon"></i>
            </div>
            <div class="stat-value" id="reqLastDay">0</div>
            <div class="stat-sub">Volume diário acumulado</div>
          </div>

          <div class="stat-card">
            <div class="stat-header">
              <span class="stat-label">Bloqueios Rate Limit</span>
              <i class="fa-solid fa-ban stat-icon" style="color: #f87171;"></i>
            </div>
            <div class="stat-value" id="totalBlocked" style="color: #f87171;">0</div>
            <div class="stat-sub">Spam / HTTP 429 mitigados</div>
          </div>
        </div>

        <!-- Geographic Map Section -->
        <div class="section-box">
          <div class="section-header">
            <div class="section-title">
              <i class="fa-solid fa-earth-americas" style="color: #60a5fa;"></i>
              Distribuição Geográfica dos Requests
            </div>
            <span style="font-size: 12px; color: var(--text-muted);" id="geoSummary">5 regiões ativas</span>
          </div>
          <div class="section-body">
            <div id="worldMap"></div>
          </div>
        </div>

        <!-- Top IPs & Live Nodes Summary -->
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px;">
          <!-- Top IPs -->
          <div class="section-box" style="margin-bottom: 0;">
            <div class="section-header">
              <div class="section-title">
                <i class="fa-solid fa-users-viewfinder" style="color: #60a5fa;"></i>
                Principais IPs de Origem
              </div>
            </div>
            <div class="table-wrapper">
              <table>
                <thead>
                  <tr>
                    <th>Endereço IP</th>
                    <th>Volume</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody id="topIPsTable">
                  <tr><td colspan="3" style="text-align: center; color: var(--text-muted);">Aguardando tráfego...</td></tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- HTTP Status Codes Summary -->
          <div class="section-box" style="margin-bottom: 0;">
            <div class="section-header">
              <div class="section-title">
                <i class="fa-solid fa-chart-simple" style="color: #60a5fa;"></i>
                Distribuição de Status HTTP
              </div>
            </div>
            <div class="section-body">
              <div style="display: flex; flex-direction: column; gap: 12px;">
                <div>
                  <div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
                    <span>HTTP 2xx (Sucesso)</span>
                    <strong class="mono" id="stat2xx" style="color: #34d399;">0</strong>
                  </div>
                  <div style="height: 6px; background: var(--card-sub); border-radius: 3px; overflow: hidden;">
                    <div id="bar2xx" style="height: 100%; width: 0%; background: #34d399; transition: width 0.3s;"></div>
                  </div>
                </div>

                <div>
                  <div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
                    <span>HTTP 429 (Rate Limit Bloqueado)</span>
                    <strong class="mono" id="stat429" style="color: #f87171;">0</strong>
                  </div>
                  <div style="height: 6px; background: var(--card-sub); border-radius: 3px; overflow: hidden;">
                    <div id="bar429" style="height: 100%; width: 0%; background: #f87171; transition: width 0.3s;"></div>
                  </div>
                </div>

                <div>
                  <div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
                    <span>HTTP 5xx (Erros de Gateway)</span>
                    <strong class="mono" id="stat5xx" style="color: #fbbf24;">0</strong>
                  </div>
                  <div style="height: 6px; background: var(--card-sub); border-radius: 3px; overflow: hidden;">
                    <div id="bar5xx" style="height: 100%; width: 0%; background: #fbbf24; transition: width 0.3s;"></div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- ============================================================== -->
      <!-- TAB 2: UPSTREAM BACKENDS -->
      <!-- ============================================================== -->
      <section id="tab-backends" class="tab-content">
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
                <tr><td colspan="8" style="text-align: center; color: var(--text-muted); padding: 24px;">Carregando destinos...</td></tr>
              </tbody>
            </table>
          </div>

          <!-- Add Backend Form -->
          <div style="padding: 18px 20px; background: rgba(17, 24, 39, 0.4); border-top: 1px solid var(--border);">
            <div style="font-size: 12px; font-weight: 600; text-transform: uppercase; color: var(--text-muted); margin-bottom: 12px; letter-spacing: 0.5px;">
              Adicionar Novo Servidor Upstream
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
                Adicionar Servidor
              </button>
            </form>
          </div>
        </div>
      </section>

      <!-- ============================================================== -->
      <!-- TAB 3: LOAD BALANCING ALGORITHMS -->
      <!-- ============================================================== -->
      <section id="tab-algorithms" class="tab-content">
        <div class="section-box">
          <div class="section-header">
            <div class="section-title">
              <i class="fa-solid fa-scale-balanced" style="color: #60a5fa;"></i>
              Seleção do Algoritmo de Balanceamento
            </div>
            <span style="font-size: 12px; color: var(--text-muted);">Hot-swap em tempo de execução</span>
          </div>
          <div class="section-body">
            <div class="algo-grid">
              <div class="algo-card" id="btn-round_robin" onclick="setAlgorithm('round_robin')">
                <div class="algo-title">
                  Round-Robin (Atômico)
                  <i class="fa-solid fa-check algo-indicator"></i>
                </div>
                <div class="algo-desc">Alternância sequencial lock-free (~35ns/op). Ideal para nós de capacidade homogênea com distribuição uniforme.</div>
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
                  IP Hash (Sticky Session)
                  <i class="fa-solid fa-check algo-indicator"></i>
                </div>
                <div class="algo-desc">Hashing determinístico via xxhash do IP do cliente para persistência de sessão com failover dinâmico.</div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- ============================================================== -->
      <!-- TAB 4: RATE LIMIT & REDIS -->
      <!-- ============================================================== -->
      <section id="tab-ratelimit" class="tab-content">
        <div class="settings-grid">
          <div class="section-box">
            <div class="section-header">
              <div class="section-title">
                <i class="fa-solid fa-database" style="color: #ec4899;"></i>
                Status da Conexão Redis
              </div>
            </div>
            <div class="section-body">
              <div class="form-group">
                <label>Estratégia Ativa</label>
                <div class="mono" id="rlStrategy" style="font-size: 14px; color: #fff; font-weight: 600;">--</div>
              </div>
              <div class="form-group">
                <label>Status do Servidor Redis</label>
                <div id="redisStatus">--</div>
              </div>
              <div class="form-group">
                <label>Serviço Docker</label>
                <div class="mono" style="color: var(--text-muted);">redis:7-alpine (:6379)</div>
              </div>
            </div>
          </div>

          <div class="section-box">
            <div class="section-header">
              <div class="section-title">
                <i class="fa-solid fa-shield-halved" style="color: #60a5fa;"></i>
                Ajuste do Limite por IP
              </div>
            </div>
            <div class="section-body">
              <div class="form-group">
                <label>Taxa de Reabastecimento (Rate req/s por IP)</label>
                <input type="number" id="inputRate" value="5" min="1" />
              </div>
              <div class="form-group">
                <label>Capacidade Máxima de Rajada (Burst Tokens)</label>
                <input type="number" id="inputBurst" value="10" min="1" />
              </div>
              <div style="display: flex; gap: 8px;">
                <button onclick="updateRateLimit()" class="btn">
                  <i class="fa-solid fa-check"></i>
                  Aplicar Limites
                </button>
                <button onclick="setTestPreset(5, 10)" class="btn btn-secondary btn-sm">Preset Teste (5/10)</button>
                <button onclick="setTestPreset(100000, 200000)" class="btn btn-secondary btn-sm">Preset Bench (100k)</button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- ============================================================== -->
      <!-- TAB 5: GENERAL SETTINGS -->
      <!-- ============================================================== -->
      <section id="tab-settings" class="tab-content">
        <div class="settings-grid">
          <!-- Server & Network Mode -->
          <div class="section-box">
            <div class="section-header">
              <div class="section-title">
                <i class="fa-solid fa-network-wired" style="color: #60a5fa;"></i>
                Modo Operacional e Portas
              </div>
            </div>
            <div class="section-body">
              <div class="form-group">
                <label>Modo do Proxy</label>
                <select id="cfgMode">
                  <option value="both">Both (L7 HTTP + L4 TCP)</option>
                  <option value="l7">Apenas L7 (HTTP/1.1 & HTTP/2)</option>
                  <option value="l4">Apenas L4 (TCP Stream)</option>
                </select>
              </div>
              <div class="form-group">
                <label>Porta L7 (HTTP)</label>
                <input type="text" id="cfgL7Listen" value=":8080" disabled />
              </div>
              <div class="form-group">
                <label>Porta L4 (TCP)</label>
                <input type="text" id="cfgL4Listen" value=":9000" disabled />
              </div>
            </div>
          </div>

          <!-- Timeouts & Keep-Alive -->
          <div class="section-box">
            <div class="section-header">
              <div class="section-title">
                <i class="fa-solid fa-hourglass-half" style="color: #fbbf24;"></i>
                Timeouts e Conexões
              </div>
            </div>
            <div class="section-body">
              <div class="form-group">
                <label>Read Timeout</label>
                <input type="text" id="cfgReadTimeout" value="30s" />
              </div>
              <div class="form-group">
                <label>Write Timeout</label>
                <input type="text" id="cfgWriteTimeout" value="30s" />
              </div>
              <div class="form-group">
                <label>Dial Timeout</label>
                <input type="text" id="cfgDialTimeout" value="2s" />
              </div>
            </div>
          </div>

          <!-- Health Check Settings -->
          <div class="section-box">
            <div class="section-header">
              <div class="section-title">
                <i class="fa-solid fa-heart-pulse" style="color: #ec4899;"></i>
                Active Health Check
              </div>
            </div>
            <div class="section-body">
              <div class="form-group">
                <label>Endpoint HTTP</label>
                <input type="text" id="cfgHcPath" value="/health" />
              </div>
              <div class="form-group">
                <label>Intervalo de Ping</label>
                <input type="text" id="cfgHcInterval" value="3s" />
              </div>
              <div class="form-group">
                <label>Unhealthy Threshold (Falhas consecutivas)</label>
                <input type="number" id="cfgHcUnhealthy" value="3" min="1" max="10" />
              </div>
            </div>
          </div>

          <!-- RFC Headers & Security -->
          <div class="section-box">
            <div class="section-header">
              <div class="section-title">
                <i class="fa-solid fa-fingerprint" style="color: #34d399;"></i>
                Cabeçalhos e Protocolo
              </div>
            </div>
            <div class="section-body">
              <div class="form-group">
                <label>Headers Injetados</label>
                <div class="mono" style="color: var(--text-muted); font-size: 12px; line-height: 1.6;">
                  X-Forwarded-For<br>
                  X-Real-IP<br>
                  X-Forwarded-Proto<br>
                  X-Forwarded-Host
                </div>
              </div>
              <div class="form-group">
                <label>Buffer Pooling L4</label>
                <div class="mono" style="color: var(--text-muted);">32 KB por conexão (sync.Pool)</div>
              </div>
              <button onclick="saveGeneralSettings()" class="btn" style="margin-top: 10px; width: 100%; justify-content: center;">
                <i class="fa-solid fa-floppy-disk"></i>
                Salvar Configurações Gerais
              </button>
            </div>
          </div>
        </div>
      </section>

    </main>
  </div>

  <div id="toast">
    <i id="toastIcon" class="fa-solid fa-circle-check" style="color: var(--green);"></i>
    <span id="toastMsg">Mensagem</span>
  </div>

  <!-- Leaflet Maps JS -->
  <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js" integrity="sha256-20nQCchB9co0qIjJZRGuk2/Z9VM+kNiyxNV1lvTlZBo=" crossorigin=""></script>

  <script>
    let map = null;
    let mapMarkers = [];

    function initMap() {
      if (map) return;
      map = L.map('worldMap', {
        center: [20, 0],
        zoom: 2,
        minZoom: 1,
        maxZoom: 6,
        attributionControl: false
      });

      // CartoDB Dark Matter tiles
      L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
        maxZoom: 19,
        subdomains: 'abcd'
      }).addTo(map);
    }

    function updateMapMarkers(geoList) {
      if (!map) initMap();

      mapMarkers.forEach(m => map.removeLayer(m));
      mapMarkers = [];

      if (!geoList) return;

      geoList.forEach(g => {
        const marker = L.circleMarker([g.lat, g.lng], {
          radius: Math.max(6, Math.min(22, 6 + Math.log2(g.count + 1) * 3)),
          fillColor: '#3b82f6',
          color: '#60a5fa',
          weight: 2,
          opacity: 0.9,
          fillOpacity: 0.5
        }).addTo(map);

        marker.bindPopup(
          '<div style="color: #111; font-size: 12px;">' +
            '<strong>' + g.city + ', ' + g.country + '</strong><br>' +
            'Volume: ' + g.count.toLocaleString() + ' requisições' +
          '</div>'
        );
        mapMarkers.push(marker);
      });
    }

    function switchTab(tabId) {
      document.querySelectorAll('.tab-content').forEach(el => el.classList.remove('active'));
      document.querySelectorAll('.sidebar-menu .nav-item').forEach(el => el.classList.remove('active'));

      const target = document.getElementById('tab-' + tabId);
      if (target) target.classList.add('active');

      const titles = {
        'overview': { icon: 'fa-chart-pie', text: 'Visão Geral do Tráfego' },
        'backends': { icon: 'fa-network-wired', text: 'Servidores Upstream' },
        'algorithms': { icon: 'fa-scale-balanced', text: 'Algoritmos de Balanceamento' },
        'ratelimit': { icon: 'fa-shield-halved', text: 'Rate Limiting & Redis' },
        'settings': { icon: 'fa-sliders', text: 'Configurações Gerais' }
      };

      if (titles[tabId]) {
        document.getElementById('pageTitle').innerHTML = 
          '<i class="fa-solid ' + titles[tabId].icon + '" style="color: #60a5fa;"></i> <span>' + titles[tabId].text + '</span>';
      }

      // Highlight active nav item
      const navLinks = document.querySelectorAll('.sidebar-menu .nav-item');
      navLinks.forEach(link => {
        if (link.getAttribute('onclick') && link.getAttribute('onclick').includes(tabId)) {
          link.classList.add('active');
        }
      });

      if (tabId === 'overview' && map) {
        setTimeout(() => { map.invalidateSize(); }, 200);
      }
    }

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
        const [resStatus, resAnalytics] = await Promise.all([
          fetch('/api/status'),
          fetch('/api/analytics')
        ]);

        if (resStatus.ok) {
          const data = await resStatus.json();
          document.getElementById('uptimeBadge').innerText = 'Uptime: ' + data.uptime;
          document.getElementById('sidebarMode').innerText = (data.mode || 'both').toUpperCase() + ' Mode';

          if (data.rate_limit) {
            let stratText = data.rate_limit.strategy ? data.rate_limit.strategy.toUpperCase() : 'MEMORY';
            if (data.rate_limit.strategy && data.rate_limit.strategy.includes('redis')) {
              if (data.rate_limit.healthy) {
                stratText = 'Redis (' + (data.rate_limit.redis_addr || '127.0.0.1:6379') + ')';
                document.getElementById('redisStatus').innerHTML = '<span class="badge badge-online"><i class="fa-solid fa-circle" style="font-size: 6px;"></i> Conectado</span>';
              } else {
                stratText = 'Redis [Fallback Local Ativo]';
                document.getElementById('redisStatus').innerHTML = '<span class="badge badge-warn"><i class="fa-solid fa-circle-exclamation" style="font-size: 8px;"></i> Offline (Fallback)</span>';
              }
            } else {
              document.getElementById('redisStatus').innerText = 'Modo In-Memory';
            }
            document.getElementById('rlStrategy').innerText = stratText;
          }

          if (data.l7) {
            highlightAlgoBtn(data.l7.algorithm);
          }
        }

        if (resAnalytics.ok) {
          const a = await resAnalytics.json();
          document.getElementById('reqLastMin').innerText = (a.requests_last_minute || 0).toLocaleString();
          document.getElementById('reqLastHour').innerText = (a.requests_last_hour || 0).toLocaleString();
          document.getElementById('reqLastDay').innerText = (a.requests_last_day || 0).toLocaleString();
          document.getElementById('totalBlocked').innerText = (a.total_429 || 0).toLocaleString();

          document.getElementById('stat2xx').innerText = (a.total_2xx || 0).toLocaleString();
          document.getElementById('stat429').innerText = (a.total_429 || 0).toLocaleString();
          document.getElementById('stat5xx').innerText = (a.total_5xx || 0).toLocaleString();

          const total = (a.total_2xx + a.total_4xx + a.total_5xx) || 1;
          document.getElementById('bar2xx').style.width = ((a.total_2xx / total) * 100) + '%';
          document.getElementById('bar429').style.width = ((a.total_429 / total) * 100) + '%';
          document.getElementById('bar5xx').style.width = ((a.total_5xx / total) * 100) + '%';

          // Update Top IPs
          const topTbody = document.getElementById('topIPsTable');
          if (a.top_ips && a.top_ips.length > 0) {
            let html = '';
            a.top_ips.forEach(ip => {
              html += '<tr>' +
                '<td class="mono" style="color: #fff;">' + ip.ip + '</td>' +
                '<td class="mono">' + ip.count.toLocaleString() + '</td>' +
                '<td><span class="badge badge-online">Ativo</span></td>' +
              '</tr>';
            });
            topTbody.innerHTML = html;
          } else {
            topTbody.innerHTML = '<tr><td colspan="3" style="text-align: center; color: var(--text-muted); padding: 16px;">Sem tráfego registrado.</td></tr>';
          }

          // Update Leaflet Map
          if (a.geo_locations) {
            updateMapMarkers(a.geo_locations);
          }
        }

        await fetchBackends();
      } catch (err) {
        console.error('Fetch error:', err);
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
          showToast('Servidor ' + target + ' adicionado');
          document.getElementById('newTarget').value = '';
          fetchStatus();
        } else {
          const msg = await res.text();
          showToast('Erro: ' + msg, true);
        }
      } catch (err) {
        showToast('Erro de rede', true);
      }
    }

    async function removeBackend(type, target) {
      if (!confirm('Remover servidor ' + target + '?')) return;
      try {
        const res = await fetch('/api/backends', {
          method: 'DELETE',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ type, target })
        });
        if (res.ok) {
          showToast('Servidor removido');
          fetchStatus();
        } else {
          showToast('Falha ao remover', true);
        }
      } catch (err) {
        showToast('Erro de rede', true);
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
          showToast('Status do servidor atualizado');
          fetchStatus();
        } else {
          showToast('Falha ao atualizar status', true);
        }
      } catch (err) {
        showToast('Erro de rede', true);
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
          showToast('Rate limit ajustado: ' + rate + ' req/s (burst: ' + burst + ')');
          fetchStatus();
        } else {
          showToast('Erro ao salvar limites', true);
        }
      } catch(e) {
        showToast('Erro de conexão', true);
      }
    }

    function setTestPreset(r, b) {
      document.getElementById('inputRate').value = r;
      document.getElementById('inputBurst').value = b;
      updateRateLimit();
    }

    async function saveGeneralSettings() {
      const mode = document.getElementById('cfgMode').value;
      const hcPath = document.getElementById('cfgHcPath').value;
      const hcUnhealthy = parseInt(document.getElementById('cfgHcUnhealthy').value) || 3;

      try {
        const res = await fetch('/api/settings', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            mode: mode,
            health_check: {
              path: hcPath,
              unhealthy_threshold: hcUnhealthy
            }
          })
        });
        if (res.ok) {
          showToast('Configurações gerais salvas com sucesso');
          fetchStatus();
        } else {
          showToast('Erro ao salvar configurações', true);
        }
      } catch (err) {
        showToast('Erro de rede', true);
      }
    }

    // Initialize map on start
    window.addEventListener('DOMContentLoaded', () => {
      initMap();
      fetchStatus();
      setInterval(fetchStatus, 2500);
    });
  </script>
</body>
</html>
`
