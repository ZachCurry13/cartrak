import { h } from './ui.js';
import { PAGES, MAX_BAR, allowed, current, defaults, save, page, icon, isFuelCar } from './navlayout.js';

// Customize: choose the pages in the bottom bar and the tiles in the More sheet, in any order, or let them follow how often
// each page is opened. Saved to the person's account.
export function openEditor(session, done) {
  const now = current(session);
  const state = { bar: [...now.bar], tiles: [...now.tiles], auto: now.auto };
  const msg = h('p', { class: 'muted', role: 'status' });
  const body = h('div');
  const close = () => bg.remove();
  const move = (list, i, d) => { const j = i + d; if (j < 0 || j >= list.length) return; [list[i], list[j]] = [list[j], list[i]]; draw(); };
  const section = (title, list, max, note) => {
    const spare = PAGES.filter(([k]) => allowed(k, session) && !list.includes(k));
    const add = h('select', { 'aria-label': `Add to ${title}`, onchange: (e) => { if (!e.target.value) return; if (list.length >= max) { msg.textContent = `${title} holds ${max} at most.`; return; } list.push(e.target.value); draw(); } },
      h('option', { value: '' }, '+ Add…'), ...spare.map(([k, , label]) => h('option', { value: k }, label)));
    return h('section', {}, h('h3', { style: 'margin:14px 0 6px' }, title), h('p', { class: 'muted', style: 'margin:0 0 6px' }, note),
      ...list.map((k, i) => { const [, , label, d] = page(k); return h('div', { class: 'row navrow' }, icon(d), h('span', { class: 'grow' }, label),
        h('button', { type: 'button', class: 'ico sm', 'aria-label': `Move ${label} up`, disabled: i === 0, onclick: () => move(list, i, -1) }, '↑'),
        h('button', { type: 'button', class: 'ico sm', 'aria-label': `Move ${label} down`, disabled: i === list.length - 1, onclick: () => move(list, i, 1) }, '↓'),
        h('button', { type: 'button', class: 'ico sm', 'aria-label': `Remove ${label}`, disabled: list.length <= 1, onclick: () => { list.splice(i, 1); draw(); } }, '×')); }),
      spare.length ? h('p', {}, add) : null);
  };
  function draw() {
    const auto = h('input', { type: 'checkbox', id: 'navauto', checked: state.auto, onchange: (e) => { state.auto = e.target.checked; } });
    body.replaceChildren(
      h('p', {}, h('label', { for: 'navauto' }, auto, ' Arrange by how often I use them'), h('div', { class: 'muted' }, 'Keeps both lists in order of the pages you open most on this phone; Home stays first.')),
      section('Bottom bar', state.bar, MAX_BAR, `Up to ${MAX_BAR}, then More.`),
      section('More tiles', state.tiles, 12, 'The big buttons at the top of More.'),
      h('p', { class: 'seg', style: 'margin-top:14px' },
        h('button', { type: 'button', class: 'on', onclick: async () => { try { await save(state); close(); done(); } catch (e) { msg.textContent = e.message; } } }, 'Save'),
        h('button', { type: 'button', onclick: () => { Object.assign(state, defaults(session.me.role === 'admin', isFuelCar(session))); state.bar = [...state.bar]; state.tiles = [...state.tiles]; draw(); } }, 'Reset'),
        h('button', { type: 'button', onclick: close }, 'Cancel')), msg);
  }
  const sheet = h('div', { class: 'sheet more', role: 'dialog', 'aria-label': 'Customize' }, h('div', { class: 'mhandle', 'aria-hidden': 'true' }), h('div', { class: 'mhead' }, h('h2', {}, 'Customize')), body);
  const bg = h('div', { class: 'sheetbg bottom' }, sheet);
  bg.addEventListener('click', (e) => { if (e.target === bg) close(); });
  document.body.append(bg);
  draw();
}
