import { h, num, temp } from './ui.js';

// "Battery and motor": the live temperatures and voltages the car reports. Anything the car does not send is simply
// left out, so a car with no motor temperature shows no motor temperature.
export function powerTile() {
  const rows = {};
  const row = (key, label) => {
    rows[key] = h('b', {}, '—');
    return h('div', { class: 'pw', hidden: true }, h('span', { class: 'muted' }, label), rows[key]);
  };
  const cells = [row('batt', 'Battery temp'), row('pack', 'Pack voltage'), row('flow', 'Battery power'), row('coolant', 'Coolant temp'), row('motorT', 'Motor temp'), row('motor', 'Motor'), row('v12', '12 V battery')];
  const el = h('div', { class: 'tile wide', style: 'order:99' }, h('div', { class: 'tl muted' }, 'Battery and motor'), h('div', { class: 'pwgrid' }, ...cells));
  const show = (key, text) => { rows[key].textContent = text || '—'; rows[key].parentElement.hidden = !text; };
  return {
    el,
    set(x) {
      show('batt', x.batteryTempC != null ? temp(x.batteryTempC) : '');
      show('pack', x.packVolts != null ? num(x.packVolts, 0, ' V') : '');
      show('motorT', x.motorTempC != null ? temp(x.motorTempC) : '');
      show('motor', x.motorVolts != null || x.motorAmps != null ? [x.motorVolts != null ? num(x.motorVolts, 0, ' V') : null, x.motorAmps != null ? num(x.motorAmps, 0, ' A') : null].filter(Boolean).join(' · ') : '');
      const kw = x.packAmps != null && x.packVolts != null ? (-x.packAmps * x.packVolts) / 1000 : null; // negative current is charge going in
      show('flow', kw != null && Math.abs(kw) >= 0.2 ? `${num(Math.abs(kw), 1, ' kW')} ${kw > 0 ? 'in' : 'out'}` : '');
      show('coolant', x.coolantC != null ? temp(x.coolantC) : '');
      show('v12', x.battery12v != null ? num(x.battery12v, 1, ' V') : '');
      el.hidden = !cells.some((c) => !c.hidden);
    },
  };
}

// Shrink a tile's text until it fits its box, so the live view never cuts a value short with "…".
export function fitText(node, minPx = 11) {
  node.style.fontSize = '';
  let size = parseFloat(getComputedStyle(node).fontSize);
  while (node.scrollWidth > node.clientWidth + 1 && size > minPx) {
    size -= 1;
    node.style.fontSize = `${size}px`;
  }
}
