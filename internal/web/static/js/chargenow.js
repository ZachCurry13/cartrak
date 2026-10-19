import { h, chip, nice, num, pct, ago } from './ui.js';
import { api } from './api.js';
import { chargeEtaBox } from './chargecard.js';

// "Right now" at the top of the Charging page: whether the car is plugged in, whether it is taking charge, and how
// fast, with the time to a target when it is. It refreshes on its own while the page is open.
export function chargeNow() {
  const el = h('section', { class: 'card' }, h('h2', {}, 'Right now'), h('p', { class: 'muted' }, 'Checking the car…'));
  let timer = 0;
  async function load() {
    let x;
    try { x = await api('/api/latest?summary=1'); } catch (e) { return; }
    if (!x) return;
    const plugged = /^connected$/i.test(x.plug || '') || (/connect/i.test(x.plug || '') && !/dis|not/i.test(x.plug || ''));
    const taking = (x.chargeKw || 0) > 0.2 || (/charging/i.test(x.chargeState || '') && !/not|complete|stop/i.test(x.chargeState || ''));
    el.replaceChildren(...[
      h('h2', {}, 'Right now'),
      h('p', {}, chip(taking ? 'ok' : 'work', taking ? 'Charging' : (nice(x.chargeState) || 'Not charging')), ' ',
        chip(plugged ? 'ok' : 'work', plugged ? 'Plugged in' : 'Not plugged in'), ' ', h('span', { class: 'muted' }, x.ts ? `reading from ${ago(x.ts)}` : '')),
      h('div', { class: 'grid' },
        h('div', { class: 'stat' }, h('div', { class: 'v' }, pct(x.socPct)), h('div', { class: 'l' }, 'Battery')),
        h('div', { class: 'stat' }, h('div', { class: 'v' }, taking && x.chargeKw != null ? num(x.chargeKw, 1, ' kW') : '—'), h('div', { class: 'l' }, 'Power')),
        x.energyKwh != null ? h('div', { class: 'stat' }, h('div', { class: 'v' }, num(x.energyKwh, 1, ' kWh')), h('div', { class: 'l' }, 'Energy left')) : null),
      taking ? chargeEtaBox(x, null) : null,
    ].filter(Boolean));
  }
  load();
  timer = setInterval(load, 30000);
  return { el, stop: () => clearInterval(timer) };
}
