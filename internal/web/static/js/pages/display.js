import { h, isMetric, setMetric, getTheme, setTheme, getTimeMode, setTimeMode } from '../ui.js';
import { MODES, getPref, setPref } from '../mode.js';

// One labelled choice with optional help text.
function setting(id, label, select, help) {
  select.id = id;
  return h('div', { class: 'setting' }, h('label', { for: id }, label), select, help ? h('p', { class: 'muted' }, help) : null);
}

function choice(options, value, onchange) {
  const s = h('select', {}, ...options.map(([v, t]) => h('option', { value: v }, t)));
  s.value = value;
  s.addEventListener('change', () => onchange(s.value));
  return s;
}

// These choices apply to this device only.
export default async function display(root) {
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Display and data')),
    h('p', { class: 'muted' }, 'These choices apply to this device only.'),
    h('section', { class: 'group' }, h('div', { class: 'menu' },
      setting('s-un', 'Units', choice([['imperial', 'Miles, °F, psi'], ['metric', 'Kilometres, °C, kPa']], isMetric() ? 'metric' : 'imperial', (v) => setMetric(v === 'metric'))),
      setting('s-th', 'Theme', choice([['auto', 'Match my phone'], ['light', 'Light'], ['dark', 'Dark']], getTheme(), setTheme)),
      setting('s-tm', 'Trip times', choice([['trip', 'Where the trip happened'], ['device', 'This device']], getTimeMode(), setTimeMode),
        'Trips show the local time where they started, with the zone name.'),
      setting('s-dm', 'Data use', choice(Object.entries(MODES), getPref(), (v) => { setPref(v); location.reload(); }),
        'Normal refreshes every 30 s. Live pushes each new reading. Data saver shows the essentials, refreshes every 5 min, and loads big things only when you tap.'))));
}
