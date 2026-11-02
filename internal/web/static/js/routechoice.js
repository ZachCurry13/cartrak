import { h, dist } from './ui.js';
import { api } from './api.js';

// For a trip with no phone GPS: work out the roads it could have taken and let the owner pick, with each one's length next to
// what the odometer said. The one closest to the odometer is offered first, and chosen for new estimates on its own.
export function routeChoiceBar(t, onChosen) {
  const box = h('div', { style: 'margin:8px 0 0' });
  const msg = h('p', { class: 'muted', role: 'status' });
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
  box.append(h('button', { class: 'sec', onclick: load }, 'Choose the road'), msg);
  return box;
}
