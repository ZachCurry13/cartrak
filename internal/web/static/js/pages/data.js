import { h, ago } from '../ui.js';
import { api } from '../api.js';
import { currentCar } from '../car.js';
import { carWords } from '../carwords.js';
import { isSaver } from '../mode.js';
import { dataRow } from '../datarows.js';
import { timelineChart } from '../timeline.js';

const KEY = 'cartrak.datatab';
const keep = { get: () => { try { return localStorage.getItem(KEY); } catch (e) { return null; } }, set: (v) => { try { localStorage.setItem(KEY, v); } catch (e) { /* ignore */ } } };
const pad = (n) => String(n).padStart(2, '0');
const dayOf = (ts) => { const d = new Date(ts * 1000); return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`; };
const timeOf = (ts) => { const d = new Date(ts * 1000); return `${pad(d.getHours())}:${pad(d.getMinutes())}`; };
const dayRange = (ts) => { const d = new Date(ts * 1000); d.setHours(0, 0, 0, 0); const a = Math.floor(d.getTime() / 1000); return [a, a + 86399]; };
const stamp = (ts) => new Date(ts * 1000).toLocaleString([], { weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });

// Every field, and every moment one changed, as spreadsheets in a zip. Admins only: it is the raw data.
function exportCard(session) {
  const days = h('select', { id: 'xd' }, ...[['1', 'Last 24 hours'], ['7', 'Last 7 days'], ['30', 'Last 30 days'], ['90', 'Last 90 days']].map(([v, t]) => h('option', { value: v }, t)));
  days.value = '7';
  const priv = h('input', { type: 'checkbox', id: 'xp' });
  // A download is opened by the browser itself and cannot say which car, so the car is named in the address. Without it the first car
  // answers, which is how an export meant for the second car came back with the first one's data.
  const cars = (session && session.cars) || [];
  const car = h('select', { id: 'xc', 'aria-label': 'Which car' }, ...cars.map((c) => h('option', { value: String(c.id) }, c.name)));
  car.value = currentCar() || (cars[0] ? String(cars[0].id) : '');
  const link = h('a', { class: 'btn', href: '#' }, 'Download the zip');
  const set = () => { link.href = `/api/export/fields.zip?days=${days.value}${priv.checked ? '&private=1' : ''}${car.value ? `&car=${encodeURIComponent(car.value)}` : ''}`; };
  days.addEventListener('change', set); priv.addEventListener('change', set); car.addEventListener('change', set);
  set();
  return h('section', { class: 'card' }, h('h2', {}, 'Export every field over time'),
    h('p', { class: 'muted' }, 'A zip of spreadsheets: every field of the newest reading, the fields that never changed, and a table with a row each time anything changed. Nothing is cut off, unlike printing this page.'),
    cars.length > 1 ? h('p', {}, h('label', { for: 'xc' }, 'Which car'), car) : null,
    h('label', { for: 'xd' }, 'How far back'), days,
    h('p', {}, h('label', { for: 'xp' }, priv, ' Include positions and the VIN (they are left out otherwise, so the file is safe to send)')), h('p', {}, link));
}

// The car in plain words, in tabs, for any moment in its history.
export default async function data(root, _a, session) {
  let d = null;
  let tab = keep.get() || 'summary';
  const head = h('div');
  const tabs = h('div', { class: 'seg datatabs' });
  const body = h('div');
  root.append(h('div', { class: 'page-title' }, h('h2', {}, 'Car data'), h('a', { class: 'btn sec', href: '#/fields' }, 'Field watcher')),
    h('p', { class: 'muted' }, 'Everything the car tells CarTrak, in plain words. Tap the ? next to anything to see what it means. You can also look at how the car was at any earlier time.'), head, tabs, body);

  async function load(at) {
    d = await api(`/api/car-data${at ? `?at=${at}` : ''}`);
    await drawHead();
    drawBody();
  }

  async function drawHead() {
    if (!d.ready) {
      head.replaceChildren(h('div', { class: 'card' }, h('p', {}, d.first ? `CarTrak has no reading that early. Its first reading is from ${stamp(d.first)}.` : 'No readings yet. Data appears after the first successful sync.'),
        d.first ? h('button', { onclick: () => load(d.first) }, 'Go to the first reading') : null));
      tabs.replaceChildren();
      body.replaceChildren();
      return;
    }
    const date = h('input', { type: 'date', value: dayOf(d.ts), min: d.first ? dayOf(d.first) : undefined, 'aria-label': 'Day' });
    const time = h('input', { type: 'time', value: timeOf(d.ts), 'aria-label': 'Time' });
    const go = () => { const t = new Date(`${date.value}T${time.value || '00:00'}`).getTime() / 1000; if (t) load(Math.floor(t)); };
    const [from, to] = dayRange(d.ts);
    const line = await api(`/api/timeline?from=${from}&to=${to}`).catch(() => null);
    head.replaceChildren(h('section', { class: 'card' },
      h('div', {}, h('b', {}, stamp(d.ts)), ' ', d.latest ? h('span', { class: 'chip ok' }, 'latest reading') : h('span', { class: 'chip work' }, `${ago(d.ts)}`)),
      h('p', { class: 'seg' },
        h('button', { class: 'sec', disabled: !d.prev, onclick: () => load(d.prev) }, '◀ Earlier'),
        h('button', { class: 'sec', disabled: !d.next, onclick: () => load(d.next) }, 'Later ▶'),
        d.latest ? null : h('button', { onclick: () => load(0) }, 'Latest')),
      h('div', { class: 'cols2' }, h('div', {}, h('label', {}, 'Day'), date), h('div', {}, h('label', {}, 'Time'), time)),
      h('p', {}, h('button', { class: 'sec', onclick: go }, 'Show the car at this time'),
        h('span', { class: 'muted' }, ' It shows the last reading CarTrak had by then.')),
      line && line.points.length > 1 ? timelineChart(line.points, from, to, d.ts, (t) => load(t), carWords(session).fuel) : null));
  }

  function drawBody() {
    if (!d.ready) return;
    const names = [...d.report.tabs.map((t) => [t.id, t.title]), ['raw', 'All fields']];
    if (!names.some(([id]) => id === tab)) tab = names[0][0];
    tabs.replaceChildren(...names.map(([id, title]) => h('button', { class: id === tab ? 'on' : '', onclick: () => { tab = id; keep.set(id); drawBody(); } }, title)));
    if (d.trimmed) return body.replaceChildren(h('div', { class: 'banner' }, 'The full detail of this older reading was removed to save space. CarTrak keeps the main numbers, trips, charging and charts.'));
    if (tab === 'raw') return raw();
    const t = d.report.tabs.find((x) => x.id === tab);
    body.replaceChildren(t.intro ? h('p', { class: 'muted' }, t.intro) : null,
      ...t.groups.map((g) => h('section', { class: 'card' }, h('h2', {}, g.title), g.intro ? h('p', { class: 'muted' }, g.intro) : null, ...g.rows.map((r) => dataRow(r, d.ts, d.latest)))));
  }

  // Every raw value, searchable. It is a big download, so Data saver asks first.
  async function raw() {
    const show = async () => {
      const r = await api(`/api/car-data/raw${d.latest ? '' : `?at=${d.ts}`}`);
      const entries = Object.entries(r.fields || {}).sort((a, b) => a[0].localeCompare(b[0]));
      const find = h('input', { type: 'search', placeholder: `Search ${entries.length} values…`, 'aria-label': 'Search values' });
      const list = h('section', { class: 'card' });
      const draw = () => {
        const q = find.value.trim().toLowerCase();
        const rows = entries.filter(([k, v]) => !q || k.toLowerCase().includes(q) || String(v).toLowerCase().includes(q));
        list.replaceChildren(...(rows.length ? rows.map(([k, v]) => h('div', { class: 'row' }, h('span', { class: 'grow muted' }, k), h('b', {}, String(v)))) : [h('p', { class: 'muted' }, 'No matches.')]));
      };
      find.addEventListener('input', draw);
      body.replaceChildren(session && session.me.role === 'admin' ? exportCard(session) : null, h('p', { class: 'muted' }, 'The raw list exactly as Ford sent it, for reference. The other tabs explain these in plain words.'), find, h('div', { style: 'height:8px' }), list);
      draw();
    };
    if (isSaver()) {
      body.replaceChildren(h('div', { class: 'card' }, h('p', { class: 'muted' }, 'Data saver is on. The raw list is a large download.'), h('button', { onclick: show }, 'Load all values')));
    } else {
      await show();
    }
  }

  await load(0);
}
