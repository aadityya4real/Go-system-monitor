const state = {
  timer: null,
  history: [],
};

const el = {
  timestamp: document.querySelector("#timestamp"),
  refresh: document.querySelector("#refresh"),
  cpuUsed: document.querySelector("#cpu-used"),
  cpuCores: document.querySelector("#cpu-cores"),
  memoryUsed: document.querySelector("#memory-used"),
  memoryFree: document.querySelector("#memory-free"),
  diskUsed: document.querySelector("#disk-used"),
  diskFree: document.querySelector("#disk-free"),
  duration: document.querySelector("#duration"),
  status: document.querySelector("#status"),
  warning: document.querySelector("#warning"),
  disks: document.querySelector("#disks"),
  metrics: document.querySelector("#metrics"),
  trend: document.querySelector("#trend"),
};

el.refresh.addEventListener("change", schedule);
window.addEventListener("resize", drawTrend);

schedule();
load();

function schedule() {
  if (state.timer) {
    clearInterval(state.timer);
  }
  state.timer = setInterval(load, Number(el.refresh.value));
}

async function load() {
  try {
    const response = await fetch("/api/stats", { cache: "no-store" });
    const data = await response.json();
    render(data);
  } catch (error) {
    el.status.textContent = "offline";
    el.warning.textContent = error.message;
  }
}

function render(data) {
  const metrics = data.metrics || [];
  const byName = new Map(metrics.map((metric) => [metricKey(metric), metric]));
  const cpu = value(byName, "gosysmon_cpu_used_ratio");
  const memory = value(byName, "gosysmon_memory_used_ratio");
  const cores = value(byName, "gosysmon_cpu_logical_cores");
  const diskRows = disks(metrics);
  const primaryDisk = diskRows[0];

  el.timestamp.textContent = `Collected ${new Date(data.collected_at).toLocaleString()}`;
  el.cpuUsed.textContent = cpu == null ? "--" : percent(cpu);
  el.cpuCores.textContent = cores == null ? "-- cores" : `${cores} cores`;
  el.memoryUsed.textContent = memory == null ? "--" : percent(memory);
  el.memoryFree.textContent = bytes(value(byName, "gosysmon_memory_available_bytes"));
  el.diskUsed.textContent = primaryDisk ? percent(primaryDisk.used) : "--";
  el.diskFree.textContent = primaryDisk ? `${bytes(primaryDisk.free)} free` : "-- free";
  el.duration.textContent = `${data.duration_ms} ms`;
  el.status.textContent = data.warning ? "partial" : "healthy";
  el.warning.textContent = data.warning || "";

  state.history.push({
    cpu,
    memory,
    disk: primaryDisk ? primaryDisk.used : null,
  });
  state.history = state.history.slice(-48);

  renderDisks(diskRows);
  renderMetrics(metrics);
  drawTrend();
}

function metricKey(metric) {
  const labels = metric.labels || {};
  const suffix = Object.keys(labels)
    .sort()
    .map((key) => `${key}=${labels[key]}`)
    .join(",");
  return suffix ? `${metric.name}{${suffix}}` : metric.name;
}

function value(byName, name) {
  const metric = byName.get(name);
  return metric ? metric.value : null;
}

function disks(metrics) {
  const rows = new Map();
  for (const metric of metrics) {
    const path = metric.labels && metric.labels.path;
    if (!path) continue;
    if (!rows.has(path)) rows.set(path, { path });
    const row = rows.get(path);
    if (metric.name === "gosysmon_disk_size_bytes") row.size = metric.value;
    if (metric.name === "gosysmon_disk_free_bytes") row.free = metric.value;
    if (metric.name === "gosysmon_disk_available_bytes") row.available = metric.value;
    if (metric.name === "gosysmon_disk_used_ratio") row.used = metric.value;
  }
  return [...rows.values()].sort((a, b) => a.path.localeCompare(b.path));
}

function renderDisks(rows) {
  el.disks.replaceChildren(
    ...rows.map((row) => {
      const item = document.createElement("div");
      item.className = "disk-row";
      const used = row.used || 0;
      item.innerHTML = `
        <div class="disk-head">
          <strong>${escapeHTML(row.path)}</strong>
          <span>${percent(used)}</span>
        </div>
        <div class="bar"><span class="${level(used)}" style="width:${Math.max(0, Math.min(100, used * 100))}%"></span></div>
        <div class="disk-path">${bytes(row.free)} free of ${bytes(row.size)}</div>
      `;
      return item;
    })
  );
}

function renderMetrics(metrics) {
  el.metrics.replaceChildren(
    ...metrics.slice(0, 40).map((metric) => {
      const item = document.createElement("div");
      item.className = "metric-row";
      item.innerHTML = `
        <div class="metric-head">
          <strong>${escapeHTML(metricKey(metric))}</strong>
          <span class="metric-value">${formatNumber(metric.value)}</span>
        </div>
        <div class="metric-help">${escapeHTML(metric.help || metric.type)}</div>
      `;
      return item;
    })
  );
}

function drawTrend() {
  const canvas = el.trend;
  const rect = canvas.getBoundingClientRect();
  const scale = window.devicePixelRatio || 1;
  canvas.width = Math.max(1, Math.floor(rect.width * scale));
  canvas.height = Math.max(1, Math.floor(rect.height * scale));

  const ctx = canvas.getContext("2d");
  ctx.scale(scale, scale);
  ctx.clearRect(0, 0, rect.width, rect.height);

  const padding = 28;
  ctx.strokeStyle = "#313a42";
  ctx.lineWidth = 1;
  for (let i = 0; i <= 4; i++) {
    const y = padding + ((rect.height - padding * 2) * i) / 4;
    ctx.beginPath();
    ctx.moveTo(padding, y);
    ctx.lineTo(rect.width - padding, y);
    ctx.stroke();
  }

  line(ctx, rect, "cpu", "#65a6ff");
  line(ctx, rect, "memory", "#45d6b5");
  line(ctx, rect, "disk", "#f0b84d");
  legend(ctx, rect);
}

function line(ctx, rect, key, color) {
  const values = state.history.map((row) => row[key]).filter((n) => n != null);
  if (values.length < 2) return;

  const padding = 28;
  const width = rect.width - padding * 2;
  const height = rect.height - padding * 2;
  const step = width / Math.max(1, values.length - 1);

  ctx.strokeStyle = color;
  ctx.lineWidth = 2;
  ctx.beginPath();
  values.forEach((value, index) => {
    const x = padding + index * step;
    const y = padding + height - Math.max(0, Math.min(1, value)) * height;
    if (index === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  });
  ctx.stroke();
}

function legend(ctx, rect) {
  const items = [
    ["CPU", "#65a6ff"],
    ["Memory", "#45d6b5"],
    ["Disk", "#f0b84d"],
  ];
  ctx.font = "12px system-ui";
  items.forEach(([label, color], index) => {
    const x = rect.width - 220 + index * 72;
    ctx.fillStyle = color;
    ctx.fillRect(x, 16, 10, 10);
    ctx.fillStyle = "#9aa6af";
    ctx.fillText(label, x + 16, 25);
  });
}

function percent(value) {
  return `${(value * 100).toFixed(1)}%`;
}

function level(value) {
  if (value >= 0.9) return "danger";
  if (value >= 0.75) return "warn";
  return "";
}

function bytes(value) {
  if (value == null) return "--";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = value;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit++;
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function formatNumber(value) {
  if (Math.abs(value) >= 1024) return bytes(value);
  if (value >= 0 && value <= 1) return value.toFixed(4);
  return Number(value).toLocaleString();
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, (char) => {
    return {
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#39;",
    }[char];
  });
}
