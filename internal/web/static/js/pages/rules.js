import { h, chip } from '../ui.js';
import { api } from '../api.js';

const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const toTime = (m) => `${String(Math.floor(m / 60) % 24).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`;
const fromTime = (s) => { const [a, b] = s.split(':').map(Number); return a * 60 + b; };
const summary = (r) => `${DAYS.filter((_, i) => r.days & (1 << i)).join(', ') || 'no days'} · ${toTime(r.startMin)}–${r.endMin === 1440 ? '24:00' : toTime(r.endMin)}${r.geoRadius ? ' · near a place' : ''}`;

export default async function rules(root, _a, session) {
  const admin = session.me.role === 'admin';
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('h2', {}, 'Trip rules'),
    h('p', { class: 'muted' }, 'A trip is labelled by the highest-priority rule that matches when it starts. Anything you set by hand on a trip always wins.'), box);

  async function draw() {
    const rs = await api('/api/rules');
    box.replaceChildren(...rs.map((r) => h('section', { class: 'card' },
      h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow' }, r.name), chip(r.category, r.category)),
      h('div', { class: 'muted' }, `${summary(r)} · priority ${r.priority}${r.enabled ? '' : ' · off'}`),
      admin ? h('p', { class: 'seg' }, h('button', { class: 'sec', onclick: () => edit(r) }, 'Edit'),
        h('button', { class: 'danger', onclick: async () => { if (confirm(`Delete "${r.name}"?`)) { await api(`/api/rules/${r.id}`, { method: 'DELETE' }); draw(); } } }, 'Delete')) : null)));
    if (admin) {
      box.append(h('p', { class: 'seg' }, h('button', { onclick: () => edit(null) }, 'Add rule'),
        h('button', { class: 'sec', onclick: async (e) => { await api('/api/rules/apply', { method: 'POST' }); e.target.textContent = 'Applied ✓'; } }, 'Re-apply to past trips')));
    }
  }

  function edit(r) {
    const v = r || { name: '', category: 'work', priority: 0, enabled: true, days: 62, startMin: 480, endMin: 1020 };
    const name = h('input', { value: v.name, maxlength: 60, placeholder: 'Weekday work hours' });
    const cat = h('select', {}, h('option', { value: 'work' }, 'Work'), h('option', { value: 'personal' }, 'Personal'));
    cat.value = v.category;
    const st = h('input', { type: 'time', value: toTime(v.startMin) }), en = h('input', { type: 'time', value: v.endMin === 1440 ? '23:59' : toTime(v.endMin) });
    const pri = h('input', { type: 'number', value: v.priority }), on = h('input', { type: 'checkbox' });
    on.checked = v.enabled;
    const days = DAYS.map((d, i) => { const c = h('input', { type: 'checkbox' }); c.checked = !!(v.days & (1 << i)); return [c, h('label', {}, c, d)]; });
    const lat = h('input', { type: 'number', step: 'any', value: v.geoLat ?? '', placeholder: 'Latitude' });
    const lon = h('input', { type: 'number', step: 'any', value: v.geoLon ?? '', placeholder: 'Longitude' });
    const rad = h('input', { type: 'number', value: v.geoRadius ?? '', placeholder: 'Radius (metres)' });
    const msg = h('p', { class: 'muted', role: 'alert' });
    // Fill the place from one you named under More, Places, instead of typing coordinates.
    const named = [];
    const pick = h('select', { id: 'rp', 'aria-label': 'Use a named place' }, h('option', { value: '' }, 'Choose a place you named…'));
    api('/api/named-places').then((ps) => { named.push(...ps); for (const x of ps) pick.append(h('option', { value: String(x.id) }, x.name)); }).catch(() => { /* none */ });
    pick.addEventListener('change', () => {
      const x = named.find((n) => String(n.id) === pick.value);
      if (x) { lat.value = x.lat; lon.value = x.lon; rad.value = x.radiusM; }
    });
    box.replaceChildren(h('section', { class: 'card' }, h('h2', {}, r ? 'Edit rule' : 'New rule'),
      h('label', {}, 'Name'), name, h('label', {}, 'Label trips as'), cat,
      h('label', {}, 'Days'), h('div', { class: 'days' }, days.map((d) => d[1])),
      h('label', {}, 'Trips starting from'), st, h('label', {}, 'until'), en,
      h('label', {}, 'Priority (higher wins)'), pri,
      h('label', {}, 'Optional: only when the trip starts near a place'), pick, lat, lon, rad,
      h('label', {}, h('span', {}, on, ' Enabled')),
      h('p', { class: 'seg' }, h('button', { onclick: async () => {
        try {
          const num = (el) => el.value === '' ? null : Number(el.value);
          const body = { name: name.value, category: cat.value, priority: Number(pri.value) || 0, enabled: on.checked,
            days: days.reduce((a, [c], i) => a | (c.checked ? 1 << i : 0), 0), startMin: fromTime(st.value),
            endMin: en.value === '23:59' ? 1440 : fromTime(en.value), geoLat: num(lat), geoLon: num(lon), geoRadius: num(rad) };
          await api(r ? `/api/rules/${r.id}` : '/api/rules', { method: r ? 'PUT' : 'POST', body });
          draw();
        } catch (e) { msg.textContent = e.message; }
      } }, 'Save'), h('button', { class: 'sec', onclick: draw }, 'Cancel')), msg));
  }
  await draw();
}
