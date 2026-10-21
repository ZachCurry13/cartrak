import { h, dist } from '../ui.js';
import { api } from '../api.js';
import { createMap, mapConfig } from '../map.js';
import { flowChart, dayRing, ringKey, badgeGrid } from '../funcharts.js';
import { periodTitle, clockText } from '../funwords.js';

// Insights: Wrapped for a month or the year, everywhere you have driven, achievements, where the energy goes and a typical day.
// Under them, the detailed charts (#/analytics/charging, driving, records, battery).

const TABS = [['charging', 'Charging'], ['driving', 'Driving'], ['records', 'Records'], ['battery', 'Battery']];
const card = (title, ...kids) => h('section', { class: 'card' }, h('h2', {}, title), ...kids);

// A row of choices that redraws a part of the page.
function choices(list, value, onpick) {
  const seg = h('div', { class: 'seg', style: 'margin-bottom:10px' });
  const draw = (v) => seg.replaceChildren(...list.map(([k, t]) => h('button', { type: 'button', class: k === v ? 'on' : '', 'aria-pressed': String(k === v), onclick: () => { draw(k); onpick(k); } }, t)));
  draw(value);
  return seg;
}

function wrappedCards() {
  const now = new Date();
  const last = new Date(now.getFullYear(), now.getMonth() - 1, 1);
  const key = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
  const months = Array.from({ length: 12 }, (_, i) => new Date(now.getFullYear(), now.getMonth() - i, 1));
  const pick = h('select', { 'aria-label': 'Another month', onchange: (e) => { if (e.target.value) location.hash = `#/wrapped/${e.target.value}`; } },
    h('option', { value: '' }, 'Another month…'), ...months.map((d) => h('option', { value: key(d) }, periodTitle(key(d)))));
  const teaser = (href, kick, title, cls) => h('a', { class: `wteaser ${cls}`, href }, h('span', { class: 'kick' }, kick), h('b', {}, title), h('span', {}, 'Tap to play ▶'));
  return h('section', {}, h('div', { class: 'wteasers' }, teaser(`#/wrapped/${key(last)}`, 'Wrapped', periodTitle(key(last)), 'w-teal'), teaser(`#/wrapped/${now.getFullYear()}`, 'Wrapped so far', String(now.getFullYear()), 'w-violet')),
    h('p', {}, pick));
}

// The part of the map where most of the driving is: the middle 90% of the points each way, so one long trip does not shrink
// everyday driving to a dot. Zooming out shows the rest.
function mostOf(routes) {
  const pts = routes.flat();
  if (pts.length < 20) return null;
  const q = (arr, f) => arr[Math.floor(f * (arr.length - 1))];
  const lats = pts.map((p) => p[0]).sort((a, b) => a - b), lons = pts.map((p) => p[1]).sort((a, b) => a - b);
  return [[q(lats, 0.05), q(lons, 0.05)], [q(lats, 0.95), q(lons, 0.95)]];
}

async function glowCard() {
  const cfg = await mapConfig();
  if (!cfg.enabled) return null;
  const map = createMap(cfg, { height: 320, glow: true });
  const note = h('p', { class: 'muted', style: 'margin-top:8px' });
  const load = async (days) => {
    note.textContent = 'Drawing…';
    const d = await api(`/api/analytics/everywhere?days=${days}`);
    map.show({ routes: d.routes.map((points) => ({ points, color: '#5eead4', width: 2.2, opacity: 0.28 })), fit: mostOf(d.routes) });
    note.textContent = `${d.trips.toLocaleString()} trips · ${dist(d.km, 0)} · brightest where you drive most. Drag and pinch to explore.`;
  };
  const c = card('Everywhere you’ve been', choices([['90', '90 days'], ['365', 'Year'], ['0', 'All time']], '365', load), map.el, note);
  load(365);
  return c;
}

async function badgesCard() {
  const list = await api('/api/analytics/badges').catch(() => []);
  if (!list.length) return null;
  const earned = list.filter((b) => b.earned).length;
  const box = h('div', {}, badgeGrid(list, '$', 6));
  const more = h('button', { type: 'button', class: 'sec sm', onclick: () => { box.replaceChildren(badgeGrid(list, '$')); more.remove(); } }, `See all ${list.length}`);
  return card(`Achievements · ${earned} of ${list.length}`, box, h('p', { class: 'seg', style: 'margin-top:12px' }, more));
}

function lifeCards() {
  const flow = h('div'), ring = h('div'), sub = h('p', { class: 'muted' });
  const load = async (days) => {
    const d = await api(`/api/analytics/life?days=${days}`);
    flow.replaceChildren(flowChart(d.flow));
    const r = d.ring;
    ring.replaceChildren(dayRing(r), ringKey(), r.leaveAt ? h('p', { class: 'muted', style: 'text-align:center' }, `Out the door around ${clockText(r.leaveAt)} on weekdays${r.plugInAt ? `; plugged in around ${clockText(r.plugInAt)}` : ''}.`) : null);
    const drive = d.flow.cityKwh + d.flow.highwayKwh, km = d.flow.cityKm + d.flow.highwayKm;
    sub.textContent = drive > 0 && km > 0 ? `Highway driving took ${Math.round((d.flow.highwayKwh / drive) * 100)}% of the driving energy for ${Math.round((d.flow.highwayKm / km) * 100)}% of the distance.` : '';
  };
  load(30);
  return [card('Where every kWh went', choices([['30', '30 days'], ['90', '90 days'], ['365', 'Year']], '30', load), flow, sub), card('A day in the life', ring)];
}

export default async function analytics(root, args, session) {
  const tab = args[0];
  const head = [h('p', {}, h('a', { href: '#/reports' }, '← Reports')), h('div', { class: 'page-title' }, h('h2', {}, 'Insights')),
    h('nav', { class: 'atabs', 'aria-label': 'Insights' }, h('a', { href: '#/analytics', class: tab ? '' : 'on' }, 'Highlights'), ...TABS.map(([k, t]) => h('a', { href: `#/analytics/${k}`, class: tab === k ? 'on' : '' }, t)))];
  if (tab && TABS.some(([k]) => k === tab)) {
    const mod = await import(tab === 'charging' || tab === 'driving' ? '../insightsmore.js' : '../insightsmore2.js');
    const body = h('div');
    root.append(...head, body);
    await mod[tab](body, session);
    return;
  }
  root.append(...head, wrappedCards());
  const [glow, badges] = await Promise.all([glowCard(), badgesCard()]);
  root.append(...[glow, badges, ...lifeCards()].filter(Boolean));
}
