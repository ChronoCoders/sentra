const POLL_MS = 2000;
const SPARK_POINTS = 60;

let prev = null;
let prevTs = null;

const seriesDown = [];
const seriesUp = [];

function fmtBytes(bytes) {
  if (bytes == null) return '-';
  const b = Number(bytes);
  if (!Number.isFinite(b)) return '-';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let v = b;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++; }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function fmtRateBps(bytesPerSec) {
  if (bytesPerSec == null) return '-';
  const bps = Number(bytesPerSec);
  if (!Number.isFinite(bps)) return '-';
  const bits = bps * 8;
  const units = ['bps', 'Kbps', 'Mbps', 'Gbps'];
  let v = bits;
  let i = 0;
  while (v >= 1000 && i < units.length - 1) { v /= 1000; i++; }
  return `${v.toFixed(v < 10 && i > 0 ? 2 : 1)} ${units[i]}`;
}

function fmtAgo(sec) {
  if (sec == null) return '-';
  const s = Math.max(0, Math.floor(Number(sec)));
  if (!Number.isFinite(s)) return '-';
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  const r = s % 60;

  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${r}s`;
  return `${r}s`;
}

function peerState(handshakeSec) {
  if (handshakeSec == null) return 'down';
  const s = Number(handshakeSec);
  if (!Number.isFinite(s)) return 'down';
  if (s <= 30) return 'up';
  if (s <= 120) return 'deg';
  return 'down';
}

function drawSpark(canvas, values) {
  const ctx = canvas.getContext('2d');
  const w = canvas.width, h = canvas.height;
  ctx.clearRect(0, 0, w, h);

  ctx.globalAlpha = 0.25;
  ctx.beginPath();
  for (let i = 1; i < 6; i++) {
    const y = (h / 6) * i;
    ctx.moveTo(0, y);
    ctx.lineTo(w, y);
  }
  ctx.strokeStyle = '#1f2a3a';
  ctx.stroke();
  ctx.globalAlpha = 1;

  if (!values.length) return;

  const max = Math.max(...values, 1);
  const min = Math.min(...values, 0);
  const range = Math.max(max - min, 1);

  ctx.beginPath();
  values.forEach((v, i) => {
    const x = (i / (SPARK_POINTS - 1)) * (w - 2) + 1;
    const y = h - 1 - ((v - min) / range) * (h - 2);
    if (i === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  });

  ctx.lineWidth = 2;
  ctx.strokeStyle = '#7dd3fc';
  ctx.stroke();
}

async function fetchStatus() {
  const res = await fetch('/api/status', { cache: 'no-store' });
  if (!res.ok) throw new Error(`status ${res.status}`);
  return res.json();
}

async function fetchLogs() {
  const res = await fetch('/api/logs?tail=150', { cache: 'no-store' });
  if (!res.ok) throw new Error(`logs ${res.status}`);
  return res.text();
}

function sumTotals(data) {
  let rx = 0, tx = 0;
  for (const p of (data.peers || [])) {
    rx += Number(p.rx || 0);
    tx += Number(p.tx || 0);
  }
  return { rx, tx };
}

function computeRates(curr, dtSec) {
  if (!prev || !dtSec) return { downBps: null, upBps: null, perPeer: {} };

  const currTotals = sumTotals(curr);
  const prevTotals = sumTotals(prev);

  const downBps = (currTotals.rx - prevTotals.rx) / dtSec;
  const upBps = (currTotals.tx - prevTotals.tx) / dtSec;

  const prevMap = new Map((prev.peers || []).map(p => [p.key, p]));
  const perPeer = {};
  for (const p of (curr.peers || [])) {
    const pp = prevMap.get(p.key);
    if (!pp) continue;
    perPeer[p.key] = {
      downBps: (Number(p.rx || 0) - Number(pp.rx || 0)) / dtSec,
      upBps: (Number(p.tx || 0) - Number(pp.tx || 0)) / dtSec
    };
  }

  return { downBps, upBps, perPeer };
}

function renderHeader(data, rates) {
  document.getElementById('iface').textContent = data.interface || '-';
  document.getElementById('port').textContent = data.port ?? '-';
  document.getElementById('publicIp').textContent = data.public_ip || '-';

  const totals = sumTotals(data);
  const totalRxEl = document.getElementById('totalRx');
  const totalTxEl = document.getElementById('totalTx');
  if (totalRxEl) totalRxEl.textContent = fmtBytes(totals.rx);
  if (totalTxEl) totalTxEl.textContent = fmtBytes(totals.tx);

  const statusEl = document.getElementById('status');
  const peers = data.peers || [];
  const anyUp = peers.some(p => peerState(p.handshake) !== 'down');
  const anyDeg = peers.some(p => peerState(p.handshake) === 'deg');
  statusEl.textContent = anyUp ? (anyDeg ? 'Degraded' : 'Connected') : 'Down';
  statusEl.className = 'status ' + (anyUp ? (anyDeg ? 'deg' : 'up') : 'down');

  document.getElementById('downRate').textContent = fmtRateBps(rates.downBps);
  document.getElementById('upRate').textContent = fmtRateBps(rates.upBps);

  const alerts = [];
  for (const p of peers) {
    const st = peerState(p.handshake);
    if (st === 'down') alerts.push(`Peer offline (${(p.key || '').slice(0, 10)}…)`);
    if (!p.endpoint) alerts.push(`No endpoint (${(p.key || '').slice(0, 10)}…)`);
  }
  document.getElementById('alerts').textContent = alerts.length ? alerts.join(' · ') : 'No alerts';

  if (rates.downBps != null && rates.upBps != null) {
    seriesDown.push(Math.max(0, rates.downBps));
    seriesUp.push(Math.max(0, rates.upBps));
    while (seriesDown.length > SPARK_POINTS) seriesDown.shift();
    while (seriesUp.length > SPARK_POINTS) seriesUp.shift();

    const merged = seriesDown.map((v, i) => v + (seriesUp[i] || 0));
    drawSpark(document.getElementById('rateSpark'), merged);
  }
}

function renderPeers(data, rates) {
  const tbody = document.getElementById('peers');
  tbody.innerHTML = '';

  for (const p of (data.peers || [])) {
    const st = peerState(p.handshake);
    const dotClass = st === 'up' ? 'dotUp' : st === 'deg' ? 'dotDeg' : 'dotDown';
    const r = rates.perPeer[p.key];
    const rateStr = r ? `↓ ${fmtRateBps(r.downBps)} / ↑ ${fmtRateBps(r.upBps)}` : '-';

    const endpointCell = p.endpoint
      ? `<span class="mono">${p.endpoint}</span>`
      : `<span class="badge">no-endpoint</span>`;

    const tr = document.createElement('tr');
    if (st === 'down') tr.classList.add('isDown');

    tr.innerHTML = `
      <td><span class="stateDot ${dotClass}" title="${st}"></span></td>
      <td class="mono">${(p.key || '').slice(0, 16)}…</td>
      <td>${endpointCell}</td>
      <td title="${p.handshake ?? '-'}s">${fmtAgo(p.handshake)}</td>
      <td>${rateStr}</td>
      <td>${fmtBytes(p.rx)}</td>
      <td>${fmtBytes(p.tx)}</td>
    `;
    tbody.appendChild(tr);
  }
}

function applyLogFilter(text, filter) {
  const f = (filter || '').trim().toLowerCase();
  if (!f) return text;
  return text
    .split('\n')
    .filter(line => line.toLowerCase().includes(f))
    .join('\n');
}

async function refresh() {
  const now = Date.now();
  const dtSec = prevTs ? (now - prevTs) / 1000 : null;

  try {
    const data = await fetchStatus();
    const rates = computeRates(data, dtSec);

    renderHeader(data, rates);
    renderPeers(data, rates);

    const logsRaw = await fetchLogs();
    const filter = document.getElementById('logFilter').value;
    document.getElementById('logs').textContent = applyLogFilter(logsRaw, filter);

    prev = data;
    prevTs = now;
  } catch (e) {
    console.error(e);
    const statusEl = document.getElementById('status');
    statusEl.textContent = 'API Error';
    statusEl.className = 'status down';
    document.getElementById('alerts').textContent = String(e);
  }
}

document.getElementById('restartBtn').onclick = async () => {
  try { await fetch('/api/restart', { method: 'POST' }); } catch {}
};

document.getElementById('copyLogsBtn').onclick = async () => {
  const text = document.getElementById('logs').textContent || '';
  try { await navigator.clipboard.writeText(text); } catch {}
};

setInterval(refresh, POLL_MS);
refresh();

async function fetchHealth() {
  const res = await fetch('/api/health', { cache: 'no-store' });
  if (!res.ok) throw new Error(`health ${res.status}`);
  return res.json();
}

function fmtPct(x) {
  const n = Number(x);
  if (!Number.isFinite(n)) return '-';
  return `${n.toFixed(1)}%`;
}

function fmtGB(used, total) {
  const u = Number(used), t = Number(total);
  if (!Number.isFinite(u) || !Number.isFinite(t) || t <= 0) return '-';
  return `${u.toFixed(1)}/${t.toFixed(1)} GB`;
}

function fmtMB(used, total) {
  const u = Number(used), t = Number(total);
  if (!Number.isFinite(u) || !Number.isFinite(t) || t <= 0) return '-';
  return `${u}/${t} MB`;
}

async function refreshHealth() {
  try {
    const h = await fetchHealth();
    const host = h.host || {};
    const wg = h.wg_easy || {};

    const cpuEl = document.getElementById('hCpu');
    const loadEl = document.getElementById('hLoad');
    const ramEl = document.getElementById('hRam');
    const diskEl = document.getElementById('hDisk');
    const dsEl = document.getElementById('dStatus');
    const dhEl = document.getElementById('dHealth');
    const drEl = document.getElementById('dRestarts');

    if (cpuEl) cpuEl.textContent = fmtPct(host.cpu_percent);
    if (loadEl) loadEl.textContent = `${(host.load1 ?? 0).toFixed(2)} ${(host.load5 ?? 0).toFixed(2)} ${(host.load15 ?? 0).toFixed(2)}`;
    if (ramEl) ramEl.textContent = fmtMB(host.mem_used_mb, host.mem_total_mb);
    if (diskEl) diskEl.textContent = fmtGB(host.disk_used_gb, host.disk_total_gb);

    if (dsEl) dsEl.textContent = wg.status || '-';
    if (dhEl) dhEl.textContent = wg.health || '-';
    if (drEl) drEl.textContent = (wg.restart_count ?? '-').toString();
  } catch (e) {
    // health kartı başarısızsa UI’yi bozma
  }
}

// health’i status poll ile aynı ritimde yenile
setInterval(refreshHealth, POLL_MS);
refreshHealth();

function setBadge(el, text, level) {
  if (!el) return;
  el.textContent = text;
  el.classList.remove('badgeOk','badgeWarn','badgeBad');
  if (level === 'ok') el.classList.add('badgeOk');
  if (level === 'warn') el.classList.add('badgeWarn');
  if (level === 'bad') el.classList.add('badgeBad');
}

function healthLevel(wg) {
  const status = (wg.status || '').toLowerCase();
  const health = (wg.health || '').toLowerCase();
  if (status !== 'running') return 'bad';
  if (health === 'healthy') return 'ok';
  if (health === 'starting' || health === 'none') return 'warn';
  return 'bad';
}

// override refreshHealth with better visuals
async function refreshHealth() {
  try {
    const h = await fetchHealth();
    const host = h.host || {};
    const wg = h.wg_easy || {};

    const cpuEl = document.getElementById('hCpu');
    const loadEl = document.getElementById('hLoad');
    const ramEl = document.getElementById('hRam');
    const diskEl = document.getElementById('hDisk');
    const dsEl = document.getElementById('dStatus');
    const dhEl = document.getElementById('dHealth');
    const drEl = document.getElementById('dRestarts');

    if (cpuEl) cpuEl.textContent = (Number(host.cpu_percent) === 0 ? 'measuring…' : fmtPct(host.cpu_percent));
    if (loadEl) loadEl.textContent = `${(host.load1 ?? 0).toFixed(2)} ${(host.load5 ?? 0).toFixed(2)} ${(host.load15 ?? 0).toFixed(2)}`;
    if (ramEl) ramEl.textContent = fmtMB(host.mem_used_mb, host.mem_total_mb);
    if (diskEl) diskEl.textContent = fmtGB(host.disk_used_gb, host.disk_total_gb);

    const lvl = healthLevel(wg);
    setBadge(dsEl, wg.status || '-', lvl);
    setBadge(dhEl, wg.health || '-', lvl);
    if (drEl) drEl.textContent = (wg.restart_count ?? '-').toString();
  } catch (e) {
    // ignore
  }
}

// ---------- Timeline ----------
const TL_MAX = 200;
let timeline = []; // newest last
let lastPeerState = new Map(); // key -> state

function pushEvent(e) {
  timeline.push(e);
  if (timeline.length > TL_MAX) timeline = timeline.slice(timeline.length - TL_MAX);
  renderTimeline();
}

function renderTimeline() {
  const el = document.getElementById('timeline');
  if (!el) return;

  const filter = (document.getElementById('eventFilter')?.value || '').trim().toLowerCase();
  const rows = timeline.filter(e => {
    if (!filter) return true;
    return (e.type + ' ' + e.msg + ' ' + e.level).toLowerCase().includes(filter);
  });

  el.innerHTML = rows.map(e => {
    const cls = e.level === 'error' ? 'tlRow tlErr' : e.level === 'warn' ? 'tlRow tlWarn' : 'tlRow tlInfo';
    return `<div class="${cls}">
      <div class="tlTs">${e.ts}</div>
      <div class="tlType">${e.type}</div>
      <div class="tlMsg">${escapeHtml(e.msg)}</div>
    </div>`;
  }).join('');
  el.scrollTop = el.scrollHeight;
}

function escapeHtml(s) {
  return String(s)
    .replaceAll('&','&amp;')
    .replaceAll('<','&lt;')
    .replaceAll('>','&gt;')
    .replaceAll('"','&quot;')
    .replaceAll("'","&#039;");
}

async function fetchEvents() {
  const res = await fetch('/api/events?window=300', { cache: 'no-store' });
  if (!res.ok) throw new Error(`events ${res.status}`);
  return res.json();
}

let lastBackendEventSig = new Set();
async function refreshBackendEvents() {
  try {
    const evs = await fetchEvents();
    for (const e of (evs || [])) {
      const sig = `${e.ts}|${e.level}|${e.type}|${e.msg}`;
      if (lastBackendEventSig.has(sig)) continue;
      lastBackendEventSig.add(sig);
      // keep set bounded
      if (lastBackendEventSig.size > 500) {
        lastBackendEventSig = new Set(Array.from(lastBackendEventSig).slice(-300));
      }
      pushEvent(e);
    }
  } catch {}
}

// Peer connect/disconnect based on handshake state changes
function refreshPeerEvents(statusData) {
  const peers = statusData.peers || [];
  for (const p of peers) {
    const st = peerState(p.handshake); // up/deg/down
    const prevSt = lastPeerState.get(p.key);
    lastPeerState.set(p.key, st);

    if (!prevSt) continue;

    if (prevSt === 'down' && st !== 'down') {
      pushEvent({
        ts: new Date().toISOString(),
        level: 'info',
        type: 'peer',
        msg: `CONNECTED ${p.key.slice(0, 10)}… (${p.endpoint || 'no-endpoint'})`
      });
    }

    if (prevSt !== 'down' && st === 'down') {
      pushEvent({
        ts: new Date().toISOString(),
        level: 'warn',
        type: 'peer',
        msg: `DISCONNECTED ${p.key.slice(0, 10)}…`
      });
    }

    // endpoint change highlight
    const prevSnap = prev?.peers?.find(x => x.key === p.key);
    if (prevSnap && prevSnap.endpoint && p.endpoint && prevSnap.endpoint !== p.endpoint) {
      pushEvent({
        ts: new Date().toISOString(),
        level: 'info',
        type: 'peer',
        msg: `ENDPOINT CHANGED ${p.key.slice(0, 10)}… ${prevSnap.endpoint} → ${p.endpoint}`
      });
    }
  }
}

document.getElementById('clearTimelineBtn')?.addEventListener('click', () => {
  timeline = [];
  renderTimeline();
});

// Hook peer events into existing refresh() by wrapping it
const _oldRefresh = refresh;
refresh = async function() {
  const beforePrev = prev;
  await _oldRefresh();
  // after refresh, prev is updated to latest
  if (prev && beforePrev) {
    refreshPeerEvents(prev);
  } else if (prev && !beforePrev) {
    // initialize peer state map on first data
    for (const p of (prev.peers || [])) lastPeerState.set(p.key, peerState(p.handshake));
  }
};

// backend event poll
setInterval(refreshBackendEvents, 4000);
refreshBackendEvents();


// ---------- Timeline ----------
(() => {
  if (window.__timelineInit) return;
  window.__timelineInit = true;

  const TL_MAX = 200;
  let timeline = [];
  let lastPeerState = new Map();

  function pushEvent(e) {
    timeline.push(e);
    if (timeline.length > TL_MAX) timeline = timeline.slice(timeline.length - TL_MAX);
    renderTimeline();
  }

  function escapeHtml(s) {
    return String(s)
      .replaceAll('&','&amp;')
      .replaceAll('<','&lt;')
      .replaceAll('>','&gt;')
      .replaceAll('"','&quot;')
      .replaceAll("'","&#039;");
  }

  function renderTimeline() {
    const el = document.getElementById('timeline');
    if (!el) return;

    const filter = (document.getElementById('eventFilter')?.value || '').trim().toLowerCase();
    const rows = timeline.filter(e => {
      if (!filter) return true;
      return (e.type + ' ' + e.msg + ' ' + e.level).toLowerCase().includes(filter);
    });

    el.innerHTML = rows.map(e => {
      const cls = e.level === 'error' ? 'tlRow tlErr' : e.level === 'warn' ? 'tlRow tlWarn' : 'tlRow tlInfo';
      return `<div class="${cls}">
        <div class="tlTs">${e.ts}</div>
        <div class="tlType">${e.type}</div>
        <div class="tlMsg">${escapeHtml(e.msg)}</div>
      </div>`;
    }).join('');
    el.scrollTop = el.scrollHeight;
  }

  async function fetchEvents() {
    const res = await fetch('/api/events?window=300', { cache: 'no-store' });
    if (!res.ok) throw new Error(`events ${res.status}`);
    return res.json();
  }

  let lastBackendEventSig = new Set();
  async function refreshBackendEvents() {
    try {
      const evs = await fetchEvents();
      for (const e of (evs || [])) {
        const sig = `${e.ts}|${e.level}|${e.type}|${e.msg}`;
        if (lastBackendEventSig.has(sig)) continue;
        lastBackendEventSig.add(sig);
        if (lastBackendEventSig.size > 500) {
          lastBackendEventSig = new Set(Array.from(lastBackendEventSig).slice(-300));
        }
        pushEvent(e);
      }
    } catch {}
  }

  function refreshPeerEvents(currStatus) {
    const peers = currStatus.peers || [];
    for (const p of peers) {
      const st = (typeof peerState === 'function') ? peerState(p.handshake) : 'unknown';
      const prevSt = lastPeerState.get(p.key);
      lastPeerState.set(p.key, st);

      if (!prevSt) continue;

      if (prevSt === 'down' && st !== 'down') {
        pushEvent({
          ts: new Date().toISOString(),
          level: 'info',
          type: 'peer',
          msg: `CONNECTED ${(p.key || '').slice(0, 10)}… (${p.endpoint || 'no-endpoint'})`
        });
      }

      if (prevSt !== 'down' && st === 'down') {
        pushEvent({
          ts: new Date().toISOString(),
          level: 'warn',
          type: 'peer',
          msg: `DISCONNECTED ${(p.key || '').slice(0, 10)}…`
        });
      }
    }
  }

  document.getElementById('clearTimelineBtn')?.addEventListener('click', () => {
    timeline = [];
    renderTimeline();
  });

  // refresh() hook
  if (typeof refresh === 'function') {
    const __origRefresh = refresh;
    refresh = async function() {
      const hadPrev = (typeof prev !== 'undefined') && !!prev;
      await __origRefresh();
      if (typeof prev !== 'undefined' && prev && !hadPrev) {
        for (const p of (prev.peers || [])) lastPeerState.set(p.key, (typeof peerState === 'function') ? peerState(p.handshake) : 'unknown');
      } else if (typeof prev !== 'undefined' && prev) {
        refreshPeerEvents(prev);
      }
    };
  }

  setInterval(refreshBackendEvents, 4000);
  refreshBackendEvents();
})();
