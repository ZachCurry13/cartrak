import { h, chip } from './ui.js';
import { api } from './api.js';
import { startTour } from './tourui.js';

// The setup checklist: suggestions, none required. Home shows the next few until the person hides the card; the whole list
// stays under More, Setup checklist. Installing and the tour are known only on this phone; the rest comes from /api/checklist.
const standalone = () => window.matchMedia('(display-mode: standalone)').matches || navigator.standalone === true;
const toured = () => { try { return localStorage.getItem('tourDone') === '1'; } catch (e) { return false; } };

export function setupItems(st, session) {
  const admin = session.me.role === 'admin';
  const link = (href, text) => h('a', { class: 'btn sec', href }, text);
  const all = [
    { done: standalone(), title: 'Put CarTrak on your home screen', body: 'It opens like an app and keeps you signed in. In the browser menu choose Install app or Add to Home screen.' },
    { done: toured(), title: 'Take the two-minute tour', body: 'A quick walk through each part of CarTrak.', action: h('button', { type: 'button', class: 'sec', onclick: () => startTour(session) }, 'Start') },
    { done: st.alerts, title: 'Turn on alerts on this phone', body: 'Charging done, plug in tonight, a low tire and more.', action: link('#/notifications', 'Choose') },
  ];
  if (admin) all.push(
    { done: st.home, title: 'Name Home', body: 'Trips read "Home → Work", and charging at home is priced and counted as home.', action: link('#/places', 'Places') },
    { done: st.price, title: 'Set your electricity price', body: 'See what charging costs and your cost per mile.', action: link('#/charging-settings', 'Open') },
    { done: st.gas, title: 'Compare with a gas car', body: 'Enter a gas price and the economy of the car you would drive instead, to see what you save.', action: link('#/charging-settings', 'Open') },
    { done: st.placePrice, title: 'Price the chargers you use often', body: 'A place with its own price per kWh makes every charge there exact. You can also set it from a charge on the Day view.', action: link('#/places', 'Places') },
    { done: st.phone, title: 'Track trips with your phone', body: 'Trips start sooner and follow the roads more closely.', action: link('#/phone', 'Set up') },
    { done: st.maintenance, title: 'Add a service reminder', body: 'Tire rotation, cabin filter and so on, by distance or time.', action: link('#/maintenance', 'Add') },
    { done: st.backup, title: 'Make a backup', body: 'Download a copy of your data so a failed disk cannot lose it.', action: link('#/backup', 'Open') });
  return all;
}

// One suggestion. On Home (compact) it is only the title and its button; the checklist page explains each.
export function setupRow(it, compact) {
  if (it.action) it.action.classList.add('sm');
  return h('div', { class: 'row', style: `align-items:${compact ? 'center' : 'flex-start'}` },
    h('div', { class: 'grow', style: `flex-basis:${compact ? '10em' : '14em'}` }, h('b', {}, it.title), compact ? null : h('div', { class: 'muted' }, it.body)),
    it.done ? chip('ok', 'done') : it.action || chip('work', 'to do'));
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

// The Home cards: the Fix a day offer when there is one, and the next three suggestions unless all are done or hidden.
export async function setupCards(session) {
  const st = await api('/api/checklist').catch(() => null);
  if (!st) return [];
  const out = st.redetect ? [redetectCard()] : [];
  const items = setupItems(st, session);
  const left = items.filter((x) => !x.done);
  if (st.hidden || !left.length) return out;
  const card = h('section', { class: 'card' });
  const hide = async () => { await api('/api/checklist/hide', { method: 'POST' }).catch(() => null); card.remove(); };
  card.append(h('h2', {}, 'Suggested setup'),
    h('p', { class: 'muted' }, `${items.length - left.length} of ${items.length} done. All optional.`),
    ...left.slice(0, 3).map((x) => setupRow(x, true)),
    h('p', { class: 'seg' }, h('a', { class: 'btn sec sm', href: '#/setup' }, 'See all'), h('button', { type: 'button', class: 'sec sm', onclick: hide, title: 'It stays under More, Setup checklist' }, 'Hide')));
  return [...out, card];
}
