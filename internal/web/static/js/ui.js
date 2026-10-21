// Tiny DOM helpers. All data goes in through textContent, never innerHTML.

// The browser's own append and replaceChildren turn null into the word "null". Pages pass null for "nothing here", as h() already
// treats it, so leave null, undefined and false out of every call. This is why no page can show a stray "null".
for (const name of ['append', 'prepend', 'replaceChildren', 'before', 'after']) {
  for (const proto of [Element.prototype, DocumentFragment.prototype]) {
    const native = proto[name];
    if (typeof native === 'function') proto[name] = function (...nodes) { return native.apply(this, nodes.filter((n) => n != null && n !== false)); };
  }
}
export function h(tag, attrs, ...kids) {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (k === 'class') e.className = v;
    else if (k.startsWith('on')) e.addEventListener(k.slice(2), v);
    else if (v === true) e.setAttribute(k, '');
    else if (v !== false && v != null) e.setAttribute(k, v);
  }
  for (const k of kids.flat()) {
    if (k == null || k === false) continue;
    e.append(k.nodeType ? k : document.createTextNode(String(k)));
  }
  return e;
}

// ---- units (imperial default, remembered per device) ----
let metric = false;
try { metric = localStorage.getItem('units') === 'metric'; } catch (e) { /* storage blocked */ }
export const isMetric = () => metric;
export function setMetric(m) {
  metric = m;
  try { localStorage.setItem('units', m ? 'metric' : 'imperial'); } catch (e) { /* ignore */ }
}
const dash = '—';
const n = (v) => v == null || Number.isNaN(Number(v));
export const dist = (km, d = 1) => n(km) ? dash : metric ? `${(+km).toFixed(d)} km` : `${(km * 0.621371).toFixed(d)} mi`;
export const speed = (kph) => n(kph) ? dash : metric ? `${Math.round(kph)} km/h` : `${Math.round(kph * 0.621371)} mph`;
export const temp = (c) => n(c) ? dash : metric ? `${Math.round(c)}°C` : `${Math.round(c * 9 / 5 + 32)}°F`;
// The server names weather groups in Celsius ("0 to 10 °C"). Say them in this device's units.
export function tempBandLabel(name) {
  if (metric || !name) return name;
  const f = (c) => Math.round(c * 9 / 5 + 32);
  let m = /^(-?\d+) to (-?\d+) °C$/.exec(name);
  if (m) return `${f(+m[1])} to ${f(+m[2])} °F`;
  m = /^(-?\d+) °C and above$/.exec(name);
  if (m) return `${f(+m[1])} °F and above`;
  return name;
}
// Short distances (the size of a place) in this device's units: meters, or feet.
export const shortDist = (m) => (metric ? `${Math.round(m)} m` : `${Math.round((m * 3.28084) / 10) * 10} ft`);
export const smallUnit = () => (metric ? 'meters' : 'feet');
export const fromMeters = (m) => (metric ? Math.round(m) : Math.round(m * 3.28084));
export const toMeters = (v) => (metric ? v : v / 3.28084);
export const pressure = (kpa) => n(kpa) ? dash : metric ? `${Math.round(kpa)} kPa` : `${(kpa * 0.145038).toFixed(0)} psi`;
export const pct = (v) => n(v) ? dash : `${Math.round(v)}%`;
export const num = (v, d = 1, u = '') => n(v) ? dash : `${(+v).toFixed(d)}${u}`;

export function ago(ts) {
  if (!ts) return 'never';
  const s = Math.max(0, Math.floor(Date.now() / 1000 - ts));
  if (s < 90) return 'just now';
  if (s < 5400) return `${Math.round(s / 60)} min ago`;
  if (s < 129600) return `${Math.round(s / 3600)} h ago`;
  return `${Math.round(s / 86400)} d ago`;
}
export function duration(a, b) {
  if (!a || !b) return dash;
  const m = Math.round((b - a) / 60);
  return m < 60 ? `${m} min` : `${Math.floor(m / 60)} h ${m % 60} min`;
}

export const stat = (label, value) => h('div', { class: 'stat' }, h('div', { class: 'v' }, value), h('div', { class: 'l' }, label));
// emptyState is a friendly "nothing here yet" with what to do next (an optional button link).
export const emptyState = (title, body, href, label) => h('div', { class: 'empty' }, h('b', {}, title), h('p', { class: 'muted' }, body), href ? h('a', { class: 'btn', href }, label) : null);
export const chip = (cls, text) => h('span', { class: `chip ${cls}` }, text);
// The categories of this car, so a trip's label can use the owner's own name and color.
let cats = {};
export function setCategories(list) { cats = Object.fromEntries(list.map((c) => [c.key, c])); }
export const catChip = (t) => {
  const c = cats[t.category];
  const e = chip(t.category, (c ? c.label : t.category) + (t.categorySource === 'manual' ? ' ✎' : t.categorySource === 'rule' ? ' ⚙' : t.categorySource === 'vacation' ? ' · vacation' : t.categorySource === 'mode' ? ' · mode' : ''));
  if (c && c.color) { e.style.color = c.color; e.style.borderColor = c.color; }
  return e;
};

// ---- theme: auto follows the phone, or force light/dark ----
export function getTheme() { try { return localStorage.getItem('theme') || 'auto'; } catch (e) { return 'auto'; } }
export function setTheme(t) {
  try { localStorage.setItem('theme', t); } catch (e) { /* ignore */ }
  applyTheme();
}
export function applyTheme() {
  const t = getTheme();
  if (t === 'auto') document.documentElement.removeAttribute('data-theme');
  else document.documentElement.setAttribute('data-theme', t);
}

// ---- times: where a trip happened, or this device's own clock ----
let timeMode = 'trip';
try { timeMode = localStorage.getItem('timeMode') || 'trip'; } catch (e) { /* storage blocked */ }
export const getTimeMode = () => timeMode;
export function setTimeMode(m) {
  timeMode = m;
  try { localStorage.setItem('timeMode', m); } catch (e) { /* ignore */ }
}
const deviceZone = (() => { try { return Intl.DateTimeFormat().resolvedOptions().timeZone; } catch (e) { return ''; } })();

function zoneName(ts, zone) {
  try {
    const p = new Intl.DateTimeFormat('en-US', { timeZone: zone, timeZoneName: 'short' }).formatToParts(new Date(ts * 1000));
    return (p.find((x) => x.type === 'timeZoneName') || {}).value || zone;
  } catch (e) { return zone; }
}

// when(ts, tz): a trip's time in the zone where it started (default), or this
// device's time. The zone name is added when it differs from the device's.
export function when(ts, tz) {
  if (!ts) return dash;
  const opts = { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' };
  const zone = timeMode === 'trip' && tz ? tz : '';
  if (zone) opts.timeZone = zone;
  let s;
  try { s = new Date(ts * 1000).toLocaleString([], opts); } catch (e) { delete opts.timeZone; s = new Date(ts * 1000).toLocaleString([], opts); }
  return zone && zone !== deviceZone ? `${s} ${zoneName(ts, zone)}` : s;
}

// ---- menu rows (used by the More page) ----
export const item = (href, title, sub, right) => h('a', { class: 'item', href },
  h('span', { class: 'it' }, h('b', {}, title), sub ? h('span', { class: 'muted' }, sub) : null), right || null);
export const group = (title, ...items) => h('section', { class: 'group' },
  title ? h('h3', { class: 'gt' }, title) : null, h('div', { class: 'menu' }, ...items.filter(Boolean)));

// Ford sends words like NOT_READY or DISCONNECTED; show them as plain words.
export const nice = (v) => (v ? String(v).replace(/_/g, ' ').toLowerCase().replace(/^./, (c) => c.toUpperCase()) : v);
