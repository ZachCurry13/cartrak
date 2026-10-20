import { h, when } from './ui.js';
import { api } from './api.js';

// When the road estimate is switched on, older trips that were recorded by the car alone are still straight lines. This
// offers to estimate their roads, a few at a time (the routing service is slow on purpose: it is for light use).
export async function routeEstimateBar() {
  const d = await api('/api/trips/estimate-routes').catch(() => null);
  if (!d || !d.enabled || !(d.pending > 0)) return null;
  const msg = h('span', {}, `${d.pending} older trip${d.pending === 1 ? ' is' : 's are'} drawn as a straight line.`);
  const btn = h('button', { class: 'sec' }, 'Estimate their roads');
  const list = h('ul', { class: 'muted', style: 'flex-basis:100%;margin:0;padding-left:18px' });
  const box = h('div', { class: 'banner notice', style: 'display:flex;gap:10px;align-items:center;flex-wrap:wrap' }, msg, btn, list);
  btn.addEventListener('click', async () => {
    btn.disabled = true;
    let left = d.pending, guard = 0;
    const failed = new Map();
    while (left > 0 && guard++ < 30) {
      msg.textContent = `Estimating… ${left} to go.`;
      const r = await api('/api/trips/estimate-routes?limit=10', { method: 'POST' }).catch(() => null);
      if (r) for (const f of r.failures || []) failed.set(f.id, f);
      if (!r || (r.done === 0 && r.failed > 0 && r.pending >= left)) {
        msg.textContent = failed.size ? 'Some trips could not be estimated, so they stay as a straight line:' : 'The routing service did not answer. Try again later.';
        list.replaceChildren(...[...failed.values()].map((f) => h('li', {}, h('a', { href: `#/trip/${f.id}` }, when(f.startTs)), `: ${f.why}.`)));
        btn.hidden = !!failed.size && r && r.pending <= failed.size; btn.disabled = false; return;
      }
      left = r.pending;
    }
    msg.textContent = left > 0 ? `${left} more can be done later.` : failed.size ? 'Done, except these trips, which stay as a straight line:' : 'Done. Open a trip to see its estimated road.';
    list.replaceChildren(...[...failed.values()].map((f) => h('li', {}, h('a', { href: `#/trip/${f.id}` }, when(f.startTs)), `: ${f.why}.`)));
    btn.hidden = true;
  });
  return box;
}
