import { h, ago } from '../ui.js';
import { api } from '../api.js';

// Every value Ford returned in the latest reading, searchable.
export default async function data(root) {
  const latest = await api('/api/latest');
  if (!latest) return root.append(h('p', { class: 'muted' }, 'No readings yet.'));
  const entries = Object.entries(latest.extra || {}).sort((a, b) => a[0].localeCompare(b[0]));
  const find = h('input', { type: 'search', placeholder: `Search ${entries.length} fields…`, 'aria-label': 'Search fields' });
  const list = h('section', { class: 'card' });
  const draw = () => {
    const q = find.value.trim().toLowerCase();
    const rows = entries.filter(([k, v]) => !q || k.toLowerCase().includes(q) || String(v).toLowerCase().includes(q));
    list.replaceChildren(...(rows.length ? rows.map(([k, v]) => h('div', { class: 'row' }, h('span', { class: 'grow muted' }, k), h('b', {}, String(v))))
      : [h('p', { class: 'muted' }, 'No matches.')]));
  };
  find.addEventListener('input', draw);
  root.append(h('h2', {}, 'All data'), h('p', { class: 'muted' }, `Latest reading from ${ago(latest.ts)}. Raw responses are kept in the database.`), find, h('div', { style: 'height:12px' }), list);
  draw();
}
