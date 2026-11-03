// "Match another mileage app": a GPS app and the car's odometer rarely agree (GPS reads a little short, an odometer a little long).
// Type what the other app measured for this trip and CarTrak sets one correction for every distance it takes from the odometer.
import { h, isMetric } from './ui.js';
import { api } from './api.js';

export function odometerCalibrator(t) {
  if (t.startOdo == null || t.endOdo == null) return null;
  const metric = isMetric();
  const input = h('input', { type: 'number', 'data-unit': 'dist', step: '0.1', min: 0.1, inputmode: 'decimal', 'aria-label': 'What the other app measured' });
  const msg = h('p', { class: 'muted', role: 'status' });
  const go = h('button', { class: 'sec', onclick: async () => {
    const v = Number(input.value);
    if (!(v > 0)) { msg.textContent = 'Enter what the other app said.'; return; }
    try {
      const r = await api(`/api/trips/${t.id}/calibrate`, { method: 'POST', body: { km: metric ? v : v * 1.609344 } });
      msg.textContent = `Odometer correction set to ${r.factor.toFixed(3)}. New trips use it; use Fix a day to apply it to older ones.`;
    } catch (e) { msg.textContent = e.message; }
  } }, 'Set the correction');
  return h('details', { class: 'curvebox', style: 'margin:8px 0 0' }, h('summary', { class: 'muted' }, 'Match another mileage app'),
    h('p', { class: 'muted' }, 'Type what your other app measured for this trip and CarTrak sets one correction for every distance it takes from the odometer.'),
    h('div', { class: 'row', style: 'border:0;padding:0' }, input, go), msg);
}
