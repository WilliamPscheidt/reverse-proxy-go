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
  <title>Admin Dashboard | Reverse Proxy & Load Balancer</title>
  <style>
    :root {
      --bg: #0b0f19;
      --card-bg: #131b2e;
      --card-border: #1e293b;
      --accent: #38bdf8;
      --accent-hover: #0284c7;
      --green: #10b981;
      --red: #ef4444;
      --yellow: #f59e0b;
      --text-main: #f8fafc;
      --text-muted: #94a3b8;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; }
    body { background-color: var(--bg); color: var(--text-main); min-height: 100vh; padding: 24px; }
    .container { max-width: 1200px; margin: 0 auto; }
    
    /* Header */
    header { display: flex; justify-content: space-between; align-items: center; padding-bottom: 24px; border-bottom: 1px solid var(--card-border); margin-bottom: 24px; flex-wrap: wrap; gap: 16px; }
    .brand { display: flex; align-items: center; gap: 12px; }
    .brand h1 { font-size: 22px; font-weight: 700; color: #fff; letter-spacing: -0.5px; }
    .badge { padding: 4px 10px; border-radius: 9999px; font-size: 12px; font-weight: 600; text-transform: uppercase; }
    .badge-blue { background: rgba(56, 189, 248, 0.15); color: var(--accent); border: 1px solid rgba(56, 189, 248, 0.3); }
    .badge-green { background: rgba(16, 185, 129, 0.15); color: var(--green); border: 1px solid rgba(16, 185, 129, 0.3); }
    .badge-red { background: rgba(239, 68, 68, 0.15); color: var(--red); border: 1px solid rgba(239, 68, 68, 0.3); }
    
    .nav-actions { display: flex; align-items: center; gap: 12px; }
    .btn { background: var(--accent); color: #0b0f19; padding: 8px 16px; border-radius: 6px; font-weight: 600; font-size: 13px; border: none; cursor: pointer; text-decoration: none; display: inline-flex; align-items: center; gap: 6px; transition: all 0.2s; }
    .btn:hover { background: var(--accent-hover); color: #fff; }
    .btn-secondary { background: var(--card-border); color: var(--text-main); }
    .btn-secondary:hover { background: #334155; }
    .btn-danger { background: rgba(239, 68, 68, 0.2); color: var(--red); border: 1px solid var(--red); }
    .btn-danger:hover { background: var(--red); color: #fff; }
    .btn-sm { padding: 4px 8px; font-size: 11px; }

    /* Stats Grid */
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; margin-bottom: 24px; }
    .card { background: var(--card-bg); border: 1px solid var(--card-border); border-radius: 10px; padding: 18px; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1); }
    .card-title { font-size: 12px; font-weight: 600; text-transform: uppercase; color: var(--text-muted); margin-bottom: 8px; }
    .card-value { font-size: 26px; font-weight: 700; color: #fff; }
    .card-sub { font-size: 12px; color: var(--text-muted); margin-top: 4px; }

    /* Control Panel */
    .section-title { font-size: 18px; font-weight: 600; margin-bottom: 14px; display: flex; align-items: center; justify-content: space-between; }
    .panel { background: var(--card-bg); border: 1px solid var(--card-border); border-radius: 10px; padding: 20px; margin-bottom: 24px; }

    /* Algorithm Switcher */
    .algo-container { display: flex; gap: 12px; flex-wrap: wrap; margin-bottom: 12px; }
    .algo-btn { flex: 1; min-width: 180px; padding: 12px; border: 1px solid var(--card-border); background: #0f172a; border-radius: 8px; cursor: pointer; text-align: left; transition: all 0.2s; }
    .algo-btn:hover { border-color: var(--accent); }
    .algo-btn.active { border-color: var(--accent); background: rgba(56, 189, 248, 0.1); }
    .algo-btn strong { display: block; font-size: 14px; color: #fff; margin-bottom: 4px; }
    .algo-btn span { font-size: 11px; color: var(--text-muted); }

    /* Tables */
    table { width: 100%; border-collapse: collapse; text-align: left; font-size: 13px; }
    th { padding: 12px 14px; background: #0f172a; color: var(--text-muted); font-weight: 600; border-bottom: 1px solid var(--card-border); }
    td { padding: 12px 14px; border-bottom: 1px solid var(--card-border); vertical-align: middle; }
    tr:hover td { background: rgba(255, 255, 255, 0.02); }

    /* Form */
    .form-row { display: grid; grid-template-columns: 140px 1fr 100px auto; gap: 12px; align-items: center; margin-top: 14px; }
    input, select { background: #0f172a; border: 1px solid var(--card-border); color: #fff; padding: 9px 12px; border-radius: 6px; font-size: 13px; width: 100%; outline: none; }
    input:focus, select:focus { border-color: var(--accent); }

    /* Toast */
    #toast { position: fixed; bottom: 24px; right: 24px; background: #1e293b; border: 1px solid var(--card-border); color: #fff; padding: 12px 20px; border-radius: 8px; font-size: 13px; display: none; box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.3); z-index: 1000; }
  </style>
</head>
<body>
  <div class="container">
    <!-- Header -->
    <header>
      <div class="brand">
        <span style="font-size: 26px;">⚡</span>
        <div>
          <h1>Reverse Proxy & Load Balancer</h1>
          <span style="font-size: 12px; color: var(--text-muted);">Painel de Administração e Controle de Tráfego</span>
        </div>
      </div>
      <div class="nav-actions">
        <span id="uptimeBadge" class="badge badge-blue">Uptime: --</span>
        <a href="http://localhost:2112/metrics" target="_blank" class="btn btn-secondary">📊 Prometheus /metrics</a>
        <button onclick="fetchStatus()" class="btn">🔄 Atualizar</button>
      </div>
    </header>

    <!-- Stat Cards -->
    <div class="grid">
      <div class="card">
        <div class="card-title">Backends L7 (HTTP)</div>
        <div class="card-value" id="l7Nodes">-- / --</div>
        <div class="card-sub" id="l7Reqs">0 requisições atendidas</div>
      </div>
      <div class="card">
        <div class="card-title">Backends L4 (TCP)</div>
        <div class="card-value" id="l4Nodes">-- / --</div>
        <div class="card-sub" id="l4Reqs">0 conexões ativas</div>
      </div>
      <div class="card">
        <div class="card-title">Algoritmo Ativo</div>
        <div class="card-value" id="currentAlgo" style="color: var(--accent); font-size: 20px;">--</div>
        <div class="card-sub">Hot-swap em tempo real</div>
      </div>
      <div class="card">
        <div class="card-title">Rate Limiting</div>
        <div class="card-value" id="rateLimitVal" style="color: var(--green); font-size: 20px;">--</div>
        <div class="card-sub">Sharded Token Bucket in-memory</div>
      </div>
    </div>

    <!-- Algorithm Selector Panel -->
    <div class="panel">
      <div class="section-title">
        <span>⚖️ Algoritmo de Balanceamento (Alternância em Tempo Real)</span>
      </div>
      <div class="algo-container">
        <div class="algo-btn" id="btn-round_robin" onclick="setAlgorithm('round_robin')">
          <strong>Round-Robin (Atômico)</strong>
          <span>Alternância sequencial lock-free (~35ns/op). Ideal para nós homogêneos.</span>
        </div>
        <div class="algo-btn" id="btn-least_conn" onclick="setAlgorithm('least_conn')">
          <strong>Least Connections</strong>
          <span>Roteia para o nó com menor número de conexões ativas em tempo real.</span>
        </div>
        <div class="algo-btn" id="btn-ip_hash" onclick="setAlgorithm('ip_hash')">
          <strong>IP Hash (Sticky Session)</strong>
          <span>Mapeamento determinístico de IP para preservar sessões de clientes.</span>
        </div>
      </div>
    </div>

    <!-- Backends Table Panel -->
    <div class="panel">
      <div class="section-title">
        <span>🖥️ Servidores de Destino Configurados (Upstream Backends)</span>
        <span style="font-size: 12px; color: var(--text-muted);" id="nodesCount">Total: 0</span>
      </div>
      <table>
        <thead>
          <tr>
            <th>Camada</th>
            <th>Endereço / URL de Destino</th>
            <th>Peso</th>
            <th>Status</th>
            <th>Conexões Ativas</th>
            <th>Total Requisições</th>
            <th>Falhas / Sucessos</th>
            <th style="text-align: right;">Ações</th>
          </tr>
        </thead>
        <tbody id="backendsTable">
          <tr><td colspan="8" style="text-align: center; color: var(--text-muted); padding: 24px;">Carregando destinos...</td></tr>
        </tbody>
      </table>

      <!-- Add New Backend Form -->
      <div style="margin-top: 24px; padding-top: 18px; border-top: 1px solid var(--card-border);">
        <strong style="font-size: 14px; display: block; margin-bottom: 8px;">➕ Adicionar Novo Destino de Proxy</strong>
        <form onsubmit="addBackend(event)" class="form-row">
          <select id="newType" required>
            <option value="l7">L7 (HTTP)</option>
            <option value="l4">L4 (TCP)</option>
          </select>
          <input type="text" id="newTarget" placeholder="Ex: http://127.0.0.1:8084 (L7) ou 127.0.0.1:9084 (L4)" required />
          <input type="number" id="newWeight" placeholder="Peso" value="1" min="1" max="100" required />
          <button type="submit" class="btn">Adicionar Servidor</button>
        </form>
      </div>
    </div>

    <!-- Health Check & HTTP Details Panel -->
    <div class="panel">
      <div class="section-title">
        <span>⚙️ Configuração do Health-Check & Parâmetros HTTP</span>
      </div>
      <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 16px; font-size: 13px;">
        <div style="background: #0f172a; padding: 14px; border-radius: 8px; border: 1px solid var(--card-border);">
          <strong style="color: var(--accent); display: block; margin-bottom: 6px;">💓 Health Check Ativo</strong>
          <div>Intervalo: <span id="hcInterval" style="color: #fff;">--</span></div>
          <div>Timeout por ping: <span id="hcTimeout" style="color: #fff;">--</span></div>
          <div>Endpoint verificado: <span id="hcPath" style="color: #fff;">--</span></div>
          <div>Thresholds: <span id="hcThresholds" style="color: #fff;">--</span></div>
        </div>
        <div style="background: #0f172a; padding: 14px; border-radius: 8px; border: 1px solid var(--card-border);">
          <strong style="color: var(--accent); display: block; margin-bottom: 6px;">🛡️ Cabeçalhos & Conexões</strong>
          <div>Injeção RFC: <code>X-Forwarded-For</code>, <code>X-Real-IP</code></div>
          <div>Keep-Alive: <code>MaxIdleConns: 10000</code></div>
          <div>Buffer Pooling L4: <code>32 KB (sync.Pool)</code></div>
          <div>Status Health: Detecção e dreno automático</div>
        </div>
      </div>
    </div>
  </div>

  <div id="toast"></div>

  <script>
    function showToast(msg, isError = false) {
      const t = document.getElementById('toast');
      t.innerText = msg;
      t.style.borderColor = isError ? 'var(--red)' : 'var(--green)';
      t.style.display = 'block';
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
          document.getElementById('l7Reqs').innerText = data.l7.total_requests.toLocaleString() + ' reqs (' + data.l7.active_conns + ' conns ativas)';
          document.getElementById('currentAlgo').innerText = data.l7.algorithm.toUpperCase();
          highlightAlgoBtn(data.l7.algorithm);
        }

        if (data.l4) {
          document.getElementById('l4Nodes').innerText = data.l4.healthy_nodes + ' / ' + data.l4.total_nodes;
          document.getElementById('l4Reqs').innerText = data.l4.active_conns + ' conexões TCP ativas';
        }

        if (data.rate_limit) {
          document.getElementById('rateLimitVal').innerText = data.rate_limit.enabled ? (data.rate_limit.rate.toLocaleString() + ' req/s') : 'Desativado';
        }

        if (data.health_check) {
          document.getElementById('hcInterval').innerText = (data.health_check.interval / 1e9) + 's';
          document.getElementById('hcTimeout').innerText = (data.health_check.timeout / 1e9) + 's';
          document.getElementById('hcPath').innerText = data.health_check.path;
          document.getElementById('hcThresholds').innerText = 'Falha: ' + data.health_check.unhealthy_threshold + 'x | Recupera: ' + data.health_check.healthy_threshold + 'x';
        }

        await fetchBackends();
      } catch (err) {
        console.error('Fetch status error:', err);
      }
    }

    function highlightAlgoBtn(algo) {
      document.querySelectorAll('.algo-btn').forEach(b => b.classList.remove('active'));
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
          showToast('Algoritmo alterado para: ' + algo.toUpperCase());
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
            ? '<span class="badge badge-green">🟢 Saudável</span>' 
            : '<span class="badge badge-red">🔴 Inoperante</span>';

          const toggleAction = b.alive 
            ? '<button onclick="toggleBackend(\'' + b.type + '\', \'' + b.target + '\', false)" class="btn btn-secondary btn-sm">Drenar</button>'
            : '<button onclick="toggleBackend(\'' + b.type + '\', \'' + b.target + '\', true)" class="btn btn-secondary btn-sm">Reativar</button>';

          html += '<tr>' +
            '<td><span class="badge badge-blue">' + b.type.toUpperCase() + '</span></td>' +
            '<td><strong style="color: #fff;">' + b.target + '</strong></td>' +
            '<td>' + b.weight + '</td>' +
            '<td>' + statusBadge + '</td>' +
            '<td>' + b.active_conns + '</td>' +
            '<td>' + b.total_requests.toLocaleString() + '</td>' +
            '<td>' + b.consecutive_failures + ' falhas / ' + b.consecutive_successes + ' ok</td>' +
            '<td style="text-align: right; display: flex; gap: 6px; justify-content: flex-end;">' +
              toggleAction +
              '<button onclick="removeBackend(\'' + b.type + '\', \'' + b.target + '\')" class="btn btn-danger btn-sm">Remover</button>' +
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
          showToast('Servidor ' + target + ' adicionado com sucesso!');
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
      if (!confirm('Deseja realmente remover o backend ' + target + '?')) return;
      try {
        const res = await fetch('/api/backends', {
          method: 'DELETE',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ type, target })
        });
        if (res.ok) {
          showToast('Servidor ' + target + ' removido!');
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
          showToast('Status do servidor ' + target + ' atualizado!');
          fetchStatus();
        } else {
          showToast('Falha ao atualizar status', true);
        }
      } catch (err) {
        showToast('Erro ao atualizar status', true);
      }
    }

    // Initial load and periodic polling every 2.5s
    fetchStatus();
    setInterval(fetchStatus, 2500);
  </script>
</body>
</html>
`
