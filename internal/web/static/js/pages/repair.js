import { h, dist, when, duration, num } from '../ui.js';
import { api } from '../api.js';

const local = (d) => { const p = (n) => String(n).padStart(2, '0'); return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`; };
const secs = (input) => Math.floor(new Date(input.value).getTime() / 1000);

function tripLine(t) {
  const used = t.startSoc != null && t.endSoc != null;
  return h('div', { class: 'row' }, h('div', { class: 'grow' },
    h('b', {}, `${dist(t.distanceKm)} · ${duration(t.startTs, t.endTs)}`),
    h('div', { class: 'muted' }, `${when(t.startTs)}${used ? ` · battery ${num(t.startSoc, 0)}% to ${num(t.endSoc, 0)}%` : ''}`)));
}

function chargeLine(c) {
  return h('div', { class: 'row' }, h('div', { class: 'grow' }, h('b', {}, 'Charging'),
    h('div', { class: 'muted' }, `${when(c.startTs)} · ${num(c.startSoc, 0)}% to ${num(c.endSoc, 0)}%`)));
}

const column = (title, trips, charges) => h('section', { class: 'card' }, h('h2', {}, title),
  ...(trips.length || charges.length ? [...trips.map(tripLine), ...charges.map(chargeLine)] : [h('p', { class: 'muted' }, 'Nothing.')]));

// Fix a day: redo the trips and charging of a stretch of time from the stored readings, after a preview.
export default async function repair(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const from = h('input', { type: 'datetime-local', id: 'rf', 'aria-label': 'From' });
  const to = h('input', { type: 'datetime-local', id: 'rt', 'aria-label': 'To' });
  const out = h('div');
  const msg = h('p', { class: 'muted', role: 'status' });
  const range = (a, b) => { from.value = local(a); to.value = local(b); out.replaceChildren(); msg.textContent = ''; };
  const day = (back) => { const d = new Date(); d.setHours(0, 0, 0, 0); d.setDate(d.getDate() - back); return d; };
  const soon = () => new Date(Date.now() - 31 * 60 * 1000);

  async function preview() {
    msg.textContent = 'Reading what was stored…';
    out.replaceChildren();
    try {
      const p = await api('/api/repair/preview', { method: 'POST', body: { from: secs(from), to: secs(to) } });
      msg.textContent = '';
      if (p.blockers.length) return out.replaceChildren(...p.blockers.map((b) => h('p', { class: 'banner' }, b)));
      const note = p.note ? h('p', { class: 'banner notice' }, p.note) : null;
      const kept = p.unchangedTrips + p.unchangedCharges;
      const keptText = kept ? `${p.unchangedTrips} trip${p.unchangedTrips === 1 ? '' : 's'} and ${p.unchangedCharges} charging session${p.unchangedCharges === 1 ? '' : 's'} already match the stored readings and will be left alone.` : '';
      if (!p.oldTrips.length && !p.oldCharges.length && !p.newTrips.length && !p.newCharges.length) return out.replaceChildren(...[note, h('p', { class: 'banner notice' }, `Nothing needs fixing here. ${keptText}`)].filter(Boolean));
      const word = h('input', { id: 'rw', placeholder: 'Type REBUILD', autocomplete: 'off', 'aria-label': 'Type REBUILD to confirm' });
      const go = h('button', { class: 'danger', disabled: true, onclick: apply }, 'Rebuild');
      word.addEventListener('input', () => { go.disabled = word.value.trim() !== 'REBUILD'; });
      async function apply() {
        go.disabled = true;
        msg.textContent = 'Rebuilding…';
        try {
          const r = await api('/api/repair/apply', { method: 'POST', body: { from: secs(from), to: secs(to), confirm: 'REBUILD' } });
          out.replaceChildren(h('p', { class: 'banner notice' }, `Done: ${r.replacedTrips} trip${r.replacedTrips === 1 ? '' : 's'} and ${r.replacedCharges} charging session${r.replacedCharges === 1 ? '' : 's'} replaced by ${r.trips} and ${r.charges}.`),
            h('p', {}, h('a', { class: 'btn', href: '#/trips' }, 'See trips')));
          msg.textContent = '';
        } catch (e) { msg.textContent = e.message; go.disabled = false; }
      }
      out.replaceChildren(...[note,
        h('p', { class: 'muted' }, `Widened to whole trips: ${when(p.from)} to ${when(p.to)}. ${p.readings} stored readings. ${keptText}`),
        h('div', { class: 'cols' }, column('Now', p.oldTrips, p.oldCharges), column('After rebuilding', p.newTrips, p.newCharges)),
        h('section', { class: 'card' },
          h('p', { class: 'muted' }, 'This replaces the trips and charging sessions on the left with the ones on the right. Notes, categories you chose by hand and charger kWh you typed in are kept. Public links to the old trips stop working. Make a backup first if you want to be able to undo it.'),
          h('label', { for: 'rw' }, 'To confirm, type REBUILD'), word, h('p', {}, go))].filter(Boolean));
    } catch (e) { msg.textContent = e.message; }
  }

  range(day(1), day(0)); // yesterday
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Fix a day')),
    h('p', { class: 'muted' }, 'If trips or charging were recorded wrongly (a road trip joined into one long trip, a charge that does not add up), CarTrak can redo that stretch from the readings it stored, using the current rules. You see exactly what changes before anything does.'),
    h('section', { class: 'card' },
      h('p', { class: 'seg' },
        h('button', { class: 'sec', onclick: () => range(day(1), day(0)) }, 'Yesterday'),
        h('button', { class: 'sec', onclick: () => range(day(0), soon()) }, 'Today so far'),
        h('button', { class: 'sec', onclick: () => range(day(3), day(0)) }, 'Last 3 days')),
      h('label', { for: 'rf' }, 'From'), from, h('label', { for: 'rt' }, 'To'), to,
      h('p', {}, h('button', { onclick: preview }, 'Preview')), msg),
    out);
}
