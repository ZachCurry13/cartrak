import { h, chip } from './ui.js';
import { api } from './api.js';
import { matchDetail } from './placematch.js';

const LABEL = { remembered: ['ok', 'Remembered'], saved: ['ok', 'Saved place'], found: ['ok', 'Found'], check: ['work', 'Pick one'], none: ['bad', 'Not found'] };

// Paste a list of companies (one per line, with or without the account number on the end) and get stops for the route planner.
// Ones CarTrak is sure of are filled in; the doubtful ones wait for a pick. `use(stops)` gets the confirmed places.
export function pasteListCard(use) {
  const text = h('textarea', { rows: 6, style: 'width:100%;box-sizing:border-box', maxlength: 10000, 'aria-label': 'List of companies', placeholder: 'Paste the list here' });
  const msg = h('p', { class: 'muted', role: 'status' }), out = h('div');

  async function find() {
    msg.textContent = 'Looking them up...'; out.replaceChildren();
    try {
      const r = await api('/api/company-list/resolve', { method: 'POST', body: { text: text.value } });
      msg.textContent = r.lookupOff ? 'Address lookup is off (City names in Settings), so only saved and remembered places were used.' : '';
      show(r);
    } catch (e) { msg.textContent = e.message; }
  }

  function show(r) {
    const pick = r.rows.map((x) => x.pick || null); // what each line will use; null waits for a choice
    const skip = r.rows.map(() => false);
    const list = h('div'), go = h('button', {}, 'Use these as stops');
    const ready = () => { const n = r.rows.filter((_, i) => !skip[i]).length; go.disabled = !n || r.rows.some((_, i) => !skip[i] && !pick[i]); };
    // A company can go by another name on the map than on the list: search for it under that name; the pick is remembered by its number.
    const another = (x) => {
      const q = h('input', { type: 'search', placeholder: 'Search by another name or town', 'aria-label': `Another name for ${x.name}`, style: 'flex:1 1 10em;min-width:0' });
      const note = h('div', { class: 'muted', role: 'status' });
      const go = async () => {
        if (!q.value.trim()) return;
        note.textContent = 'Searching...';
        try { x.options = await api(`/api/named-places/search?q=${encodeURIComponent(q.value.trim())}`); note.textContent = x.options.length ? '' : 'Nothing found. Try the town or the address.'; draw(); } catch (e) { note.textContent = e.message; }
      };
      q.addEventListener('keydown', (e) => { if (e.key === 'Enter') { e.preventDefault(); go(); } });
      return h('div', { style: 'margin-top:6px' }, h('div', { class: 'row', style: 'gap:8px;flex-wrap:wrap' }, q, h('button', { class: 'sec', type: 'button', onclick: go }, 'Search')), note);
    };
    const draw = () => {
      list.replaceChildren(...r.rows.map((x, i) => {
        const [kind, label] = LABEL[x.status];
        const chosen = pick[i];
        return h('div', { class: 'card', style: `margin:8px 0;padding:10px;${skip[i] ? 'opacity:.55' : ''}` },
          h('div', { class: 'row', style: 'gap:8px;flex-wrap:wrap' }, h('b', { class: 'grow', style: 'flex:1 1 8em;min-width:0;overflow-wrap:anywhere' }, x.name), chip(kind, chosen ? label : x.options.length ? label : x.error ? 'Lookup failed' : 'Not found')),
          chosen ? h('div', { class: 'muted' }, chosen.address || (x.status === 'saved' ? 'One of your places' : '')) : null,
          !chosen && !skip[i] ? h('div', {}, ...x.options.map((m) => h('button', { class: 'sec', style: 'display:block;width:100%;text-align:left;margin:4px 0;white-space:normal', onclick: () => { pick[i] = { name: x.name, address: m.address, lat: m.lat, lon: m.lon }; draw(); } }, h('b', {}, m.name), h('div', { class: 'muted' }, matchDetail(m)))),
            x.options.length ? null : h('p', { class: 'muted' }, x.error ? `The lookup service did not answer (${x.error}), so this is not the same as nothing being found. Wait a minute and paste again, or search below.` : 'No match. Search it under another name below, skip it, or add the town to the name and paste again.')) : null,
          !chosen && !skip[i] ? another(x) : null,
          h('div', { class: 'seg' }, chosen && !skip[i] && x.status !== 'saved' ? h('button', { class: 'text sm', onclick: () => { pick[i] = null; draw(); } }, 'Change') : null,
            h('button', { class: 'text sm', onclick: () => { skip[i] = !skip[i]; draw(); } }, skip[i] ? 'Put back' : 'Skip')));
      }));
      ready();
    };
    go.onclick = async () => {
      go.disabled = true;
      const stops = [];
      for (let i = 0; i < r.rows.length; i++) {
        if (skip[i]) continue;
        const p = pick[i], x = r.rows[i];
        stops.push({ name: p.name, label: p.address ? `${p.name}, ${p.address}` : p.name, lat: p.lat, lon: p.lon });
        if (x.ref && x.status !== 'remembered' && x.status !== 'saved') await api('/api/company-list/remember', { method: 'POST', body: { ref: x.ref, name: p.name, address: p.address || '', lat: p.lat, lon: p.lon } }).catch(() => {});
      }
      use(stops);
    };
    out.replaceChildren(r.more ? h('p', { class: 'banner' }, `Only the first 10 are used. ${r.more} more left out: paste them as the next day's list.`) : null, list, h('p', {}, go),
      h('p', { class: 'muted' }, 'Places you pick are remembered by their number, so the next list fills them in by itself.'));
    draw();
  }
  return h('div', {},
    h('p', { class: 'muted' }, 'Paste your list, one company per line. Only the company name is looked up. The account number and country some systems put after the name, and the lines under a name that only say when you last met, are skipped. These two lines are one search for "Acme Widgets":'),
    h('pre', { class: 'muted', style: 'margin:4px 0 8px;white-space:pre-wrap;overflow-wrap:anywhere' }, 'Acme Widgets, Inc.-1234567-U.S.A.\n12 - Visit'),
    h('p', { class: 'muted' }, 'The account number is only used to remember which place you picked, so next time that company is filled in by itself. Names are searched near Home. The trip starts from the Start on the Stops tab (the car position unless you changed it).'), text, h('p', {}, h('button', { onclick: find }, 'Find them')), msg, out);
}
