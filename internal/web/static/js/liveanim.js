// Ambient motion for the Live page that follows what the car is doing. The car reports only every 30 s to 10 min, so nothing here
// pretends to be live: each effect shows a state (driving, charging, asleep, locked, remote start) and stops when the last reading is
// too old for that state to still be true. Data saver and "reduce motion" turn it all off.
import { h } from './ui.js';
import { isSaver } from './mode.js';

const FRESH = { driving: 90, charging: 360 }; // seconds a reading may be old and the state still be shown
const ASLEEP = 20 * 60; // parked with no new reading for this long: the car is asleep
const ALL = ['fx-driving', 'fx-charging', 'fx-parked', 'fx-asleep', 'fx-locked', 'fx-remote', 'fx-heat', 'fx-cool', 'fx-mild'];
const spans = (cls, n) => Array.from({ length: n }, (_, i) => h('span', { class: cls, style: `--i:${i}` }));

export function liveFx(dock) {
  const layer = h('div', { class: 'fx', 'aria-hidden': 'true' },
    ...spans('fx-z', 3), ...spans('fx-dot', 5), ...spans('fx-air', 6), h('span', { class: 'fx-lock' }));
  dock.prepend(layer);
  const s = { mode: 'parked', age: Infinity, locked: false, remote: false, outsideC: null };
  function apply() {
    const cls = [];
    if (!isSaver()) {
      if (s.mode === 'driving' && s.age <= FRESH.driving) cls.push('fx-driving');
      else if (s.mode === 'charging' && s.age <= FRESH.charging) cls.push('fx-charging');
      else if (s.mode === 'parked') cls.push(s.age > ASLEEP ? 'fx-asleep' : 'fx-parked');
      if (s.locked && s.mode === 'parked') cls.push('fx-locked');
      if (s.remote) cls.push('fx-remote', s.outsideC != null && s.outsideC < 15 ? 'fx-heat' : s.outsideC != null && s.outsideC > 24 ? 'fx-cool' : 'fx-mild');
    }
    dock.classList.remove(...ALL);
    dock.classList.add(...cls);
  }
  return { set(patch) { Object.assign(s, patch); apply(); } };
}
