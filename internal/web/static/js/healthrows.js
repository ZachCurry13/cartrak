import { h, chip, temp, pressure, dist, speed, ago } from './ui.js';

// Formats a row of the Vehicle health report. Units follow this device's metric/imperial choice.
const fmt = {
  volts: (v) => `${v.toFixed(v < 100 ? 2 : 0)} V`,
  pct: (v) => `${Math.round(v * 10) / 10}%`,
  kwh: (v) => `${v.toFixed(1)} kWh`,
  amps: (v) => `${v.toFixed(2)} A`,
  seconds: (v) => `${Math.round(v)} s`,
  tempC: (v) => temp(v),
  km: (v) => dist(v, 0),
  kpa: (v) => pressure(v),
  kph: (v) => speed(v),
  minutes: (v) => (v >= 60 ? `${Math.floor(v / 60)} h ${Math.round(v % 60)} min` : `${Math.round(v)} min`),
  duration: (v) => (v >= 120 ? `${Math.round(v / 60)} min` : `${Math.round(v)} s`),
  nm: (v) => `${Math.round(v)} N·m`,
  accel: (v) => `${v.toFixed(2)} m/s²`,
  degps: (v) => `${v.toFixed(2)} °/s`,
  deg: (v) => `${Math.round(v)}°`,
  rpm: (v) => `${Math.round(v)} rpm`,
};

export const format = (kind, v) => (fmt[kind] ? fmt[kind](v) : String(v));

export const value = (row) => (row.kind === 'text' || row.num == null ? row.text || '' : format(row.kind, row.num));

// Ford updates some values only when the car wakes, so say when a value is old.
export function age(row) {
  const t = row.at ? Date.parse(row.at) / 1000 : 0;
  return t && Date.now() / 1000 - t > 7200 ? `updated ${ago(t)}` : '';
}

const STATE = { warn: ['work', 'Check'], bad: ['bad', 'Act'] };

export function rowEl(row) {
  const s = STATE[row.state];
  const old = age(row);
  return h('div', { class: 'hrow' },
    h('span', { class: 'hl' }, row.label),
    h('span', { class: 'hv' }, h('b', {}, value(row)), s ? ' ' : '', s ? chip(s[0], s[1]) : null),
    row.note ? h('span', { class: 'hn muted' }, row.note) : null,
    old ? h('span', { class: 'hn muted' }, old) : null);
}
