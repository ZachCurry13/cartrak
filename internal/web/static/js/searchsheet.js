import { h } from './ui.js';
import { buildIndex, search } from './searchindex.js';

// A search box that opens over any page: type a word and jump to the page or setting that has it.
let open = null;

export async function openSearch(admin) {
  if (open) return;
  const box = h('input', { type: 'search', placeholder: 'Search settings and pages', 'aria-label': 'Search settings and pages', autocomplete: 'off' });
  const list = h('div', { role: 'listbox' });
  const none = h('p', { class: 'muted', hidden: true }, 'Nothing matches. Try a different word.');
  const sheet = h('div', { class: 'sheet', role: 'dialog', 'aria-label': 'Search' }, box, none, list);
  const bg = h('div', { class: 'sheetbg' }, sheet);
  const close = () => { bg.remove(); open = null; document.removeEventListener('keydown', onKey); };
  const onKey = (e) => { if (e.key === 'Escape') close(); };
  bg.addEventListener('click', (e) => { if (e.target === bg) close(); });
  document.addEventListener('keydown', onKey);
  document.body.append(bg);
  open = bg;
  box.focus();
  const index = await buildIndex(admin);
  box.addEventListener('input', () => {
    const q = box.value.trim();
    const hits = q ? search(index, q, 8) : [];
    none.hidden = !q || hits.length > 0;
    list.replaceChildren(...hits.map((e) => h('a', { class: 'sg', href: e.href, role: 'option', onclick: close },
      h('b', {}, e.title), ' ', h('span', { class: 'muted' }, e.where), e.text ? h('div', { class: 'muted' }, e.text) : null)));
  });
}
