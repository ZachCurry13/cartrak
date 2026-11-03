// A car that has sent nothing for a day: a parked car sleeps for hours, but a day or more is something else, most often a weak 12 V
// battery (the car's modem needs it) or no cell signal. Takes the answer of /api/status; null while the car is reporting normally.
import { h, ago } from './ui.js';

const SILENT_AFTER = 24 * 3600;

export function silentNote(status) {
  const last = status && status.carLastReport;
  if (!last || !status.poller || !status.poller.linked || Date.now() / 1000 - last < SILENT_AFTER) return null;
  return h('div', { class: 'banner' }, h('b', {}, `The car last reported ${ago(last)}. `),
    'A parked car sleeps, but a day or more can mean a weak 12 V battery or no cell signal. ', h('a', { href: '#/health' }, 'Health'), ' shows its last 12 V reading.');
}
