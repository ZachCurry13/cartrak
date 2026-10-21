import { h, chip, isMetric, setMetric, getTheme, setTheme, applyTheme } from '../ui.js';
import { api } from '../api.js';
import { MODES, getPref, setPref } from '../mode.js';
import { startTour } from '../tourui.js';

// A row of buttons where one is chosen.
function pick(label, options, value, onpick) {
  const seg = h('div', { class: 'seg', role: 'group', 'aria-label': label });
  const draw = (v) => seg.replaceChildren(...options.map(([k, t]) => h('button', { class: k === v ? '' : 'sec', 'aria-pressed': String(k === v), onclick: () => { onpick(k); draw(k); } }, t)));
  draw(value);
  return h('div', { class: 'setting' }, h('label', {}, label), seg);
}

function step(done, title, body, action) {
  return h('div', { class: 'row', style: 'align-items:flex-start' },
    h('div', { class: 'grow', style: 'flex-basis:14em' }, h('b', {}, title), h('div', { class: 'muted' }, body)),
    action || null, done ? chip('ok', 'done') : chip('work', 'to do'));
}

// Help for everyone: the settings people reach for most, a short checklist, and what each part of the app is for.
export default async function help(root, _a, session) {
  const admin = session.me.role === 'admin';
  let linked = true;
  if (admin) { const st = await api('/api/status').catch(() => null); linked = !st || st.poller.linked; }

  const quick = h('section', { class: 'card' }, h('h2', {}, 'Quick settings'),
    h('p', { class: 'muted' }, 'These apply to this device only.'),
    pick('Units', [['imperial', 'Miles, °F, psi'], ['metric', 'Kilometres, °C, kPa']], isMetric() ? 'metric' : 'imperial', (v) => { setMetric(v === 'metric'); dispatchEvent(new Event('hashchange')); }),
    pick('Theme', [['auto', 'Automatic'], ['light', 'Light'], ['dark', 'Dark']], getTheme(), (v) => { setTheme(v); applyTheme(); }),
    pick('Data use', Object.entries(MODES), getPref(), (v) => setPref(v)),
    h('p', {}, h('a', { href: '#/display' }, 'More display settings')));

  const checklist = h('section', { class: 'card' }, h('h2', {}, 'Getting set up'),
    admin ? step(linked, 'Connect your car', 'CarTrak reads your car from Ford. Until it is connected there is nothing to show.', h('a', { class: 'btn', href: linked ? '#/ford' : '#/start' }, 'Open')) : null,
    h('div', { class: 'row', style: 'align-items:flex-start' }, h('div', { class: 'grow', style: 'flex-basis:14em' }, h('b', {}, 'Setup checklist'), h('div', { class: 'muted' }, 'Suggested next steps, all optional: install, alerts, places, prices and more.')),
      h('a', { class: 'btn sec', href: '#/setup' }, 'Open')));

  const tabs = h('section', { class: 'card' }, h('h2', {}, 'What each part is for'),
    ...[['Home', 'Battery, range, what the car is doing, the day so far and the month.'], ['Trips', 'Every drive, a day-by-day view, and trip labels.'], ['Live', 'A dock screen that follows driving, charging and parking.'],
      ['Charging', 'Sessions, cost, speed and charging by place.'], ['Reports', 'More, Reports: the year, statistics, efficiency insights, mileage and exports.'], ['Health', 'Tires, the 12 V battery and anything the car reports as wrong.'], ['Data', 'Everything the car sends, in plain words.'],
      ['More', 'Settings, people, cars, backups and this help.']].map(([t, b]) => h('div', { class: 'row' }, h('b', {}, t), h('span', { class: 'muted grow', style: 'flex-basis:14em' }, b))),
    h('p', {}, h('a', { href: '#/guide' }, 'Everything CarTrak can do'), ' · ', h('a', { href: '#/status' }, 'Is everything working?'), ' · ', h('a', { href: '#/about' }, "What's new")));

  const tour = h('section', { class: 'card' }, h('h2', {}, 'Guided tour'), h('p', { class: 'muted' }, 'A two-minute walk through each part of CarTrak. Tap the ? at the top any time to run it again.'), h('p', {}, h('button', { type: 'button', onclick: () => startTour(session) }, 'Start the tour')));
  root.append(h('div', { class: 'page-title' }, h('h2', {}, 'Help')), tour, quick, checklist, tabs);
}
