import { h } from './ui.js';
import { api } from './api.js';
import { matchDetail } from './placematch.js';

// The companies remembered by their account number, under Paste a list: see where each one goes, move it to the right place,
// or forget it. `load()` refreshes the list after a list was used.
export function accountList() {
  const box = h('div');
  async function load() {
    try {
      const l = await api('/api/company-list/accounts');
      box.replaceChildren(l.length ? h('details', { class: 'card', style: 'margin:8px 0;padding:10px' }, h('summary', {}, h('b', {}, `Your accounts (${l.length})`)),
        h('p', { class: 'muted' }, 'Where each account number goes. A pasted list fills these in by itself.'), ...l.map(row)) : '');
    } catch { box.replaceChildren(); }
  }
  const row = (a) => {
    const more = h('div');
    const change = () => {
      const q = h('input', { type: 'search', value: a.name, 'aria-label': `Search for ${a.name}`, style: 'flex:1 1 10em;min-width:0' });
      const found = h('div', { role: 'status' });
      const go = async () => {
        found.replaceChildren(h('p', { class: 'muted' }, 'Searching...'));
        try {
          const r = await api(`/api/named-places/search?q=${encodeURIComponent(q.value.trim())}`);
          found.replaceChildren(...(r.length ? r.map((m) => h('button', { class: 'sec', type: 'button', style: 'display:block;width:100%;text-align:left;margin:4px 0;white-space:normal', onclick: async () => {
            await api('/api/company-list/remember', { method: 'POST', body: { ref: a.ref, name: a.name, address: m.address || '', lat: m.lat, lon: m.lon } }).catch(() => {});
            load();
          } }, h('b', {}, m.name), h('div', { class: 'muted' }, matchDetail(m)))) : [h('p', { class: 'muted' }, 'Nothing found. Try the town or the address.')]));
        } catch (e) { found.replaceChildren(h('p', { class: 'muted' }, e.message)); }
      };
      q.addEventListener('keydown', (e) => { if (e.key === 'Enter') { e.preventDefault(); go(); } });
      more.replaceChildren(h('div', { class: 'row', style: 'gap:8px;flex-wrap:wrap;margin-top:6px' }, q, h('button', { class: 'sec', type: 'button', onclick: go }, 'Search')), found);
    };
    const forget = async () => {
      if (!confirm(`Forget where ${a.name} (${a.ref}) is? The next list asks again.`)) return;
      await api(`/api/company-list/accounts/${encodeURIComponent(a.ref)}`, { method: 'DELETE' }).catch(() => {});
      load();
    };
    return h('div', { style: 'border-top:1px solid var(--line);padding:8px 0' },
      h('div', { class: 'row', style: 'gap:8px;flex-wrap:wrap;align-items:center' },
        h('div', { style: 'flex:1 1 10em;min-width:0;overflow-wrap:anywhere' }, h('b', {}, a.name), h('div', { class: 'muted' }, [a.ref, a.address].filter(Boolean).join(' · '))),
        h('div', { class: 'seg' }, h('button', { class: 'text sm', type: 'button', onclick: change }, 'Change'), h('button', { class: 'text sm', type: 'button', onclick: forget }, 'Forget'))),
      more);
  };
  load();
  return { el: box, load };
}
