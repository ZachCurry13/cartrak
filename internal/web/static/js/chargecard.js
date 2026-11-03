import { h, temp } from './ui.js';
import { api } from './api.js';
import { PRESETS, getTarget, setTarget, chargeEta, minutesText, clockAt } from './charge.js';

// "Time until 80% / 90% / 100% / another percent", for a car that is charging. Pass the car's own
// time-to-full figure (minutes) when it reports one.
const LEVEL = { l1: 'wall outlet (Level 1)', l2: 'Level 2', dc: 'fast' };

// What the estimate rests on, in words: the kind of charger, how many of this car's earlier charges it learned from, and for a fast
// charger what the battery temperature means.
function because(r) {
  const out = [`Based on ${r.sessions} of your earlier ${LEVEL[r.level] || ''} charges, including how charging slows near full.`.replace('  ', ' ')];
  if (r.level === 'dc' && r.batteryC != null) {
    if (r.preconditioned) out.push(`The battery is warm for the weather (${temp(r.batteryC)}), so it can take the full speed.`);
    else if (r.tempClass === 'cold') out.push(`The battery is cold (${temp(r.batteryC)}), so fast charging runs slower until it warms.`);
    else out.push(`The battery is at ${temp(r.batteryC)}.`);
  }
  return out.join(' ');
}

export function chargeEtaBox(latest, fordMinutesToFull) {
  const box = h('div', { class: 'eta' });
  const input = h('input', { type: 'number', 'data-unit': '%', min: 20, max: 100, inputmode: 'numeric', 'aria-label': 'Other percent', style: 'width:5.5em' });
  function draw() {
    const t = getTarget();
    const other = !PRESETS.includes(t);
    input.value = other ? t : '';
    input.hidden = !other; // an empty box only appears once Other is chosen
    const pick = (v) => () => { setTarget(v); draw(); };
    const r = chargeEta(latest, t, fordMinutesToFull);
    let line = 'Not enough data for an estimate.';
    if (r && r.done) line = `Already at ${t}% or above.`;
    else if (r && r.idle) line = 'Not charging right now.';
    else if (r) line = `About ${minutesText(r.minutes)} (around ${clockAt(r.minutes)}).${r.source === 'ford' ? ' From the car.' : ' An estimate from the current charging power.'}`;
    const lineEl = h('b', {}, line), detailEl = h('p', { class: 'muted' });
    box.replaceChildren(
      h('div', { class: 'muted' }, 'Time until'),
      h('p', { class: 'seg' }, ...PRESETS.map((p) => h('button', { class: p === t ? 'on' : '', onclick: pick(p) }, `${p}%`)),
        h('button', { class: other ? 'on' : '', onclick: () => { input.hidden = false; input.focus(); } }, 'Other'), input),
      h('div', {}, lineEl), detailEl);
    if (r && !r.done && !r.idle) learned(t, lineEl, detailEl);
  }
  // The estimate learned from the car's own past charges replaces the plain one when there are enough of them.
  async function learned(t, lineEl, detailEl) {
    let r;
    try { r = await api(`/api/charges/eta?target=${t}`); } catch (e) { return; }
    if (!r || !r.learned || getTarget() !== t || !lineEl.isConnected) return;
    lineEl.textContent = `About ${minutesText(r.minutes)} (around ${clockAt(r.minutes)}).`;
    detailEl.textContent = because(r);
  }
  input.addEventListener('change', () => { if (input.value) { setTarget(Number(input.value)); draw(); } });
  draw();
  return box;
}
