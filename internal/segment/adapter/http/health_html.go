package http

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/selseleha/go-challenge/internal/segment"
)

var healthPage = template.Must(template.New("health").Funcs(template.FuncMap{
	"readyClass": func(status string) string {
		if status == segment.StatusReady {
			return "ok"
		}
		return "warn"
	},
	"boolClass": func(ok bool) string {
		if ok {
			return "ok"
		}
		return "bad"
	},
	"fmtDur": func(d time.Duration) string {
		d = d.Round(time.Second)
		if d < time.Minute {
			return d.String()
		}
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	},
	"fmtFloat": func(v float64, prec int) string {
		return fmt.Sprintf("%.*f", prec, v)
	},
	"fmtInt": func(v int64) string {
		return fmt.Sprintf("%d", v)
	},
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>ES Health</title>
<style>
  :root {
    --bg: #0f1419;
    --panel: #1a222c;
    --text: #e7ecf1;
    --muted: #8b9aab;
    --ok: #3dd68c;
    --warn: #f5a524;
    --bad: #f31260;
    --line: #2a3542;
    --accent: #6ea8fe;
    --pg: #7dd3fc;
    --redis: #f87171;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    min-height: 100vh;
    font-family: "IBM Plex Sans", "Segoe UI", sans-serif;
    background:
      radial-gradient(1200px 600px at 10% -10%, #1b2a3d 0%, transparent 55%),
      radial-gradient(900px 500px at 100% 0%, #14241f 0%, transparent 50%),
      var(--bg);
    color: var(--text);
  }
  main {
    max-width: 1100px;
    margin: 0 auto;
    padding: 2.5rem 1.25rem 3rem;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: 1rem;
    margin-bottom: 1.75rem;
  }
  h1 {
    margin: 0;
    font-size: 1.75rem;
    letter-spacing: -0.03em;
  }
  .sub { color: var(--muted); margin-top: 0.35rem; font-size: 0.95rem; }
  .badge {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    padding: 0.45rem 0.8rem;
    border-radius: 999px;
    font-weight: 600;
    font-size: 0.85rem;
    border: 1px solid var(--line);
    background: var(--panel);
  }
  .badge .dot {
    width: 0.55rem; height: 0.55rem; border-radius: 50%;
    background: var(--muted);
  }
  .badge.ok .dot { background: var(--ok); box-shadow: 0 0 10px var(--ok); }
  .badge.warn .dot { background: var(--warn); box-shadow: 0 0 10px var(--warn); }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 1rem;
  }
  .top4 {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 1rem;
    margin-bottom: 1rem;
  }
  @media (max-width: 1100px) {
    .top4 { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  }
  .card {
    background: color-mix(in srgb, var(--panel) 92%, black);
    border: 1px solid var(--line);
    border-radius: 14px;
    padding: 1.1rem 1.15rem 1.2rem;
  }
  .card.wide { grid-column: 1 / -1; }
  .card h2 {
    margin: 0 0 0.85rem;
    font-size: 0.78rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--muted);
    font-weight: 600;
  }
  .metric {
    font-size: 1.65rem;
    font-weight: 700;
    letter-spacing: -0.03em;
    font-variant-numeric: tabular-nums;
  }
  .metric small {
    display: block;
    margin-top: 0.25rem;
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--muted);
  }
  .row {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.45rem 0;
    border-top: 1px solid var(--line);
    font-size: 0.92rem;
  }
  .row:first-of-type { border-top: 0; padding-top: 0; }
  .k { color: var(--muted); }
  .v { font-variant-numeric: tabular-nums; }
  .ok { color: var(--ok); }
  .warn { color: var(--warn); }
  .bad { color: var(--bad); }
  canvas { width: 100%; height: 280px; display: block; }
  .hint { color: var(--muted); font-size: 0.8rem; margin-top: 0.6rem; }
  footer {
    margin-top: 1.5rem;
    color: var(--muted);
    font-size: 0.8rem;
  }
  code { color: var(--accent); }
</style>
</head>
<body>
<main>
  <header>
    <div>
      <h1>Estimation Service</h1>
      <div class="sub">Live monitor · store <code id="store">{{.Store}}</code> · refresh every 3s</div>
    </div>
    <div id="badge" class="badge {{readyClass .Status}}">
      <span class="dot"></span><span id="statusText">{{.Status}}</span>
    </div>
  </header>

  <div class="top4">
    <section class="card">
      <h2>Postgres (source)</h2>
      <div class="metric" style="color:var(--pg)"><span id="pgCount">{{if .Hot.HasSource}}{{fmtInt .Hot.SourceMembers}}{{else}}—{{end}}</span>
        <small id="pgLabel">{{if .Hot.HasSource}}memberships rows{{else}}not used for this store{{end}}</small>
      </div>
    </section>

    <section class="card">
      <h2>Redis / hot store</h2>
      <div class="metric" style="color:var(--redis)"><span id="redisCount">{{fmtInt .Hot.Members}}</span>
        <small id="redisLabel">members in hot store</small>
      </div>
      <div class="row"><span class="k">Segments</span><span class="v" id="segCount">{{.Hot.Segments}}</span></div>
      <div class="row"><span class="k">Lag (PG − hot)</span><span class="v" id="lag">—</span></div>
    </section>

    <section class="card">
      <h2>Runtime</h2>
      <div class="metric"><span id="probe">{{fmtFloat .ProbeMS 2}} ms</span><small>estimate probe</small></div>
      <div class="row"><span class="k">Uptime</span><span class="v" id="uptime">{{fmtDur .Uptime}}</span></div>
      <div class="row"><span class="k">Goroutines</span><span class="v" id="goroutines">{{.GoRoutines}}</span></div>
      <div class="row"><span class="k">Heap alloc</span><span class="v" id="alloc">{{fmtFloat .AllocMiB 1}} MiB</span></div>
    </section>

    <section class="card">
      <h2>Product config</h2>
      <div class="metric" style="color:var(--accent)"><span id="cfgStore">{{.Product.Store}}</span>
        <small>active store mode</small>
      </div>
      <div class="row"><span class="k">Membership TTL</span><span class="v" id="cfgTTL">{{.Product.MembershipTTL}}</span></div>
      <div class="row"><span class="k">Outbox batch</span><span class="v" id="cfgOutboxBatch">{{.Product.OutboxBatchSize}}</span></div>
      <div class="row"><span class="k">Outbox poll</span><span class="v" id="cfgOutboxPoll">{{.Product.OutboxPollInterval}}</span></div>
      <div class="row"><span class="k">Warm page</span><span class="v" id="cfgWarmPage">{{.Product.WarmPageSize}}</span></div>
      <div class="row"><span class="k">Outbox keep</span><span class="v" id="cfgCleanup">{{.Product.CleanupRetention}}</span></div>
    </section>
  </div>

  <div class="grid">
    <section class="card wide">
      <h2>Segment estimates (live API)</h2>
      <canvas id="bars" width="1000" height="280"></canvas>
      <div class="hint" id="chartHint">Fetching estimates for each known segment…</div>
    </section>

    <section class="card wide">
      <h2>Totals over time</h2>
      <canvas id="lines" width="1000" height="220"></canvas>
      <div class="hint">Postgres source vs Redis/hot members across refreshes</div>
    </section>
  </div>

  <footer>
    JSON: <code>GET /v1/health?format=json</code> · estimates: <code>GET /v1/segments/{name}/estimate</code>
  </footer>
</main>
<script>
(function () {
  const HISTORY = 40;
  const history = { t: [], pg: [], hot: [] };
  const colors = ["#6ea8fe","#3dd68c","#f5a524","#f87171","#c084fc","#22d3ee","#fb7185","#a3e635"];

  function fmtInt(n) {
    if (n == null || Number.isNaN(n)) return "—";
    return new Intl.NumberFormat().format(n);
  }
  function fmtDur(sec) {
    sec = Math.floor(sec || 0);
    if (sec < 60) return sec + "s";
    return Math.floor(sec / 60) + "m" + (sec % 60) + "s";
  }

  function drawBars(canvas, items) {
    const ctx = canvas.getContext("2d");
    const w = canvas.width, h = canvas.height;
    ctx.clearRect(0, 0, w, h);
    if (!items.length) {
      ctx.fillStyle = "#8b9aab";
      ctx.font = "14px sans-serif";
      ctx.fillText("No segments yet", 16, 32);
      return;
    }
    const max = Math.max(1, ...items.map(i => i.count));
    const pad = 40, gap = 10;
    const barW = Math.max(12, (w - pad * 2 - gap * (items.length - 1)) / items.length);
    items.forEach((it, i) => {
      const x = pad + i * (barW + gap);
      const bh = ((h - 60) * it.count) / max;
      const y = h - 40 - bh;
      ctx.fillStyle = colors[i % colors.length];
      ctx.fillRect(x, y, barW, bh);
      ctx.fillStyle = "#e7ecf1";
      ctx.font = "12px sans-serif";
      ctx.textAlign = "center";
      ctx.fillText(fmtInt(it.count), x + barW / 2, y - 6);
      ctx.fillStyle = "#8b9aab";
      const label = it.segment.length > 10 ? it.segment.slice(0, 9) + "…" : it.segment;
      ctx.fillText(label, x + barW / 2, h - 18);
    });
  }

  function drawLines(canvas) {
    const ctx = canvas.getContext("2d");
    const w = canvas.width, h = canvas.height;
    ctx.clearRect(0, 0, w, h);
    const n = history.t.length;
    if (n < 2) {
      ctx.fillStyle = "#8b9aab";
      ctx.font = "14px sans-serif";
      ctx.fillText("Collecting samples…", 16, 32);
      return;
    }
    const vals = history.pg.concat(history.hot).filter(v => v != null);
    const max = Math.max(1, ...vals);
    const pad = 36;
    function yOf(v) { return h - pad - ((h - pad * 2) * v) / max; }
    function xOf(i) { return pad + ((w - pad * 2) * i) / (n - 1); }
    function stroke(series, color) {
      ctx.beginPath();
      ctx.strokeStyle = color;
      ctx.lineWidth = 2;
      let started = false;
      series.forEach((v, i) => {
        if (v == null) return;
        const x = xOf(i), y = yOf(v);
        if (!started) { ctx.moveTo(x, y); started = true; }
        else ctx.lineTo(x, y);
      });
      ctx.stroke();
    }
    stroke(history.pg, "#7dd3fc");
    stroke(history.hot, "#f87171");
    ctx.fillStyle = "#8b9aab";
    ctx.font = "12px sans-serif";
    ctx.fillText("postgres", 16, 18);
    ctx.fillStyle = "#7dd3fc";
    ctx.fillRect(70, 10, 12, 4);
    ctx.fillStyle = "#8b9aab";
    ctx.fillText("hot/redis", 100, 18);
    ctx.fillStyle = "#f87171";
    ctx.fillRect(165, 10, 12, 4);
  }

  async function fetchEstimates(names) {
    const results = await Promise.all(names.map(async (segment) => {
      try {
        const res = await fetch("/v1/segments/" + encodeURIComponent(segment) + "/estimate", {
          headers: { Accept: "application/json" }
        });
        if (!res.ok) return { segment, count: 0, err: true };
        const body = await res.json();
        return { segment, count: body.count || 0 };
      } catch (e) {
        return { segment, count: 0, err: true };
      }
    }));
    results.sort((a, b) => b.count - a.count);
    return results;
  }

  async function tick() {
    try {
      const res = await fetch("/v1/health?format=json", { headers: { Accept: "application/json" } });
      const h = await res.json();
      const hot = h.hot_store || {};
      document.getElementById("store").textContent = h.store || "";
      document.getElementById("statusText").textContent = h.status || "";
      const badge = document.getElementById("badge");
      badge.className = "badge " + (h.status === "ready" ? "ok" : "warn");

      const hasSource = !!hot.has_source;
      document.getElementById("pgCount").textContent = hasSource ? fmtInt(hot.source_members) : "—";
      document.getElementById("pgLabel").textContent = hasSource ? "memberships rows" : "not used for this store";
      document.getElementById("redisCount").textContent = fmtInt(hot.members);
      document.getElementById("segCount").textContent = fmtInt(hot.segments);
      document.getElementById("lag").textContent = hasSource
        ? fmtInt((hot.source_members || 0) - (hot.members || 0))
        : "—";
      document.getElementById("probe").textContent = (h.probe_ms || 0).toFixed(2) + " ms";
      document.getElementById("uptime").textContent = fmtDur(h.uptime_sec);
      document.getElementById("goroutines").textContent = h.goroutines;
      document.getElementById("alloc").textContent = (h.alloc_mib || 0).toFixed(1) + " MiB";

      const p = h.product || {};
      document.getElementById("cfgStore").textContent = p.store || h.store || "—";
      document.getElementById("cfgTTL").textContent = p.membership_ttl || "—";
      document.getElementById("cfgOutboxBatch").textContent = p.outbox_batch_size != null ? p.outbox_batch_size : "—";
      document.getElementById("cfgOutboxPoll").textContent = p.outbox_poll_interval || "—";
      document.getElementById("cfgWarmPage").textContent = p.warm_page_size != null ? p.warm_page_size : "—";
      document.getElementById("cfgCleanup").textContent = p.cleanup_retention || "—";

      history.t.push(Date.now());
      history.pg.push(hasSource ? (hot.source_members || 0) : null);
      history.hot.push(hot.members || 0);
      if (history.t.length > HISTORY) {
        history.t.shift(); history.pg.shift(); history.hot.shift();
      }
      drawLines(document.getElementById("lines"));

      const names = hot.segment_names || [];
      document.getElementById("chartHint").textContent =
        names.length
          ? ("Calling estimate for " + names.length + " segment(s)…")
          : "No segments discovered yet";
      const items = await fetchEstimates(names);
      drawBars(document.getElementById("bars"), items);
      const ok = items.filter(i => !i.err).length;
      document.getElementById("chartHint").textContent =
        names.length ? ("Updated " + ok + "/" + names.length + " segment estimates") : "No segments discovered yet";
    } catch (e) {
      document.getElementById("chartHint").textContent = "Refresh failed: " + e;
    }
  }

  tick();
  setInterval(tick, 3000);
})();
</script>
</body>
</html>`))

func wantsHTML(r *http.Request) bool {
	if r.URL.Query().Get("format") == "json" {
		return false
	}
	if r.URL.Query().Get("format") == "html" {
		return true
	}
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}
	// Browsers send text/html first; API clients usually ask for json or */*.
	return strings.Contains(accept, "text/html")
}

func writeHealthHTML(w http.ResponseWriter, status int, info segment.HealthInfo) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = healthPage.Execute(w, info)
}
