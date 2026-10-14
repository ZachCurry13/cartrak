import { h, chip, when, ago } from '../ui.js';
import { api } from '../api.js';

// Parked watch (experimental): jolts and alarms on a parked car, and how often the car reports at all.
export default async function parkedwatch(root, _a, session) {
  const admin = session.me.role === 'admin';
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Parked watch')),
    h('p', { class: 'muted' }, 'Experimental. A parked car\'s motion sensors read close to zero. CarTrak checks every reading of a parked car, and can alert you if the motion jumps (a bump from another car\'s door, for example) or the alarm goes off. The car only reports now and then, so a short knock can be missed. This page shows what CarTrak has seen, so you can judge how well it works.'), box);

  async function load() {
    const d = await api('/api/parked-watch');
    const st = d.stats;
    box.replaceChildren(...[
      h('section', { class: 'card' }, h('h2', {}, 'Status'),
        h('p', {}, d.on ? chip('ok', `Watching: bump level ${d.level} m/s²`) : chip('work', 'Switched off in Settings'), ' ',
          h('span', { class: 'muted' }, 'Turn on the alerts under More, Notifications (Bump while parked, Car alarm). Change the level in More, Settings.')),
        h('div', { class: 'grid' },
          h('div', { class: 'stat' }, h('div', { class: 'v' }, String(st.checked)), h('div', { class: 'l' }, `parked readings checked${st.since ? ` since ${when(st.since)}` : ''}`)),
          h('div', { class: 'stat' }, h('div', { class: 'v' }, st.maxSeen ? `${st.maxSeen.toFixed(2)}` : '—'), h('div', { class: 'l' }, st.maxTs ? `strongest m/s², ${ago(st.maxTs)}` : 'strongest motion seen (m/s²)'))),
        admin ? h('p', {}, h('button', { class: 'sec', onclick: async () => { if (confirm('Clear the totals and the list and start again?')) { await api('/api/parked-watch/reset', { method: 'POST' }); load(); } } }, 'Start a fresh trial')) : null),
      h('section', { class: 'card' }, h('h2', {}, 'What it noticed'),
        ...(d.events.length ? d.events.map((e) => h('div', { class: 'row' }, h('div', { class: 'grow' }, h('b', {}, e.kind === 'alarm' ? 'Alarm' : 'Motion'), ' ', h('span', { class: 'muted' }, e.note)), h('span', { class: 'muted' }, when(e.ts))))
          : [h('p', { class: 'muted' }, 'Nothing above a light touch yet. A bump that was never reported by the car cannot appear here.')]))].filter(Boolean));
  }
  await load();
}
