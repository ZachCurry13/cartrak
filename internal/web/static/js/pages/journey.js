import { h, stat, chip, catChip, dist, when, duration, num } from '../ui.js';
import { api } from '../api.js';
import { createMap, mapConfig } from '../map.js';
import { efficiency, kwhText } from '../energy.js';

const hm = (sec) => { const m = Math.round(sec / 60); return m < 60 ? `${m} min` : `${Math.floor(m / 60)} h ${m % 60} min`; };
const money = (cur, v) => `${cur}${v.toFixed(2)}`;

// Several trips as one: totals, the legs, the stops between them, what it used and what charging cost.
export default async function journey(root, args, session) {
  const id = args[0];
  const admin = session.me.role === 'admin';
  const mc = await mapConfig();
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/trips' }, '← Trips')), box);

  async function load() {
    const d = await api(`/api/journeys/${id}`);
    const s = d.summary;
    const cur = d.currency;
    const name = h('input', { value: d.journey.name, maxlength: 80, 'aria-label': 'Journey name', disabled: !admin });
    const save = h('button', { class: 'sec', onclick: async () => { await api(`/api/journeys/${id}`, { method: 'PATCH', body: { name: name.value } }); load(); } }, 'Save name');
    const map = mc.enabled ? createMap(mc, { height: 320 }) : null;
    box.replaceChildren(
      h('div', { class: 'page-title' }, h('h2', {}, d.journey.name), chip('work', `${s.trips} trips`)),
      h('p', { class: 'muted' }, `${when(s.startTs)} to ${when(s.endTs)}`),
      h('section', { class: 'card' }, h('div', { class: 'grid' },
        stat('Distance', dist(s.distanceKm, 1)), stat('Driving time', hm(s.movingSec)), stat('Elapsed', hm(s.elapsedSec)),
        ...(s.kwhUsed != null ? [stat('Energy used', `${s.kwhEstimated ? '≈ ' : ''}${kwhText(s.kwhUsed)}`), stat('Efficiency', efficiency(s.distanceKm, s.kwhUsed) || '—')] : []),
        ...(s.chargedKwh > 0 ? [stat('Charged on the way', kwhText(s.chargedKwh))] : []),
        ...(s.cost != null ? [stat(s.costEstimate ? 'Cost (estimate)' : 'Charging cost', money(cur, s.cost))] : []))),
      s.cost == null ? h('p', { class: 'muted' }, 'Add your price per kWh in More, Settings to see what this journey cost.') : s.costEstimate ? h('p', { class: 'muted' }, 'No priced charging sessions were recorded, so the cost is the energy used (plus charging losses) at your price per kWh.') : null,
      map ? h('section', { class: 'card' }, h('h2', {}, 'Route'), map.el) : null,
      h('section', { class: 'card' }, h('h2', {}, 'Legs and stops'),
        ...d.trips.flatMap((t, i) => {
          const stop = i > 0 ? s.stops.find((x) => x.toTs === t.startTs) : null;
          return [
            stop ? h('div', { class: 'muted', style: 'padding:6px 0' }, `Parked ${hm(stop.seconds)}${stop.charging ? ` and charged${stop.addedKwh ? ` ${num(stop.addedKwh, 1, ' kWh')}` : ''}` : ''}`) : null,
            h('a', { class: 'row', href: `#/trip/${t.id}`, style: 'color:inherit;text-decoration:none' },
              h('div', { class: 'grow' }, h('b', {}, dist(t.distanceKm)), ' ', catChip(t), h('div', { class: 'muted' }, `${when(t.startTs, t.tz)} · ${duration(t.startTs, t.endTs)}`)),
              admin ? h('button', { class: 'sec', onclick: async (e) => { e.preventDefault(); await api(`/api/journeys/${id}`, { method: 'PATCH', body: { remove: [t.id] } }); load(); } }, 'Remove') : null),
          ];
        }).filter(Boolean)),
      admin ? h('section', { class: 'card' }, h('h2', {}, 'Manage'), name,
        h('p', { class: 'seg' }, save, h('button', { class: 'sec', onclick: async () => { if (!confirm('Ungroup this journey? The trips stay.')) return; await api(`/api/journeys/${id}`, { method: 'DELETE' }); location.hash = '#/trips'; } }, 'Ungroup trips'))) : null);
    if (map) {
      const routes = await api(`/api/trips/routes?journey=${id}`).catch(() => []);
      if (routes.length) map.show({ routes: routes.map((r) => ({ points: r.points, color: '#2fd4a0', width: 4, opacity: 0.9, onClick: () => { location.hash = `#/trip/${r.id}`; } })) });
      else map.el.replaceWith(h('p', { class: 'muted' }, 'No GPS route was recorded for these trips.'));
    }
  }
  await load();
}
