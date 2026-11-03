import { h, when } from '../ui.js';
import { changeRow } from '../repairchanges.js';
import { api } from '../api.js';

const local = (d) => { const p = (n) => String(n).padStart(2, '0'); return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`; };
const secs = (input) => Math.floor(new Date(input.value).getTime() / 1000);

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
  // A whole month, the last twelve to choose from: from its first moment to the first moment of the next (or now, for this month).
  const now = new Date();
  const months = h('select', { id: 'rm', 'aria-label': 'A whole month', onchange: (e) => {
    if (!e.target.value) return;
    const [y, m] = e.target.value.split('-').map(Number);
    const end = new Date(y, m, 1);
    range(new Date(y, m - 1, 1), end > now ? soon() : end);
    e.target.value = '';
  } }, h('option', { value: '' }, 'Pick a whole month…'), ...Array.from({ length: 12 }, (_, i) => {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
    return h('option', { value: `${d.getFullYear()}-${d.getMonth() + 1}` }, d.toLocaleDateString([], { month: 'long', year: 'numeric' }));
  }));

  async function preview() {
    msg.textContent = secs(to) - secs(from) > 8 * 86400 ? 'Reading what was stored… a month can take up to a minute.' : 'Reading what was stored…';
    out.replaceChildren();
    try {
      const p = await api('/api/repair/preview', { method: 'POST', body: { from: secs(from), to: secs(to) } });
      msg.textContent = '';
      if (p.blockers.length) return out.replaceChildren(...p.blockers.map((b) => h('p', { class: 'banner' }, b)));
      const note = p.note ? h('p', { class: 'banner notice' }, p.note) : null;
      const kept = p.unchangedTrips + p.unchangedCharges;
      const keptText = kept ? `${p.unchangedTrips} trip${p.unchangedTrips === 1 ? '' : 's'} and ${p.unchangedCharges} charging session${p.unchangedCharges === 1 ? '' : 's'} already match the stored readings and will be left alone.` : '';
      if (!p.oldTrips.length && !p.oldCharges.length && !p.newTrips.length && !p.newCharges.length) return out.replaceChildren(...[note, h('p', { class: 'banner notice' }, `Nothing needs fixing here. ${keptText}`)].filter(Boolean));
      const rows = p.changes.map((c) => changeRow(c, p, refresh));
      const word = h('input', { id: 'rw', placeholder: 'Type REBUILD', autocomplete: 'off', 'aria-label': 'Type REBUILD to confirm' });
      const go = h('button', { class: 'danger', disabled: true, onclick: apply }, 'Rebuild');
      const ticked = () => rows.filter((r) => r.box.checked).map((r) => r.key);
      function refresh() {
        const n = ticked().length;
        go.textContent = `Rebuild ${n} of ${rows.length} change${rows.length === 1 ? '' : 's'}`;
        go.disabled = n === 0 || word.value.trim() !== 'REBUILD';
      }
      const setAll = (on) => { for (const r of rows) { r.box.checked = on; r.row.classList.toggle('off', !on); } refresh(); };
      word.addEventListener('input', refresh);
      async function apply() {
        go.disabled = true;
        msg.textContent = secs(to) - secs(from) > 8 * 86400 ? 'Rebuilding… a month can take up to a minute.' : 'Rebuilding…';
        try {
          const r = await api('/api/repair/apply', { method: 'POST', body: { from: secs(from), to: secs(to), confirm: 'REBUILD', keys: ticked() } });
          out.replaceChildren(h('p', { class: 'banner notice' }, `Done: ${r.replacedTrips} trip${r.replacedTrips === 1 ? '' : 's'} and ${r.replacedCharges} charging session${r.replacedCharges === 1 ? '' : 's'} replaced by ${r.trips} and ${r.charges}.`),
            h('p', {}, h('a', { class: 'btn', href: '#/trips' }, 'See trips')));
          msg.textContent = '';
        } catch (e) { msg.textContent = e.message; refresh(); }
      }
      out.replaceChildren(...[note,
        h('p', { class: 'muted' }, `${rows.length} change${rows.length === 1 ? '' : 's'} found between ${when(p.from)} and ${when(p.to)} (${p.readings} stored readings). Untick any you want to leave as it is. ${keptText}`),
        h('p', { class: 'seg' }, h('button', { type: 'button', class: 'text', onclick: () => setAll(true) }, 'Tick all'), h('button', { type: 'button', class: 'text', onclick: () => setAll(false) }, 'Untick all')),
        ...rows.map((r) => r.row),
        h('section', { class: 'card' },
          h('p', { class: 'muted' }, 'Only the ticked changes are made. Notes, categories you chose by hand and charger kWh you typed in are kept. Public links to the old trips stop working.'),
          h('label', { for: 'rw' }, 'To confirm, type REBUILD'), word, h('p', {}, go))].filter(Boolean));
      refresh();
    } catch (e) { msg.textContent = e.message; }
  }

  range(day(1), day(0)); // yesterday
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Fix a day')),
    h('p', { class: 'muted' }, 'If trips or charging were recorded wrongly (a road trip joined into one long trip, a charge that does not add up), CarTrak can redo that stretch from the readings it stored, using the current rules. You see exactly what changes before anything does.'),
    h('section', { class: 'card' },
      h('p', { class: 'seg' },
        h('button', { class: 'sec', onclick: () => range(day(1), day(0)) }, 'Yesterday'),
        h('button', { class: 'sec', onclick: () => range(day(0), soon()) }, 'Today so far'),
        h('button', { class: 'sec', onclick: () => range(day(3), day(0)) }, 'Last 3 days'),
        h('button', { class: 'sec', onclick: () => range(day(7), day(0)) }, 'Last 7 days'),
        h('button', { class: 'sec', onclick: () => range(day(30), day(0)) }, 'Last 30 days')),
      h('label', { for: 'rm' }, 'Or a whole month'), months,
      h('label', { for: 'rf' }, 'From'), from, h('label', { for: 'rt' }, 'To'), to,
      h('p', {}, h('button', { onclick: preview }, 'Preview')), msg),
    out);
}
