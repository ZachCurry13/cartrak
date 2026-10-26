import { h } from './ui.js';
import { PRESETS, getTarget, setTarget, chargeEta, minutesText, clockAt } from './charge.js';

// "Time until 80% / 90% / 100% / another percent", for a car that is charging. Pass the car's own
// time-to-full figure (minutes) when it reports one.
export function chargeEtaBox(latest, fordMinutesToFull) {
  const box = h('div', { class: 'eta' });
  const input = h('input', { type: 'number', min: 20, max: 100, inputmode: 'numeric', 'aria-label': 'Other percent', style: 'width:5.5em' });
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
    box.replaceChildren(
      h('div', { class: 'muted' }, 'Time until'),
      h('p', { class: 'seg' }, ...PRESETS.map((p) => h('button', { class: p === t ? 'on' : '', onclick: pick(p) }, `${p}%`)),
        h('button', { class: other ? 'on' : '', onclick: () => { input.hidden = false; input.focus(); } }, 'Other'), input),
      h('div', {}, h('b', {}, line)));
  }
  input.addEventListener('change', () => { if (input.value) { setTarget(Number(input.value)); draw(); } });
  draw();
  return box;
}
