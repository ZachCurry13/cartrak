import { h, dist } from './ui.js';
import { api } from './api.js';
import { routeStopsBox } from './routestops.js';

// For a trip with no phone GPS. With no road drawn yet (a straight line between the few places the car reported), one button works
// out the road closest to the odometer and draws it. With a road already drawn, the roads it could have taken are offered, each
// with its length next to what the odometer said, to pick another.
export function routeChoiceBar(t, hasRoute, onChosen) {
  const box = h('div', { style: 'margin:8px 0 0' });
  const msg = h('p', { class: 'muted', role: 'status' });
  if (!hasRoute) {
    const go = h('button', { onclick: async () => {
      go.disabled = true;
      msg.textContent = 'Working out the road between the places the car reported…';
      try {
        const d = await api(`/api/trips/${t.id}/route-options`);
        await api(`/api/trips/${t.id}/route-choice`, { method: 'POST', body: { kind: d.options[0].kind } }); // the road closest to the odometer comes first
        onChosen();
      } catch (e) { msg.replaceChildren(e.message, ' ', h('a', { href: '#/admin' }, 'Settings')); go.disabled = false; }
    } }, 'Estimate the route');
    box.append(h('p', { class: 'muted', style: 'margin:0 0 6px' }, 'Only a few positions were recorded for this trip, so it is drawn as a straight line.'), go, msg, routeStopsBox(t, onChosen));
    return box;
  }
  async function load(e) {
    if (e) e.target.disabled = true;
    msg.textContent = 'Asking the routing service for the roads between the places the car reported…';
    try {
      const d = await api(`/api/trips/${t.id}/route-options`);
      msg.textContent = '';
      let pick = d.options[0].kind;
      const rows = d.options.map((o) => {
        const diff = d.odoKm > 0 ? ((o.km - d.odoKm) / d.odoKm) * 100 : null;
        const radio = h('input', { type: 'radio', name: `rc${t.id}`, checked: o.kind === pick, onchange: () => { pick = o.kind; } });
        return h('label', { class: 'row', style: 'gap:8px;align-items:center;min-height:44px' }, radio,
          h('span', { class: 'grow' }, h('b', {}, o.label), ' ', dist(o.km, 1), diff != null ? h('span', { class: 'muted' }, ` (${diff >= 0 ? '+' : ''}${diff.toFixed(0)}% against the odometer)`) : null));
      });
      box.replaceChildren(h('p', { class: 'muted', style: 'margin:0 0 4px' }, d.odoTrusted ? `The odometer says ${dist(d.odoKm, 1)}. Pick the road that fits.` : `The distance was set by hand (${dist(d.odoKm, 1)}), so no road is picked to match it.`),
        ...rows, h('p', {}, h('button', { onclick: async (ev) => {
          ev.target.disabled = true;
          try { await api(`/api/trips/${t.id}/route-choice`, { method: 'POST', body: { kind: pick } }); onChosen(); } catch (err) { msg.textContent = err.message; ev.target.disabled = false; }
        } }, 'Use this road')), msg);
    } catch (err) { msg.textContent = err.message; if (e) e.target.disabled = false; }
  }
  box.append(h('button', { class: 'sec', onclick: load }, 'Choose a different road'), msg, routeStopsBox(t, onChosen));
  return box;
}
