import { h, chip } from './ui.js';
import { api } from './api.js';
import { startTour } from './tourui.js';

// The setup checklist: suggestions, none required. Home shows the next few until the person hides the card; the whole list
// stays under More, Setup checklist. Installing and the tour are known only on this phone; the rest comes from /api/checklist.
// Any suggestion can be skipped (swipe it left, or tap ×): it leaves the Home card and shows as skipped on the list, with Undo.
const standalone = () => window.matchMedia('(display-mode: standalone)').matches || navigator.standalone === true;
const toured = () => { try { return localStorage.getItem('tourDone') === '1'; } catch (e) { return false; } };

export function setupItems(st, session) {
  const admin = session.me.role === 'admin';
  const link = (href, text) => h('a', { class: 'btn sec', href }, text);
  const all = [
    { key: 'install', done: standalone(), title: 'Put CarTrak on your home screen', body: 'It opens like an app and keeps you signed in. In the browser menu choose Install app or Add to Home screen.' },
    { key: 'tour', done: toured(), title: 'Take the two-minute tour', body: 'A quick walk through each part of CarTrak.', action: h('button', { type: 'button', class: 'sec', onclick: () => startTour(session) }, 'Start') },
    { key: 'alerts', done: st.alerts, title: 'Turn on alerts on this phone', body: 'Charging done, plug in tonight, a low tire and more.', action: link('#/notifications', 'Choose') },
  ];
  if (admin) all.push(
    { key: 'home', done: st.home, title: 'Name Home', body: 'Trips read "Home → Work", and charging at home is priced and counted as home.', action: link('#/places', 'Places') },
    { key: 'price', done: st.price, title: 'Set your electricity price', body: 'See what charging costs and your cost per mile.', action: link('#/charging-settings', 'Open') },
    { key: 'gas', done: st.gas, title: 'Compare with a gas car', body: 'Enter a gas price and the economy of the car you would drive instead, to see what you save.', action: link('#/charging-settings', 'Open') },
    { key: 'placeprice', done: st.placePrice, title: 'Price the chargers you use often', body: 'A place with its own price per kWh makes every charge there exact. You can also set it from a charge on the Day view.', action: link('#/places', 'Places') },
    { key: 'phone', done: st.phone, title: 'Track trips with your phone', body: 'Trips start sooner and follow the roads more closely.', action: link('#/phone', 'Set up') },
    { key: 'service', done: st.maintenance, title: 'Add a service reminder', body: 'Tire rotation, cabin filter and so on, by distance or time.', action: link('#/maintenance', 'Add') },
    { key: 'backup', done: st.backup, title: 'Make a backup', body: 'Download a copy of your data so a failed disk cannot lose it.', action: link('#/backup', 'Open') });
  const skipped = new Set(st.skipped || []);
  for (const it of all) it.skipped = !it.done && skipped.has(it.key);
  return all;
}

const skip = (key, undo) => api(`/api/checklist/skip?key=${key}${undo ? '&undo=1' : ''}`, { method: 'POST' });

// Swipe a row left far enough and it is skipped; a short swipe springs back.
function swipeable(row, onSkip) {
  let x0 = null, dx = 0;
  row.addEventListener('pointerdown', (e) => { if (e.target.closest('a,button')) return; x0 = e.clientX; dx = 0; row.style.transition = 'none'; });
  row.addEventListener('pointermove', (e) => { if (x0 == null) return; dx = Math.min(0, e.clientX - x0); row.style.transform = `translateX(${dx}px)`; row.style.opacity = String(1 + dx / 300); });
  const end = () => {
    if (x0 == null) return;
    x0 = null;
    row.style.transition = 'transform .2s, opacity .2s';
    if (dx < -90) { row.style.transform = 'translateX(-110%)'; row.style.opacity = '0'; setTimeout(onSkip, 180); } else { row.style.transform = ''; row.style.opacity = ''; }
  };
  row.addEventListener('pointerup', end);
  row.addEventListener('pointercancel', end);
  row.style.touchAction = 'pan-y';
  return row;
}

// One suggestion. On Home (compact) it is only the title, its button and ×; the checklist page explains each and offers Undo
// for a skipped one. after() runs once it was skipped or brought back.
export function setupRow(it, compact, after = () => {}) {
  if (it.action) it.action.classList.add('sm');
  const doSkip = async () => { await skip(it.key).catch(() => null); after(); };
  const right = it.done ? chip('ok', 'done')
    : it.skipped ? h('span', { style: 'display:flex;gap:6px;align-items:center' }, chip('info', 'skipped'), h('button', { type: 'button', class: 'sec sm', onclick: async () => { await skip(it.key, true).catch(() => null); after(); } }, 'Undo'))
    : h('span', { style: 'display:flex;gap:6px;align-items:center' }, it.action || chip('work', 'to do'),
      h('button', { type: 'button', class: 'ico sm', 'aria-label': `Skip: ${it.title}`, title: 'Skip this suggestion', onclick: doSkip }, '×'));
  const row = h('div', { class: 'row', style: `align-items:${compact ? 'center' : 'flex-start'}${it.skipped ? ';opacity:.6' : ''}` },
    h('div', { class: 'grow', style: `flex-basis:${compact ? '10em' : '14em'}` }, h('b', {}, it.title), compact ? null : h('div', { class: 'muted' }, it.body)), right);
  return it.done || it.skipped ? row : swipeable(row, doSkip);
}

// After an update that finds trips and charging better, Home offers once to redo days recorded before it. Fix a day is
// otherwise out of sight.
function redetectCard() {
  const card = h('section', { class: 'card', role: 'alert' });
  const seen = async () => { await api('/api/repair/seen', { method: 'POST' }).catch(() => null); card.remove(); };
  card.append(h('h2', {}, 'Trips and charging are found better now'),
    h('p', { class: 'muted' }, 'This update changed how CarTrak finds trips and charging. Days recorded before it keep their old trips. If a day looks wrong, Fix a day can redo it from the stored readings, with a preview first.'),
    h('p', { class: 'seg' }, h('a', { class: 'btn', href: '#/repair', onclick: () => seen() }, 'Open Fix a day'), h('button', { type: 'button', class: 'sec', onclick: seen }, 'Not now')));
  return card;
}

// The Home cards: the Fix a day offer when there is one, and the next three suggestions unless all are done, skipped or hidden.
export async function setupCards(session) {
  const st = await api('/api/checklist').catch(() => null);
  if (!st) return [];
  const out = st.redetect ? [redetectCard()] : [];
  const card = h('section', { class: 'card' });
  const draw = (items) => {
    const left = items.filter((x) => !x.done && !x.skipped);
    if (!left.length) { card.remove(); return false; }
    const hide = async () => { await api('/api/checklist/hide', { method: 'POST' }).catch(() => null); card.remove(); };
    const again = async () => draw(setupItems(await api('/api/checklist'), session));
    card.replaceChildren(h('h2', {}, 'Suggested setup'),
      h('p', { class: 'muted' }, `${items.filter((x) => x.done).length} of ${items.length} done. All optional; swipe one away to skip it.`),
      ...left.slice(0, 3).map((x) => setupRow(x, true, again)),
      h('p', { class: 'seg' }, h('a', { class: 'btn sec sm', href: '#/setup' }, 'See all'), h('button', { type: 'button', class: 'sec sm', onclick: hide, title: 'It stays under More, Setup checklist' }, 'Hide')));
    return true;
  };
  if (st.hidden || !draw(setupItems(st, session))) return out;
  return [...out, card];
}
