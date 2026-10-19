import { h, isMetric, setMetric, getTheme, setTheme, getTimeMode, setTimeMode } from '../ui.js';
import { MODES, getPref, setPref } from '../mode.js';
import { getReserve, setReserve } from '../charge.js';

// One labelled choice with optional help text.
function setting(id, label, select, help) {
  select.id = id;
  return h('div', { class: 'setting' }, h('label', { for: id }, label), select, help ? h('p', { class: 'muted' }, help) : null);
}

function choice(options, value, onchange) {
  const s = h('select', {}, ...options.map(([v, t]) => h('option', { value: v }, t)));
  s.value = value;
  s.addEventListener('change', () => onchange(s.value));
  return s;
}

// The battery level you do not want to go below on a road trip; Home and Live show the distance left until it.
function reserveInput() {
  const i = h('input', { type: 'number', min: 1, max: 90, inputmode: 'numeric', value: getReserve(), style: 'width:6em' });
  i.addEventListener('change', () => { setReserve(Number(i.value) || 20); i.value = getReserve(); });
  return i;
}

// The version these screens were loaded as: the address of this file carries it (/v0.26.0/js/...). Screens from before
// versioned addresses have none.
const loadedVersion = (import.meta.url.match(/\/v([^/]+)\/js\//) || [])[1] || 'older than 0.26';

// Throw away everything the app has saved on this device and load it again from the server.
async function refreshApp(btn, msg) {
  btn.disabled = true;
  msg.textContent = 'Clearing saved app files…';
  try {
    if ('serviceWorker' in navigator) for (const r of await navigator.serviceWorker.getRegistrations()) await r.unregister();
    if (window.caches) for (const k of await caches.keys()) await caches.delete(k);
  } catch (e) { /* carry on: the reload below still fetches fresh files */ }
  location.replace(`/?fresh=${Date.now()}#/display`);
}

// These choices apply to this device only.
export default async function display(root, _a, session) {
  const msg = h('p', { class: 'muted', role: 'status' });
  const btn = h('button', { onclick: () => refreshApp(btn, msg) }, 'Refresh the app');
  const server = session && session.about ? session.about.version : '';
  const stale = server && loadedVersion !== server;
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Display and data')),
    h('p', { class: 'muted' }, 'These choices apply to this device only.'),
    h('section', { class: 'group' }, h('div', { class: 'menu' },
      setting('s-un', 'Units', choice([['imperial', 'Miles, °F, psi'], ['metric', 'Kilometres, °C, kPa']], isMetric() ? 'metric' : 'imperial', (v) => setMetric(v === 'metric'))),
      setting('s-rs', 'Road-trip reserve (%)', reserveInput(), 'Home and Live show how far you can go before the battery reaches this level.'),
      setting('s-th', 'Theme', choice([['auto', 'Match my phone'], ['light', 'Light'], ['dark', 'Dark']], getTheme(), setTheme)),
      setting('s-tm', 'Trip times', choice([['trip', 'Where the trip happened'], ['device', 'This device']], getTimeMode(), setTimeMode),
        'Trips show the local time where they started, with the zone name.'),
      setting('s-dm', 'Data use', choice(Object.entries(MODES), getPref(), (v) => { setPref(v); location.reload(); }),
        'Normal refreshes every 30 s. Live pushes each new reading. Data saver shows the essentials, refreshes every 5 min, and loads big things only when you tap.'))),
    h('section', { class: 'card' }, h('h2', {}, 'If something looks old'),
      h('p', {}, `These screens are version ${loadedVersion}. The server is version ${server || 'unknown'}.`),
      stale ? h('p', { class: 'banner' }, 'They do not match, so this phone is showing old screens. Tap Refresh the app.') : h('p', { class: 'muted' }, 'They match.'),
      h('p', { class: 'muted' }, 'Refresh the app clears the files saved on this device and loads them again from the server. Your sign-in and your choices here are kept.'),
      h('p', {}, btn), msg));
}
