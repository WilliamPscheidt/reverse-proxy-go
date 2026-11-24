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
  <title>ProxyGO | Enterprise Console</title>
  
  <!-- FontAwesome 6 -->
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css" integrity="sha512-DTOQO9RWCH3ppGqcWaEA1BIZOC6xxalwEsw9c2QQeAIftl+Vegovlnee1c9QX4TctnWMn13TZye+giMm8e2LwA==" crossorigin="anonymous" referrerpolicy="no-referrer" />

  <!-- jsVectorMap (100% Free SVG Vector Map, Zero API Keys, Zero Watermarks) -->
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/jsvectormap/dist/css/jsvectormap.min.css" />
  <script src="https://cdn.jsdelivr.net/npm/jsvectormap"></script>
  <script src="https://cdn.jsdelivr.net/npm/jsvectormap/dist/maps/world.js"></script>

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
      height: 330px;
      width: 100%;
      border-radius: 6px;
      background: #0d121f;
      position: relative;
      overflow: hidden;
    }

    .jvm-tooltip {
      background: #111827 !important;
      border: 1px solid #374151 !important;
      border-radius: 6px !important;
      color: #f3f4f6 !important;
      font-size: 12px !important;
      padding: 6px 12px !important;
      box-shadow: 0 4px 14px rgba(0, 0, 0, 0.5) !important;
      z-index: 1000 !important;
    }

    .jvm-zoom-btn {
      background-color: #1f2937 !important;
      border: 1px solid #374151 !important;
      border-radius: 4px !important;
      color: #9ca3af !important;
      display: flex !important;
      align-items: center !important;
      justify-content: center !important;
      font-weight: bold !important;
      transition: all 0.15s ease !important;
    }

    .jvm-zoom-btn:hover {
      background-color: #374151 !important;
      color: #fff !important;
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

    /* Modal Backdrop & Card */
    .modal-backdrop {
      position: fixed;
      top: 0; left: 0; right: 0; bottom: 0;
      background: rgba(0, 0, 0, 0.75);
      backdrop-filter: blur(4px);
      z-index: 1000;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 20px;
    }
    .modal-card {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 10px;
      width: 100%;
      max-width: 680px;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.6);
      overflow: hidden;
      animation: modalFadeIn 0.15s ease-out;
    }
    @keyframes modalFadeIn {
      from { opacity: 0; transform: scale(0.96); }
      to { opacity: 1; transform: scale(1); }
    }
    .modal-header {
      padding: 16px 20px;
      border-bottom: 1px solid var(--border);
      display: flex;
      align-items: center;
      justify-content: space-between;
    }
    .modal-body {
      padding: 20px;
      max-height: 70vh;
      overflow-y: auto;
    }
    .btn-close {
      background: transparent;
      border: none;
      color: var(--text-muted);
      font-size: 18px;
      cursor: pointer;
      padding: 4px 8px;
      border-radius: 4px;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.15s ease;
    }
    .btn-close:hover { color: #fff; background: var(--card-sub); }

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

    /* Live Stream, WAF & Enterprise Components */
    .live-pill {
      font-size: 10px;
      font-weight: 700;
      color: #ef4444;
      background: rgba(239, 68, 68, 0.15);
      border: 1px solid rgba(239, 68, 68, 0.3);
      padding: 2px 7px;
      border-radius: 9999px;
      margin-left: auto;
      letter-spacing: 0.5px;
      animation: pulseLive 1.5s infinite;
    }
    @keyframes pulseLive {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.4; }
    }

    .method-badge {
      display: inline-block;
      font-family: ui-monospace, monospace;
      font-size: 11px;
      font-weight: 700;
      padding: 2px 6px;
      border-radius: 4px;
      text-transform: uppercase;
    }
    .method-get { color: #38bdf8; background: rgba(56, 189, 248, 0.12); }
    .method-post { color: #34d399; background: rgba(52, 211, 153, 0.12); }
    .method-put { color: #fbbf24; background: rgba(251, 191, 36, 0.12); }
    .method-delete { color: #f87171; background: rgba(248, 113, 113, 0.12); }

    .status-pill {
      display: inline-flex;
      align-items: center;
      gap: 5px;
      font-family: ui-monospace, monospace;
      font-size: 11px;
      font-weight: 700;
      padding: 2px 8px;
      border-radius: 9999px;
    }
    .status-2xx { color: #34d399; background: rgba(52, 211, 153, 0.15); border: 1px solid rgba(52, 211, 153, 0.3); }
    .status-3xx { color: #60a5fa; background: rgba(96, 165, 250, 0.15); border: 1px solid rgba(96, 165, 250, 0.3); }
    .status-4xx { color: #fbbf24; background: rgba(251, 191, 36, 0.15); border: 1px solid rgba(251, 191, 36, 0.3); }
    .status-5xx { color: #f87171; background: rgba(248, 113, 113, 0.15); border: 1px solid rgba(248, 113, 113, 0.3); }

    /* Switch Toggles */
    .switch-row {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 12px 0;
      border-bottom: 1px solid var(--border);
    }
    .switch-row:last-child { border-bottom: none; }
    .switch-label { font-size: 13px; font-weight: 500; color: #fff; }
    .switch-desc { font-size: 11px; color: var(--text-muted); margin-top: 2px; }

    .switch {
      position: relative;
      display: inline-block;
      width: 44px;
      height: 24px;
    }
    .switch input { opacity: 0; width: 0; height: 0; }
    .slider {
      position: absolute;
      cursor: pointer;
      top: 0; left: 0; right: 0; bottom: 0;
      background-color: var(--card-sub);
      border: 1px solid var(--border);
      transition: .2s;
      border-radius: 24px;
    }
    .slider:before {
      position: absolute;
      content: "";
      height: 16px;
      width: 16px;
      left: 3px;
      bottom: 3px;
      background-color: #94a3b8;
      transition: .2s;
      border-radius: 50%;
    }
    input:checked + .slider { background-color: #2563eb; border-color: #3b82f6; }
    input:checked + .slider:before { transform: translateX(20px); background-color: #fff; }

    /* Circuit Breaker States */
    .cb-card {
      background: var(--card-sub);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 16px;
      display: flex;
      flex-direction: column;
      gap: 10px;
    }
    .badge-cb-closed { color: #34d399; background: rgba(52, 211, 153, 0.15); border: 1px solid rgba(52, 211, 153, 0.3); }
    .badge-cb-open { color: #f87171; background: rgba(248, 113, 113, 0.15); border: 1px solid rgba(248, 113, 113, 0.3); }
    .badge-cb-half { color: #fbbf24; background: rgba(251, 191, 36, 0.15); border: 1px solid rgba(251, 191, 36, 0.3); }

    /* Canary Split Bar */
    .split-track {
      height: 12px;
      width: 100%;
      background: #1e293b;
      border-radius: 9999px;
      overflow: hidden;
      display: flex;
      margin: 8px 0;
    }
    .split-primary { background: #3b82f6; height: 100%; transition: width 0.3s; }
    .split-canary { background: #10b981; height: 100%; transition: width 0.3s; }
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
        <div class="brand-name">ProxyGO</div>
        <div class="brand-mode" id="sidebarMode">L4/L7 Enterprise Proxy</div>
      </div>
    </div>

    <div class="sidebar-menu">
      <div class="nav-label">Navegação</div>
      <a class="nav-item active" onclick="switchTab('overview')">
        <i class="fa-solid fa-chart-pie"></i>
        <span>Overview & Tráfego</span>
      </a>
      <a class="nav-item" onclick="switchTab('stream')">
        <i class="fa-solid fa-tower-broadcast"></i>
        <span>Logs ao Vivo</span>
        <span class="live-pill" id="liveIndicator">LIVE</span>
      </a>
      <a class="nav-item" onclick="switchTab('backends')">
        <i class="fa-solid fa-network-wired"></i>
        <span>Upstream Backends</span>
      </a>
      <a class="nav-item" onclick="switchTab('algorithms')">
        <i class="fa-solid fa-scale-balanced"></i>
        <span>Balanceamento</span>
      </a>

      <div class="nav-label">Segurança & WAF</div>
      <a class="nav-item" onclick="switchTab('waf')">
        <i class="fa-solid fa-shield-halved"></i>
        <span>WAF & Firewall</span>
      </a>
      <a class="nav-item" onclick="switchTab('ratelimit')">
        <i class="fa-solid fa-gauge-high"></i>
        <span>Rate Limit & Redis</span>
      </a>

      <div class="nav-label">Roteamento & Performance</div>
      <a class="nav-item" onclick="switchTab('routing')">
        <i class="fa-solid fa-route"></i>
        <span>Rotas L7 & Canary</span>
      </a>
      <a class="nav-item" onclick="switchTab('resilience')">
        <i class="fa-solid fa-heart-pulse"></i>
        <span>Circuit Breakers</span>
      </a>
      <a class="nav-item" onclick="switchTab('cache')">
        <i class="fa-solid fa-bolt"></i>
        <span>Edge Cache</span>
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
            <div style="display: flex; align-items: center; gap: 12px;">
              <span style="font-size: 11px; color: #60a5fa; background: rgba(37, 99, 235, 0.1); border: 1px solid rgba(37, 99, 235, 0.3); padding: 3px 8px; border-radius: 4px;">
                <i class="fa-solid fa-hand-pointer"></i> Clique em uma região azul para ver e bloquear IPs
              </span>
              <span style="font-size: 12px; color: var(--text-muted);" id="geoSummary">0 regiões ativas</span>
              <button onclick="resetAnalytics()" class="btn btn-secondary btn-sm" title="Zerar histórico de requisições">
                <i class="fa-solid fa-arrows-rotate"></i> Zerar Métricas
              </button>
            </div>
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
                    <th style="text-align: right;">Ação</th>
                  </tr>
                </thead>
                <tbody id="topIPsTable">
                  <tr><td colspan="4" style="text-align: center; color: var(--text-muted);">Aguardando tráfego...</td></tr>
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

          <!-- IP Blacklist Section Box -->
          <div class="section-box" style="grid-column: 1 / -1; margin-top: 10px;">
            <div class="section-header">
              <div class="section-title">
                <i class="fa-solid fa-ban" style="color: #f87171;"></i>
                Lista Negra de IPs (Blacklist - HTTP 403 Forbidden)
              </div>
              <span class="badge badge-offline" id="blacklistCountBadge">0 IPs Bloqueados</span>
            </div>
            <div class="section-body">
              <div style="font-size: 12px; color: var(--text-muted); margin-bottom: 14px;">
                IPs nesta lista têm conexão ou requisições HTTP imediatamente rejeitadas antes de qualquer processamento ou balanceamento.
              </div>
              <form onsubmit="manualBlockIP(event)" style="display: flex; gap: 10px; margin-bottom: 16px;">
                <input type="text" id="manualBlockInput" placeholder="Ex: 189.40.12.5 ou 203.0.113.195" style="flex: 1;" required />
                <input type="text" id="manualBlockReason" placeholder="Motivo do bloqueio (opcional)" style="flex: 1;" />
                <button type="submit" class="btn btn-danger">
                  <i class="fa-solid fa-ban"></i> Adicionar à Blacklist
                </button>
              </form>

              <div class="table-wrapper">
                <table>
                  <thead>
                    <tr>
                      <th>Endereço IP Bloqueado</th>
                      <th>Motivo</th>
                      <th>Data / Hora do Bloqueio</th>
                      <th style="text-align: right;">Ação</th>
                    </tr>
                  </thead>
                  <tbody id="blacklistTable">
                    <tr><td colspan="4" style="text-align: center; color: var(--text-muted); padding: 18px;">Nenhum IP bloqueado no momento.</td></tr>
                  </tbody>
                </table>
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

      <!-- TAB 2: LIVE REQUEST STREAM -->
      <section class="tab-content" id="tab-stream">
        <div class="section-box">
          <div class="section-header" style="flex-wrap: wrap; gap: 12px;">
            <div style="display: flex; align-items: center; gap: 12px;">
              <div class="section-title">
                <i class="fa-solid fa-tower-broadcast" style="color: #ef4444;"></i>
                Feed de Requisições ao Vivo (Server-Sent Events)
              </div>
              <span class="status-badge status-healthy" id="streamConnBadge">
                <span class="status-dot"></span> Conectado
              </span>
              <span class="status-badge" style="background: rgba(56, 189, 248, 0.15); border: 1px solid rgba(56, 189, 248, 0.3); color: #38bdf8;">
                <i class="fa-solid fa-gauge-high" style="font-size: 11px;"></i> <strong id="streamRps">0</strong> req/s
              </span>
            </div>

            <div style="display: flex; align-items: center; gap: 10px;">
              <input type="text" id="streamFilterInput" placeholder="Filtrar por IP, rota ou status..." oninput="filterStreamTable()" style="background: var(--card-sub); border: 1px solid var(--border); color: #fff; padding: 6px 12px; border-radius: 6px; font-size: 12px; width: 220px;" />
              <button onclick="toggleStream()" id="btnStreamPause" class="btn btn-secondary" style="font-size: 12px; padding: 6px 12px;">
                <i class="fa-solid fa-pause"></i> Pausar
              </button>
              <button onclick="clearStreamTable()" class="btn btn-secondary" style="font-size: 12px; padding: 6px 12px;">
                <i class="fa-solid fa-trash"></i> Limpar
              </button>
            </div>
          </div>

          <div class="table-wrapper" style="max-height: 600px; overflow-y: auto;">
            <table>
              <thead>
                <tr>
                  <th style="width: 100px;">Horário</th>
                  <th style="width: 80px;">Método</th>
                  <th>Caminho (Path)</th>
                  <th style="width: 110px;">Status</th>
                  <th style="width: 90px;">Latência</th>
                  <th>Upstream Target</th>
                  <th style="width: 140px;">IP do Cliente</th>
                  <th style="width: 70px;">País</th>
                </tr>
              </thead>
              <tbody id="streamTableBody">
                <tr><td colspan="8" style="text-align: center; color: var(--text-muted); padding: 30px;">Aguardando conexões ao vivo...</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <!-- TAB 3: WAF & SEGURANÇA -->
      <section class="tab-content" id="tab-waf">
        <div class="metrics-grid" style="margin-bottom: 24px;">
          <div class="metric-card">
            <div class="metric-header">
              <span class="metric-title">Total Ameaças Bloqueadas</span>
              <div class="metric-icon" style="background: rgba(239, 68, 68, 0.15); color: #f87171;"><i class="fa-solid fa-shield-virus"></i></div>
            </div>
            <div class="metric-value" id="wafStatTotal">0</div>
            <div class="metric-sub">WAF Rule Engine Ativo</div>
          </div>
          <div class="metric-card">
            <div class="metric-header">
              <span class="metric-title">SQL Injection</span>
              <div class="metric-icon" style="background: rgba(245, 158, 11, 0.15); color: #fbbf24;"><i class="fa-solid fa-database"></i></div>
            </div>
            <div class="metric-value" id="wafStatSQLi">0</div>
            <div class="metric-sub">Bloqueio de UNION, SELECT, DROP</div>
          </div>
          <div class="metric-card">
            <div class="metric-header">
              <span class="metric-title">Cross-Site Scripting (XSS)</span>
              <div class="metric-icon" style="background: rgba(168, 85, 247, 0.15); color: #c084fc;"><i class="fa-solid fa-code"></i></div>
            </div>
            <div class="metric-value" id="wafStatXSS">0</div>
            <div class="metric-sub">Injeção de scripts & payloads</div>
          </div>
          <div class="metric-card">
            <div class="metric-header">
              <span class="metric-title">Bad Bots & Scanners</span>
              <div class="metric-icon" style="background: rgba(236, 72, 153, 0.15); color: #f472b6;"><i class="fa-solid fa-robot"></i></div>
            </div>
            <div class="metric-value" id="wafStatBots">0</div>
            <div class="metric-sub">sqlmap, nikto, masscan, etc.</div>
          </div>
        </div>

        <div class="settings-grid">
          <div class="section-box">
            <div class="section-header">
              <div class="section-title"><i class="fa-solid fa-sliders" style="color: #60a5fa;"></i> Regras de Inspeção do WAF</div>
            </div>
            <div class="section-body">
              <div class="switch-row">
                <div>
                  <div class="switch-label">WAF Ativo Globalmente</div>
                  <div class="switch-desc">Habilita inspeção de payloads e headers em tempo real</div>
                </div>
                <label class="switch"><input type="checkbox" id="wafToggleEnabled" checked><span class="slider"></span></label>
              </div>
              <div class="switch-row">
                <div>
                  <div class="switch-label">Bloqueio de SQL Injection (SQLi)</div>
                  <div class="switch-desc">Intercepta queries SQL maliciosas em parâmetros e caminhos</div>
                </div>
                <label class="switch"><input type="checkbox" id="wafToggleSQLi" checked><span class="slider"></span></label>
              </div>
              <div class="switch-row">
                <div>
                  <div class="switch-label">Bloqueio de Cross-Site Scripting (XSS)</div>
                  <div class="switch-desc">Rejeita tags de script, manipuladores de evento e javascript:</div>
                </div>
                <label class="switch"><input type="checkbox" id="wafToggleXSS" checked><span class="slider"></span></label>
              </div>
              <div class="switch-row">
                <div>
                  <div class="switch-label">Bloqueio de Path Traversal & LFI</div>
                  <div class="switch-desc">Bloqueia tentativas de navegação de diretório (/etc/passwd, ../..)</div>
                </div>
                <label class="switch"><input type="checkbox" id="wafTogglePath" checked><span class="slider"></span></label>
              </div>
              <div class="switch-row">
                <div>
                  <div class="switch-label">Bloqueio de Bad Bots & Scanners</div>
                  <div class="switch-desc">Interrompe ferramentas automatizadas de exploração de vulnerabilidades</div>
                </div>
                <label class="switch"><input type="checkbox" id="wafToggleBots" checked><span class="slider"></span></label>
              </div>
            </div>
          </div>

          <div class="section-box">
            <div class="section-header">
              <div class="section-title"><i class="fa-solid fa-earth-americas" style="color: #34d399;"></i> Geo-Blocking & Sub-redes CIDR</div>
            </div>
            <div class="section-body">
              <div class="form-group">
                <label>Países Bloqueados (ISO 2 letras, separados por vírgula)</label>
                <input type="text" id="wafBlockedCountries" placeholder="ex: RU, CN, KP" />
              </div>
              <div class="form-group">
                <label>Sub-redes CIDR Bloqueadas (separadas por vírgula)</label>
                <input type="text" id="wafBlockedCIDRs" placeholder="ex: 10.0.0.0/8, 192.168.100.0/24" />
              </div>
              <button onclick="saveWAFSettings()" class="btn" style="margin-top: 10px; width: 100%; justify-content: center;">
                <i class="fa-solid fa-floppy-disk"></i> Salvar Políticas do WAF
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- TAB 4: ROTAS & CANARY -->
      <section class="tab-content" id="tab-routing">
        <div class="section-box" style="margin-bottom: 24px;">
          <div class="section-header">
            <div class="section-title"><i class="fa-solid fa-code-fork" style="color: #60a5fa;"></i> Divisão de Tráfego Canary</div>
          </div>
          <div class="section-body">
            <p style="color: var(--text-muted); font-size: 13px; margin-bottom: 16px;">
              Distribua percentualmente o tráfego de produção entre a versão estável (Primary) e a nova versão em validação (Canary) sem tempo de inatividade.
            </p>
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px;">
              <span style="font-size: 13px; font-weight: 600; color: #60a5fa;"><i class="fa-solid fa-circle" style="font-size: 8px;"></i> Versão Estável (Primary): <strong id="canaryPrimaryPct">90%</strong></span>
              <span style="font-size: 13px; font-weight: 600; color: #34d399;"><i class="fa-solid fa-circle" style="font-size: 8px;"></i> Versão Canary: <strong id="canaryCanaryPct">10%</strong></span>
            </div>
            <div class="split-track">
              <div class="split-primary" id="barPrimary" style="width: 90%;"></div>
              <div class="split-canary" id="barCanary" style="width: 10%;"></div>
            </div>
            <div style="display: flex; gap: 12px; align-items: center; margin-top: 14px;">
              <input type="range" id="canarySlider" min="0" max="100" value="10" oninput="updateCanaryDisplay(this.value)" style="flex: 1;" />
              <button onclick="applyCanaryWeight()" class="btn" style="padding: 6px 16px;">
                <i class="fa-solid fa-check"></i> Aplicar Peso
              </button>
            </div>
          </div>
        </div>

        <div class="section-box">
          <div class="section-header">
            <div class="section-title"><i class="fa-solid fa-route" style="color: #a78bfa;"></i> Rotas L7 Ativas</div>
          </div>
          <div class="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>ID da Rota</th>
                  <th>Host / VirtualHost</th>
                  <th>Prefixo do Caminho</th>
                  <th>Métodos</th>
                  <th>Canary Weight</th>
                  <th>Total Hits</th>
                  <th>Canary Hits</th>
                </tr>
              </thead>
              <tbody id="routesTableBody">
                <tr><td colspan="7" style="text-align: center; color: var(--text-muted); padding: 20px;">Carregando rotas...</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <!-- TAB 5: RESILIÊNCIA & CIRCUIT BREAKERS -->
      <section class="tab-content" id="tab-resilience">
        <div class="metrics-grid" style="margin-bottom: 24px;">
          <div class="metric-card">
            <div class="metric-header">
              <span class="metric-title">Padrão Disjuntor</span>
              <div class="metric-icon" style="background: rgba(16, 185, 129, 0.15); color: #34d399;"><i class="fa-solid fa-shield-halved"></i></div>
            </div>
            <div class="metric-value">Circuit Breaker</div>
            <div class="metric-sub">Closed • Open • Half-Open</div>
          </div>
          <div class="metric-card">
            <div class="metric-header">
              <span class="metric-title">Retry Automático</span>
              <div class="metric-icon" style="background: rgba(59, 130, 246, 0.15); color: #60a5fa;"><i class="fa-solid fa-rotate-right"></i></div>
            </div>
            <div class="metric-value">2 Tentativas</div>
            <div class="metric-sub">Exponential Backoff & Jitter</div>
          </div>
          <div class="metric-card">
            <div class="metric-header">
              <span class="metric-title">Gatilhos de Falha</span>
              <div class="metric-icon" style="background: rgba(245, 158, 11, 0.15); color: #fbbf24;"><i class="fa-solid fa-triangle-exclamation"></i></div>
            </div>
            <div class="metric-value">5 Falhas</div>
            <div class="metric-sub">Cooldown de 5s para Half-Open</div>
          </div>
          <div class="metric-card">
            <div class="metric-header">
              <span class="metric-title">Status Global</span>
              <div class="metric-icon" style="background: rgba(52, 211, 153, 0.15); color: #34d399;"><i class="fa-solid fa-circle-check"></i></div>
            </div>
            <div class="metric-value" style="color: #34d399;" id="resilienceHealth">100% OK</div>
            <div class="metric-sub">Todos os circuitos fechados</div>
          </div>
        </div>

        <div class="section-box">
          <div class="section-header">
            <div class="section-title"><i class="fa-solid fa-heart-pulse" style="color: #ec4899;"></i> Circuit Breakers por Servidor Upstream</div>
          </div>
          <div class="section-body">
            <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px;" id="circuitBreakersGrid">
              <!-- Dynamically populated -->
            </div>
          </div>
        </div>
      </section>

      <!-- TAB 6: EDGE CACHE -->
      <section class="tab-content" id="tab-cache">
        <div class="metrics-grid" style="margin-bottom: 24px;">
          <div class="metric-card">
            <div class="metric-header">
              <span class="metric-title">Cache Hit Ratio</span>
              <div class="metric-icon" style="background: rgba(16, 185, 129, 0.15); color: #34d399;"><i class="fa-solid fa-chart-simple"></i></div>
            </div>
            <div class="metric-value" id="cacheHitRatioVal">0%</div>
            <div class="metric-sub">Latência de entrega sub-0.3ms</div>
          </div>
          <div class="metric-card">
            <div class="metric-header">
              <span class="metric-title">Total Cache Hits</span>
              <div class="metric-icon" style="background: rgba(59, 130, 246, 0.15); color: #60a5fa;"><i class="fa-solid fa-bolt"></i></div>
            </div>
            <div class="metric-value" id="cacheHitsVal">0</div>
            <div class="metric-sub">Respostas servidas da memória/Redis</div>
          </div>
          <div class="metric-card">
            <div class="metric-header">
              <span class="metric-title">Cache Misses</span>
              <div class="metric-icon" style="background: rgba(245, 158, 11, 0.15); color: #fbbf24;"><i class="fa-solid fa-forward"></i></div>
            </div>
            <div class="metric-value" id="cacheMissesVal">0</div>
            <div class="metric-sub">Encaminhados para o upstream</div>
          </div>
          <div class="metric-card">
            <div class="metric-header">
              <span class="metric-title">URLs em Cache</span>
              <div class="metric-icon" style="background: rgba(168, 85, 247, 0.15); color: #c084fc;"><i class="fa-solid fa-database"></i></div>
            </div>
            <div class="metric-value" id="cachedEntriesVal">0</div>
            <div class="metric-sub">Suporte a ETag & 304 Not Modified</div>
          </div>
        </div>

        <div class="section-box">
          <div class="section-header">
            <div class="section-title"><i class="fa-solid fa-broom" style="color: #f87171;"></i> Invalidação & Purge de Cache</div>
          </div>
          <div class="section-body">
            <p style="color: var(--text-muted); font-size: 13px; margin-bottom: 14px;">
              Purge de respostas cacheadas instantaneamente. Digite <code class="mono" style="color: #60a5fa;">*</code> para limpar todo o cache ou forneça um caminho específico.
            </p>
            <div style="display: flex; gap: 12px; max-width: 500px;">
              <input type="text" id="purgePatternInput" value="*" placeholder="Padrão de URL (ex: * ou /api/users)" class="form-control" />
              <button onclick="purgeCache()" class="btn btn-secondary" style="border-color: rgba(239, 68, 68, 0.4); color: #f87171; white-space: nowrap;">
                <i class="fa-solid fa-trash-can"></i> Limpar Cache (Purge)
              </button>
            </div>
          </div>
        </div>
      </section>

    </main>
  </div>

  <!-- Region IPs & Blacklist Modal -->
  <div id="regionModal" class="modal-backdrop" style="display: none;" onclick="closeModalOnBackdrop(event)">
    <div class="modal-card">
      <div class="modal-header">
        <div style="display: flex; align-items: center; gap: 12px;">
          <div style="width: 34px; height: 34px; background: rgba(37, 99, 235, 0.15); border: 1px solid rgba(96, 165, 250, 0.3); border-radius: 8px; display: flex; align-items: center; justify-content: center;">
            <i class="fa-solid fa-location-dot" style="color: #60a5fa; font-size: 16px;"></i>
          </div>
          <div>
            <div style="font-size: 15px; font-weight: 600; color: #fff;" id="modalRegionTitle">São Paulo, Brasil</div>
            <div style="font-size: 11px; color: var(--text-muted);" id="modalRegionSub">Tráfego registrado por IP</div>
          </div>
        </div>
        <button onclick="closeRegionModal()" class="btn-close" title="Fechar"><i class="fa-solid fa-xmark"></i></button>
      </div>

      <div class="modal-body">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px;">
          <div style="font-size: 12px; color: var(--text-muted);">
            Endereços IP detectados nesta região. Bloqueie IPs abusivos para rejeitar acessos com <strong style="color: #f87171;">HTTP 403 Forbidden</strong>.
          </div>
          <span class="badge badge-protocol" id="modalIPCount">0 IPs</span>
        </div>

        <div class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Endereço IP</th>
                <th>Volume</th>
                <th>Status</th>
                <th style="text-align: right;">Ação de Segurança</th>
              </tr>
            </thead>
            <tbody id="modalIPsTable">
              <tr><td colspan="4" style="text-align: center; color: var(--text-muted); padding: 18px;">Carregando dados da região...</td></tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>

  <div id="toast">
    <i id="toastIcon" class="fa-solid fa-circle-check" style="color: var(--green);"></i>
    <span id="toastMsg">Mensagem</span>
  </div>

  <script>
    let vectorMap = null;
    let activeGeoLocations = [];
    let currentModalRegion = null;

    function initMap() {
      if (vectorMap) return;
      if (typeof jsVectorMap === 'undefined') return;

      try {
        vectorMap = new jsVectorMap({
          selector: '#worldMap',
          map: 'world',
          backgroundColor: '#0d121f',
          draggable: true,
          zoomButtons: true,
          zoomOnScroll: false,
          regionStyle: {
            initial: {
              fill: '#1e293b',
              fillOpacity: 1,
              stroke: '#334155',
              strokeWidth: 0.5
            },
            hover: {
              fill: '#2563eb',
              fillOpacity: 0.85
            }
          },
          markerStyle: {
            initial: {
              r: 6,
              fill: '#3b82f6',
              stroke: '#93c5fd',
              strokeWidth: 2,
              fillOpacity: 0.95
            },
            hover: {
              fill: '#60a5fa',
              stroke: '#ffffff',
              strokeWidth: 3,
              cursor: 'pointer'
            }
          },
          markers: [],
          onMarkerClick: function(event, index) {
            if (activeGeoLocations && activeGeoLocations[index]) {
              openRegionModal(activeGeoLocations[index]);
            }
          },
          onMarkerTooltipShow: function(event, tooltip, index) {
            const m = vectorMap.params.markers[index];
            if (m) {
              tooltip.text(
                '<div style="font-weight: 600; color: #fff; margin-bottom: 3px;">' + m.name + '</div>' +
                '<div style="font-size: 11px; color: #9ca3af;">Volume: <strong style="color: #60a5fa;">' + 
                (m.count || 0).toLocaleString() + '</strong> requisições</div>' +
                '<div style="font-size: 10px; color: #60a5fa; margin-top: 4px;"><i class="fa-solid fa-hand-pointer"></i> Clique para ver IPs e gerenciar blacklist</div>',
                true
              );
            }
          }
        });
      } catch (err) {
        console.error('jsVectorMap init error:', err);
      }
    }

    function updateMapMarkers(geoList) {
      if (!vectorMap) initMap();
      if (!vectorMap) return;

      activeGeoLocations = geoList || [];

      try {
        vectorMap.removeMarkers();

        if (!geoList || geoList.length === 0) return;

        const markers = geoList.map(g => ({
          name: g.city + ', ' + g.country,
          coords: [g.lat, g.lng],
          count: g.count,
          style: {
            r: Math.max(6, Math.min(18, 6 + Math.log2(g.count + 1) * 2.2)),
            fill: g.count > 0 ? '#3b82f6' : '#64748b',
            stroke: g.count > 0 ? '#93c5fd' : '#475569'
          }
        }));

        vectorMap.addMarkers(markers);

        // Keep modal live updated if open
        if (currentModalRegion) {
          const updated = activeGeoLocations.find(g => g.id === currentModalRegion.id || (g.city === currentModalRegion.city && g.country === currentModalRegion.country));
          if (updated) {
            currentModalRegion = updated;
            renderModalIPsTable(updated);
          }
        }
      } catch (err) {
        console.error('Update map error:', err);
      }
    }

    function switchTab(tabId) {
      document.querySelectorAll('.tab-content').forEach(el => el.classList.remove('active'));
      document.querySelectorAll('.sidebar-menu .nav-item').forEach(el => el.classList.remove('active'));

      const target = document.getElementById('tab-' + tabId);
      if (target) target.classList.add('active');

      const titles = {
        'overview': { icon: 'fa-chart-pie', text: 'Visão Geral do Tráfego' },
        'stream': { icon: 'fa-tower-broadcast', text: 'Logs & Feed de Requisições ao Vivo' },
        'backends': { icon: 'fa-network-wired', text: 'Servidores Upstream' },
        'algorithms': { icon: 'fa-scale-balanced', text: 'Algoritmos de Balanceamento' },
        'waf': { icon: 'fa-shield-halved', text: 'Web Application Firewall (WAF) & Segurança' },
        'ratelimit': { icon: 'fa-gauge-high', text: 'Rate Limiting & Redis' },
        'routing': { icon: 'fa-route', text: 'Roteamento L7 & Canary Deployments' },
        'resilience': { icon: 'fa-heart-pulse', text: 'Resiliência: Circuit Breakers & Retries' },
        'cache': { icon: 'fa-bolt', text: 'Edge Cache Distribuído' },
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

      if (tabId === 'overview' && vectorMap && vectorMap.updateSize) {
        setTimeout(() => { vectorMap.updateSize(); }, 200);
      } else if (tabId === 'waf') {
        fetchWAF();
      } else if (tabId === 'routing') {
        fetchRoutes();
      } else if (tabId === 'resilience') {
        fetchCircuitBreakers();
      } else if (tabId === 'cache') {
        fetchCacheStats();
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
          if (document.getElementById('blacklistCountBadge')) {
            document.getElementById('blacklistCountBadge').innerText = (data.blacklist_count || 0) + ' IPs Bloqueados';
          }

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
              const statusBadge = ip.blacklisted
                ? '<span class="badge badge-offline"><i class="fa-solid fa-ban" style="font-size: 8px;"></i> Blacklist</span>'
                : '<span class="badge badge-online"><i class="fa-solid fa-circle" style="font-size: 6px;"></i> Permitido</span>';

              const actionBtn = ip.blacklisted
                ? '<button onclick="unblockIP(\'' + ip.ip + '\')" class="btn btn-secondary btn-sm" style="color: #34d399; font-size: 11px; padding: 2px 7px;" title="Desbloquear"><i class="fa-solid fa-shield"></i> Desbloquear</button>'
                : '<button onclick="blockIP(\'' + ip.ip + '\')" class="btn btn-danger btn-sm" style="font-size: 11px; padding: 2px 7px;" title="Bloquear IP com HTTP 403"><i class="fa-solid fa-ban"></i> Bloquear</button>';

              html += '<tr>' +
                '<td><strong class="mono" style="color: #fff;">' + ip.ip + '</strong></td>' +
                '<td class="mono">' + ip.count.toLocaleString() + '</td>' +
                '<td>' + statusBadge + '</td>' +
                '<td style="text-align: right;">' + actionBtn + '</td>' +
              '</tr>';
            });
            topTbody.innerHTML = html;
          } else {
            topTbody.innerHTML = '<tr><td colspan="4" style="text-align: center; color: var(--text-muted); padding: 16px;">Sem tráfego registrado.</td></tr>';
          }

          // Update Vector Map & Summary
          const activeGeos = a.geo_locations ? a.geo_locations.length : 0;
          document.getElementById('geoSummary').innerText = activeGeos + (activeGeos === 1 ? ' região ativa' : ' regiões ativas');
          if (a.geo_locations) {
            updateMapMarkers(a.geo_locations);
          }
        }

        await fetchBackends();
        await fetchBlacklist();
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

    async function resetAnalytics() {
      if (!confirm('Deseja zerar o histórico e contadores de requisições?')) return;
      try {
        const res = await fetch('/api/analytics/reset', { method: 'POST' });
        if (res.ok) {
          showToast('Métricas zeradas com sucesso');
          fetchStatus();
        }
      } catch (err) {
        showToast('Erro ao zerar métricas', true);
      }
    }

    function openRegionModal(region) {
      currentModalRegion = region;
      document.getElementById('modalRegionTitle').innerText = region.city + ', ' + region.country;
      document.getElementById('modalRegionSub').innerText = (region.count || 0).toLocaleString() + ' requisições acumuladas';

      renderModalIPsTable(region);
      document.getElementById('regionModal').style.display = 'flex';
    }

    function closeRegionModal() {
      document.getElementById('regionModal').style.display = 'none';
      currentModalRegion = null;
    }

    function closeModalOnBackdrop(e) {
      if (e.target && e.target.id === 'regionModal') {
        closeRegionModal();
      }
    }

    function renderModalIPsTable(region) {
      const tbody = document.getElementById('modalIPsTable');
      const countBadge = document.getElementById('modalIPCount');

      if (!region.ips || region.ips.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" style="text-align: center; color: var(--text-muted); padding: 22px;">Nenhum IP individual registrado nesta região.</td></tr>';
        countBadge.innerText = '0 IPs';
        return;
      }

      countBadge.innerText = region.ips.length + (region.ips.length === 1 ? ' IP' : ' IPs');
      let html = '';
      region.ips.forEach(item => {
        const statusBadge = item.blacklisted
          ? '<span class="badge badge-offline"><i class="fa-solid fa-ban" style="font-size: 8px;"></i> Blacklist (403)</span>'
          : '<span class="badge badge-online"><i class="fa-solid fa-circle" style="font-size: 6px;"></i> Permitido</span>';

        const actionBtn = item.blacklisted
          ? '<button onclick="unblockIP(\'' + item.ip + '\')" class="btn btn-secondary btn-sm" style="color: #34d399; border-color: rgba(16, 185, 129, 0.4);" title="Permitir acesso"><i class="fa-solid fa-shield"></i> Desbloquear</button>'
          : '<button onclick="blockIP(\'' + item.ip + '\')" class="btn btn-danger btn-sm" title="Bloquear IP com HTTP 403"><i class="fa-solid fa-ban"></i> Bloquear (Blacklist)</button>';

        html += '<tr>' +
          '<td><strong class="mono" style="color: #fff;">' + item.ip + '</strong></td>' +
          '<td class="mono">' + item.count.toLocaleString() + ' reqs</td>' +
          '<td>' + statusBadge + '</td>' +
          '<td style="text-align: right;">' + actionBtn + '</td>' +
        '</tr>';
      });

      tbody.innerHTML = html;
    }

    async function blockIP(ip, reason) {
      try {
        const res = await fetch('/api/blacklist', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ ip: ip, reason: reason || 'Bloqueado via dashboard administrativo' })
        });
        if (res.ok) {
          showToast('IP ' + ip + ' bloqueado na Blacklist (HTTP 403)');
          await fetchStatus();
          if (currentModalRegion) {
            const found = activeGeoLocations.find(g => g.id === currentModalRegion.id || (g.city === currentModalRegion.city && g.country === currentModalRegion.country));
            if (found) {
              currentModalRegion = found;
              renderModalIPsTable(found);
            }
          }
        } else {
          showToast('Erro ao bloquear IP', true);
        }
      } catch (err) {
        showToast('Erro de conexão ao bloquear IP', true);
      }
    }

    async function unblockIP(ip) {
      try {
        const res = await fetch('/api/blacklist', {
          method: 'DELETE',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ ip: ip })
        });
        if (res.ok) {
          showToast('IP ' + ip + ' desbloqueado com sucesso');
          await fetchStatus();
          if (currentModalRegion) {
            const found = activeGeoLocations.find(g => g.id === currentModalRegion.id || (g.city === currentModalRegion.city && g.country === currentModalRegion.country));
            if (found) {
              currentModalRegion = found;
              renderModalIPsTable(found);
            }
          }
        } else {
          showToast('Erro ao desbloquear IP', true);
        }
      } catch (err) {
        showToast('Erro de conexão ao desbloquear IP', true);
      }
    }

    async function manualBlockIP(e) {
      e.preventDefault();
      const input = document.getElementById('manualBlockInput');
      const reasonInput = document.getElementById('manualBlockReason');
      const ip = input.value.trim();
      const reason = reasonInput.value.trim() || 'Bloqueio manual via formulário';
      if (!ip) return;

      await blockIP(ip, reason);
      input.value = '';
      reasonInput.value = '';
    }

    async function fetchBlacklist() {
      try {
        const res = await fetch('/api/blacklist');
        if (!res.ok) return;
        const list = await res.json();
        const tbody = document.getElementById('blacklistTable');
        if (!tbody) return;

        if (!list || list.length === 0) {
          tbody.innerHTML = '<tr><td colspan="4" style="text-align: center; color: var(--text-muted); padding: 18px;">Nenhum IP bloqueado no momento.</td></tr>';
          return;
        }

        let html = '';
        list.forEach(entry => {
          const dateStr = entry.created_at ? new Date(entry.created_at).toLocaleTimeString() : 'Recentemente';
          html += '<tr>' +
            '<td><strong class="mono" style="color: #f87171;"><i class="fa-solid fa-ban" style="font-size: 10px; margin-right: 6px;"></i>' + entry.ip + '</strong></td>' +
            '<td style="color: var(--text-muted);">' + (entry.reason || 'Bloqueado') + '</td>' +
            '<td class="mono" style="color: var(--text-muted);">' + dateStr + '</td>' +
            '<td style="text-align: right;">' +
              '<button onclick="unblockIP(\'' + entry.ip + '\')" class="btn btn-secondary btn-sm" style="color: #34d399;" title="Desbloquear"><i class="fa-solid fa-shield"></i> Desbloquear</button>' +
            '</td>' +
          '</tr>';
        });
        tbody.innerHTML = html;
      } catch (err) {
        console.error('Fetch blacklist error:', err);
      }
    }

    /* --- Live Stream (SSE) Logic with Batched DOM Updates & RPS Meter --- */
    let eventSource = null;
    let streamPaused = false;
    let streamQueue = [];
    let streamRpsCounter = 0;
    let isBatching = false;

    function initEventStream() {
      if (eventSource) return;
      try {
        eventSource = new EventSource('/api/stream/requests');
        eventSource.onopen = () => {
          const badge = document.getElementById('streamConnBadge');
          if (badge) badge.className = 'status-badge status-healthy';
        };
        eventSource.onmessage = (e) => {
          streamRpsCounter++;
          if (streamPaused) return;
          try {
            const data = JSON.parse(e.data);
            streamQueue.push(data);
            if (streamQueue.length > 500) {
              streamQueue.splice(0, streamQueue.length - 200); // keep memory bounded under heavy flood
            }
          } catch (err) {}
        };
        eventSource.onerror = () => {
          const badge = document.getElementById('streamConnBadge');
          if (badge) badge.className = 'status-badge status-unhealthy';
        };

        // Start DOM render loop (10 batches per second max, zero frame drop)
        setInterval(flushStreamBatch, 100);

        // Start RPS meter (computed every 1000ms)
        setInterval(() => {
          const rpsEl = document.getElementById('streamRps');
          if (rpsEl) {
            rpsEl.innerText = streamRpsCounter.toLocaleString();
          }
          streamRpsCounter = 0;
        }, 1000);

      } catch (err) {
        console.error('SSE connection failed:', err);
      }
    }

    function flushStreamBatch() {
      if (streamPaused || streamQueue.length === 0) return;
      const tbody = document.getElementById('streamTableBody');
      if (!tbody) return;

      if (tbody.children.length === 1 && tbody.children[0].innerText.includes('Aguardando')) {
        tbody.innerHTML = '';
      }

      // Pop up to 25 items per batch
      const batch = streamQueue.splice(0, 25);
      const frag = document.createDocumentFragment();

      for (let i = batch.length - 1; i >= 0; i--) {
        const evt = batch[i];
        const tr = document.createElement('tr');
        const timeStr = new Date(evt.timestamp).toLocaleTimeString();
        const method = (evt.method || 'GET').toLowerCase();

        let statusClass = 'status-2xx';
        if (evt.status_code >= 300 && evt.status_code < 400) statusClass = 'status-3xx';
        else if (evt.status_code >= 400 && evt.status_code < 500) statusClass = 'status-4xx';
        else if (evt.status_code >= 500) statusClass = 'status-5xx';

        let upstreamBadge = '<span class="mono">' + (evt.upstream || '-') + '</span>';
        if (evt.cache_hit) {
          upstreamBadge = '<span class="badge" style="background: rgba(16, 185, 129, 0.2); color: #34d399; font-weight: 700;"><i class="fa-solid fa-bolt"></i> CACHE HIT</span>';
        } else if (evt.waf_blocked) {
          upstreamBadge = '<span class="badge" style="background: rgba(239, 68, 68, 0.2); color: #f87171; font-weight: 700;"><i class="fa-solid fa-shield-virus"></i> WAF [' + (evt.waf_rule || 'BLOCK') + ']</span>';
        }

        tr.innerHTML = 
          '<td class="mono" style="color: var(--text-muted);">' + timeStr + '</td>' +
          '<td><span class="method-badge method-' + method + '">' + (evt.method || 'GET') + '</span></td>' +
          '<td class="mono" style="font-weight: 500; max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">' + (evt.path || '/') + '</td>' +
          '<td><span class="status-pill ' + statusClass + '">' + (evt.status_code || 200) + '</span></td>' +
          '<td class="mono" style="color: #94a3b8;">' + (evt.duration_ms || 0.1).toFixed(2) + 'ms</td>' +
          '<td>${upstreamBadge}</td>' +
          '<td class="mono">' + (evt.client_ip || '127.0.0.1') + '</td>' +
          '<td><span class="badge badge-protocol">' + (evt.country || 'US') + '</span></td>';

        frag.appendChild(tr);
      }

      window.requestAnimationFrame(() => {
        tbody.insertBefore(frag, tbody.firstChild);
        while (tbody.children.length > 200) {
          tbody.removeChild(tbody.lastChild);
        }
      });
    }

    function toggleStream() {
      streamPaused = !streamPaused;
      const btn = document.getElementById('btnStreamPause');
      if (btn) {
        btn.innerHTML = streamPaused ? '<i class="fa-solid fa-play"></i> Retomar' : '<i class="fa-solid fa-pause"></i> Pausar';
      }
    }

    function clearStreamTable() {
      streamQueue = [];
      const tbody = document.getElementById('streamTableBody');
      if (tbody) tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--text-muted); padding: 30px;">Feed limpo. Aguardando novas conexões...</td></tr>';
    }

    function filterStreamTable() {
      const q = (document.getElementById('streamFilterInput').value || '').toLowerCase();
      const rows = document.querySelectorAll('#streamTableBody tr');
      rows.forEach(r => {
        const text = r.innerText.toLowerCase();
        r.style.display = text.includes(q) ? '' : 'none';
      });
    }

    /* --- WAF API Logic --- */
    async function fetchWAF() {
      try {
        const res = await fetch('/api/waf');
        if (!res.ok) return;
        const data = await res.json();
        if (data.config) {
          document.getElementById('wafToggleEnabled').checked = data.config.enabled;
          document.getElementById('wafToggleSQLi').checked = data.config.block_sqli;
          document.getElementById('wafToggleXSS').checked = data.config.block_xss;
          document.getElementById('wafTogglePath').checked = data.config.block_path_traversal;
          document.getElementById('wafToggleBots').checked = data.config.block_bad_bots;
          document.getElementById('wafBlockedCountries').value = (data.config.blocked_countries || []).join(', ');
          document.getElementById('wafBlockedCIDRs').value = (data.config.blocked_cidrs || []).join(', ');
        }
        if (data.stats) {
          document.getElementById('wafStatTotal').innerText = data.stats.total_blocked || 0;
          const rules = data.stats.blocked_by_rule || {};
          document.getElementById('wafStatSQLi').innerText = rules['SQL_INJECTION'] || 0;
          document.getElementById('wafStatXSS').innerText = rules['CROSS_SITE_SCRIPTING'] || 0;
          document.getElementById('wafStatBots').innerText = rules['BAD_BOT'] || 0;
        }
      } catch (err) {}
    }

    async function saveWAFSettings() {
      const countries = document.getElementById('wafBlockedCountries').value.split(',').map(s => s.trim().toUpperCase()).filter(s => s);
      const cidrs = document.getElementById('wafBlockedCIDRs').value.split(',').map(s => s.trim()).filter(s => s);
      const payload = {
        enabled: document.getElementById('wafToggleEnabled').checked,
        block_sqli: document.getElementById('wafToggleSQLi').checked,
        block_xss: document.getElementById('wafToggleXSS').checked,
        block_path_traversal: document.getElementById('wafTogglePath').checked,
        block_bad_bots: document.getElementById('wafToggleBots').checked,
        blocked_countries: countries,
        allowed_countries: [],
        blocked_cidrs: cidrs
      };
      try {
        const res = await fetch('/api/waf', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });
        if (res.ok) showToast('Políticas do WAF atualizadas com sucesso!');
        else showToast('Falha ao atualizar WAF', true);
      } catch (err) {
        showToast('Erro de conexão ao salvar WAF', true);
      }
    }

    /* --- Routes & Canary Logic --- */
    async function fetchRoutes() {
      try {
        const res = await fetch('/api/routes');
        if (!res.ok) return;
        const routes = await res.json();
        const tbody = document.getElementById('routesTableBody');
        if (!tbody) return;
        if (!routes || routes.length === 0) {
          tbody.innerHTML = '<tr><td colspan="7" style="text-align: center; color: var(--text-muted); padding: 20px;">Nenhuma rota customizada. Tráfego direcionado à pool padrão L7.</td></tr>';
          return;
        }
        tbody.innerHTML = routes.map(r => 
          '<tr>' +
            '<td class="mono" style="font-weight: 600; color: #fff;">' + r.config.id + '</td>' +
            '<td class="mono">' + (r.config.host || '*') + '</td>' +
            '<td class="mono" style="color: #60a5fa;">' + r.config.path_prefix + '</td>' +
            '<td>' + ((r.config.methods && r.config.methods.length) ? r.config.methods.join(', ') : 'ALL') + '</td>' +
            '<td><span class="badge" style="background: rgba(16, 185, 129, 0.15); color: #34d399; font-weight: 600;">' + r.config.canary_weight + '%</span></td>' +
            '<td class="mono">' + (r.total_hits || 0) + '</td>' +
            '<td class="mono" style="color: #34d399;">' + (r.canary_hits || 0) + '</td>' +
          '</tr>'
        ).join('');
      } catch (err) {}
    }

    function updateCanaryDisplay(val) {
      document.getElementById('canaryCanaryPct').innerText = val + '%';
      document.getElementById('canaryPrimaryPct').innerText = (100 - val) + '%';
      document.getElementById('barCanary').style.width = val + '%';
      document.getElementById('barPrimary').style.width = (100 - val) + '%';
    }

    async function applyCanaryWeight() {
      const weight = parseInt(document.getElementById('canarySlider').value, 10);
      try {
        const res = await fetch('/api/routes/canary', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ id: 'default', weight: weight })
        });
        showToast('Peso do Canary atualizado para ' + weight + '%!');
        fetchRoutes();
      } catch (err) {
        showToast('Peso do Canary aplicado em memória!', false);
      }
    }

    /* --- Circuit Breakers Logic --- */
    async function fetchCircuitBreakers() {
      try {
        const res = await fetch('/api/circuit-breakers');
        if (!res.ok) return;
        const list = await res.json();
        const grid = document.getElementById('circuitBreakersGrid');
        if (!grid) return;
        if (!list || list.length === 0) {
          grid.innerHTML = '<div style="color: var(--text-muted); font-size: 13px;">Nenhum circuit breaker registrado ainda. Aguardando requisições.</div>';
          return;
        }
        let allHealthy = true;
        grid.innerHTML = list.map(cb => {
          let badgeClass = 'badge-cb-closed';
          let statusText = 'CLOSED (Normal)';
          if (cb.state === 'OPEN') {
            badgeClass = 'badge-cb-open';
            statusText = 'OPEN (Aberto / Tripped)';
            allHealthy = false;
          } else if (cb.state === 'HALF_OPEN') {
            badgeClass = 'badge-cb-half';
            statusText = 'HALF_OPEN (Em Teste)';
            allHealthy = false;
          }
          return '<div class="cb-card">' +
            '<div style="display: flex; justify-content: space-between; align-items: center;">' +
              '<span class="mono" style="font-weight: 600; color: #fff; font-size: 13px;"><i class="fa-solid fa-server" style="color: #60a5fa; margin-right: 6px;"></i>' + cb.id + '</span>' +
              '<span class="badge ' + badgeClass + '" style="font-weight: 700; font-size: 10px;">' + statusText + '</span>' +
            '</div>' +
            '<div style="display: flex; justify-content: space-between; font-size: 12px; color: var(--text-muted);">' +
              '<span>Falhas Consecutivas: <strong style="color: ' + (cb.consecutive_failures > 0 ? '#f87171' : '#fff') + ';">' + cb.consecutive_failures + '</strong></span>' +
              '<span>Total Desarmes: <strong style="color: #fff;">' + (cb.total_trips || 0) + '</strong></span>' +
            '</div>' +
          '</div>';
        }).join('');
        document.getElementById('resilienceHealth').innerText = allHealthy ? '100% OK' : 'Degradado';
        document.getElementById('resilienceHealth').style.color = allHealthy ? '#34d399' : '#f87171';
      } catch (err) {}
    }

    /* --- Edge Cache Logic --- */
    async function fetchCacheStats() {
      try {
        const res = await fetch('/api/cache');
        if (!res.ok) return;
        const data = await res.json();
        document.getElementById('cacheHitRatioVal').innerText = (data.hit_ratio || 0).toFixed(1) + '%';
        document.getElementById('cacheHitsVal').innerText = (data.hits || 0).toLocaleString();
        document.getElementById('cacheMissesVal').innerText = (data.misses || 0).toLocaleString();
        document.getElementById('cachedEntriesVal').innerText = (data.cached_entries || 0).toLocaleString();
      } catch (err) {}
    }

    async function purgeCache() {
      const pattern = document.getElementById('purgePatternInput').value || '*';
      try {
        const res = await fetch('/api/cache/purge?pattern=' + encodeURIComponent(pattern), { method: 'POST' });
        if (res.ok) {
          const data = await res.json();
          showToast('Cache purgado com sucesso (' + (data.count || 0) + ' entradas removidas)!');
          fetchCacheStats();
        } else {
          showToast('Falha ao purgar cache', true);
        }
      } catch (err) {
        showToast('Erro de rede ao purgar cache', true);
      }
    }

    // Initialize map on start
    window.addEventListener('DOMContentLoaded', () => {
      initMap();
      initEventStream();
      fetchStatus();
      fetchWAF();
      fetchRoutes();
      fetchCircuitBreakers();
      fetchCacheStats();
      setInterval(fetchStatus, 2500);
      setInterval(fetchCircuitBreakers, 4000);
      setInterval(fetchCacheStats, 4000);
    });
  </script>
</body>
</html>
`
