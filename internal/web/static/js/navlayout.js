import { api } from './api.js';

// Which pages sit in the bottom bar and which tiles fill the More sheet, in the order the person chose. It is saved to their
// account (so it follows them to another phone) and kept on this device for a quick start. With "auto" on, both are kept in
// order of how often this person opens each page on this device; Home stays first. More is always the last tab.

// Every page that can be put in the bar or the sheet: key, address, name, icon path, and who may see it.
export const PAGES = [
  ['home', '#/', 'Home', 'M3 11l9-8 9 8M5 10v10h5v-6h4v6h5V10'],
  ['trips', '#/trips', 'Trips', 'M12 21s7-6.2 7-11a7 7 0 10-14 0c0 4.8 7 11 7 11z|o12,10,2.5'],
  ['live', '#/live', 'Live', 'M13 2L4 14h7l-1 8 9-12h-7l1-8z'],
  ['charging', '#/charging', 'Charging', 'M9 2v5M15 2v5M6 7h12v4a6 6 0 01-12 0V7zM12 17v5', 'plug'],
  ['fuel', '#/fuel', 'Fuel', 'M5 21V4a1 1 0 011-1h7a1 1 0 011 1v17M4 21h11M8 8h3M14 9h2a2 2 0 012 2v5a1.5 1.5 0 003 0V9l-3-3', 'fuel'],
  ['health', '#/health', 'Health', 'M3 12h4l2-6 4 12 2-6h6', 'admin'],
  ['data', '#/data', 'Data', 'M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z', 'admin'],
  ['insights', '#/analytics', 'Insights', 'M12 3l2.5 5.5L20 9l-4 4 1 6-5-3-5 3 1-6-4-4 5.5-.5z'],
  ['reports', '#/reports', 'Reports', 'M4 20V10M10 20V4M16 20v-7M22 20H2'],
  ['day', '#/day', 'Day view', 'M4 5h16v15H4zM4 10h16M9 3v4M15 3v4'],
  ['history', '#/history', 'History', 'M12 7v5l3 3|o12,12,9'],
  ['places', '#/places', 'Places', 'M12 21s7-6.2 7-11a7 7 0 10-14 0c0 4.8 7 11 7 11z|o12,10,2.5'],
  ['plan', '#/plan', 'Plan a trip', 'M4 19l5-14 6 9 5-5M4 19h16'],
  ['maintenance', '#/maintenance', 'Service', 'M14 6l4 4-8 8H6v-4zM16 4l4 4', 'admin'],
  ['settings', '#/admin', 'Settings', 'o12,12,3.5|M12 3v3M12 18v3M3 12h3M18 12h3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M18.4 5.6l-2.1 2.1M7.7 16.3l-2.1 2.1', 'admin'],
  ['display', '#/display', 'Display', 'M4 5h16v11H4zM9 20h6M12 16v4'],
];
const byKey = new Map(PAGES.map((p) => [p[0], p]));
export const MAX_BAR = 6;

// A gas or hybrid car: it has a tank to watch instead of a plug.
export const isFuelCar = (session) => !!(session.vehicle && session.vehicle.engine && session.vehicle.powertrain && session.vehicle.powertrain !== 'ev');

export const defaults = (admin, fuelCar = false) => ({
  bar: admin ? ['home', 'trips', 'live', fuelCar ? 'fuel' : 'charging', 'health', 'data'] : ['home', 'trips', 'live', fuelCar ? 'fuel' : 'charging', 'insights'],
  tiles: admin ? ['insights', 'reports', 'day', 'places', 'plan', 'settings'] : ['reports', 'day', 'places', 'plan', 'history', 'display'],
  auto: false,
});

const LOCAL = 'cartrak.layout', USE = 'cartrak.use';
const read = (k, d) => { try { return JSON.parse(localStorage.getItem(k)) || d; } catch (e) { return d; } };
const write = (k, v) => { try { localStorage.setItem(k, JSON.stringify(v)); } catch (e) { /* storage blocked */ } };

let layout = null;

// May this person see this page now? Admin pages for admins, Charging only for a car that plugs in.
export function allowed(key, session) {
  const p = byKey.get(key);
  if (!p) return false;
  if (p[4] === 'admin') return session.me.role === 'admin';
  if (p[4] === 'plug') return !session.vehicle || session.vehicle.plug !== false;
  if (p[4] === 'fuel') return isFuelCar(session);
  return true;
}

// Counts a visit to a page, for the automatic order.
export function noteVisit(name) {
  const p = PAGES.find((x) => x[1] === `#/${name}` || (name === '' && x[0] === 'home'));
  if (!p) return;
  const use = read(USE, {});
  use[p[0]] = (use[p[0]] || 0) + 1;
  write(USE, use);
}

const byUse = (keys) => {
  const use = read(USE, {});
  return [...keys].sort((a, b) => (a === 'home' ? -1 : b === 'home' ? 1 : (use[b] || 0) - (use[a] || 0)));
};

// The layout to show: what was saved (here or on the account), ordered by use when auto is on, with pages this person may not see
// left out.
export function current(session) {
  const admin = session.me.role === 'admin';
  const l = layout || read(LOCAL, null) || defaults(admin, isFuelCar(session));
  const clean = (keys) => keys.filter((k) => byKey.has(k) && allowed(k, session));
  const bar = clean(l.bar).slice(0, MAX_BAR), tiles = clean(l.tiles);
  return { bar: l.auto ? byUse(bar) : bar, tiles: l.auto ? byUse(tiles) : tiles, auto: !!l.auto };
}

export async function load(session) {
  const saved = await api('/api/me/layout').catch(() => null);
  layout = saved && saved.bar ? saved : defaults(session.me.role === 'admin', isFuelCar(session));
  write(LOCAL, layout);
}

export async function save(next) {
  layout = { bar: next.bar.slice(0, MAX_BAR), tiles: next.tiles.slice(0, 12), auto: !!next.auto };
  write(LOCAL, layout);
  await api('/api/me/layout', { method: 'PUT', body: layout });
}

export const page = (key) => byKey.get(key);

const NS = 'http://www.w3.org/2000/svg';
// An outline icon from a path list: parts split by |, a part "ox,y,r" is a circle.
export function icon(d) {
  const s = document.createElementNS(NS, 'svg');
  s.setAttribute('viewBox', '0 0 24 24');
  s.setAttribute('aria-hidden', 'true');
  for (const part of d.split('|')) {
    const p = document.createElementNS(NS, part.startsWith('o') ? 'circle' : 'path');
    if (part.startsWith('o')) { const [cx, cy, r] = part.slice(1).split(','); p.setAttribute('cx', cx); p.setAttribute('cy', cy); p.setAttribute('r', r); } else p.setAttribute('d', part);
    s.append(p);
  }
  return s;
}

// Puts the chosen pages in the bottom bar, before More (which stays, with its own click handler).
export function applyBar(tabs, session) {
  const more = tabs.querySelector('a[href="#/settings"]');
  for (const a of [...tabs.querySelectorAll('a')]) if (a !== more) a.remove();
  for (const key of current(session).bar) {
    const [, href, label, d] = page(key);
    const a = document.createElement('a');
    a.href = href;
    const span = document.createElement('span');
    span.textContent = label;
    a.append(icon(d), span);
    tabs.insertBefore(a, more);
  }
}
