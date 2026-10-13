import { h, stat, chip, dist, speed, temp, pressure, pct, num, ago, duration, when } from '../ui.js';
import { api } from '../api.js';
import { notices } from '../notice.js';
import { isSaver, effective } from '../mode.js';
import { cachedEx } from '../offline.js';

// Find a value in the flattened "extra" map by metric name (suffix match).
export function pick(extra, ...names) {
  for (const nm of names) {
    const l = nm.toLowerCase().replace(/_/g, '');
    for (const [k, v] of Object.entries(extra || {})) {
      const lk = k.toLowerCase().replace(/_/g, '');
      if (lk === l || lk.endsWith('.' + l) || lk === l + '.value' || lk.endsWith('.' + l + '.value')) return v;
    }
  }
  return null;
}

const mapLink = (lat, lon) => `https://www.google.com/maps?q=${lat},${lon}`;

function card(title, ...kids) { return h('section', { class: 'card' }, h('h2', {}, title), ...kids); }

function kv(rows) {
  return h('div', { class: 'kv' }, rows.filter((r) => r[1] != null).flatMap(([k, v]) => [h('span', {}, k), h('span', {}, v)]));
}

export default async function dashboard(root, _args, session) {
  const box = h('div', { class: 'dash' });
  root.append(box);
  let cache = null; // status, last trip, checklist: refetched every time only in Normal mode
  let full = !isSaver(); // Data saver starts with the small summary until asked for more
  async function load() {
    const { value: latest, staleSince } = await cachedEx(`latest${full ? '' : '-s'}`, () => api(`/api/latest${full ? '' : '?summary=1'}`));
    if (staleSince) cache = cache || { status: { poller: { linked: true }, canLink: false }, trips: [], ob: null }; // offline: nothing to refetch
    else if (!cache || effective() === 'normal') {
      const [status, trips, ob] = await Promise.all([api('/api/status'), api('/api/trips?limit=1'),
        session.me.role === 'admin' ? api('/api/onboarding') : null]);
      cache = { status, trips, ob };
    }
    const { status, trips, ob } = cache;
    const out = [...(staleSince ? [h('div', { class: 'banner' }, h('b', {}, 'You are offline. '), `Showing the last readings saved on this device (${ago(Math.floor(staleSince / 1000))}).`)] : []), ...notices(session)];
    if (ob) {
      if (!ob.dismissed) out.push(h('a', { class: 'card', href: '#/start', style: 'display:block;text-decoration:none' }, h('b', {}, 'Getting started'), h('div', { class: 'muted' }, 'Finish setting up CarTrak: connect Ford, install on your phone, add trip rules →')));
    }
    if (!status.poller.linked) {
      out.push(h('div', { class: 'banner' }, h('b', {}, 'Ford account not linked. '),
        status.canLink && session.me.role === 'admin'
          ? (status.configured ? h('a', { class: 'btn', href: '#/connect', style: 'margin-top:8px' }, 'Connect Ford account') : h('a', { class: 'btn', href: '#/admin', style: 'margin-top:8px' }, 'Enter your Ford settings'))
          : 'Ask the admin to connect it.'));
    } else if (status.poller.lastError) {
      out.push(h('div', { class: 'banner' }, `Last update failed: ${status.poller.lastError}`));
    }
    if (!latest) {
      out.push(h('p', { class: 'muted' }, 'No readings yet. Data appears after the first successful sync.'));
      return box.replaceChildren(...out);
    }
    const x = latest.extra || {};
    const soc = latest.socPct;
    out.push(card('Battery',
      h('div', { class: 'grid' }, stat('Charge', pct(soc)), stat('Range', dist(latest.rangeKm, 0)),
        stat('Energy left', num(pick(x, 'XEV_BATTERY_ENERGY_REMAINING'), 1, ' kWh')),
        stat('Battery temp', temp(pick(x, 'XEV_BATTERY_TEMPERATURE')))),
      h('div', { class: 'bar', style: 'margin-top:10px' }, h('i', { style: `width:${Math.max(0, Math.min(100, soc || 0))}%` }))));

    const charging = /charg/i.test(latest.chargeState || '') && !/not|complete|stop/i.test(latest.chargeState || '');
    out.push(card('Charging', h('div', { style: 'margin-bottom:8px' }, chip(charging ? 'ok' : 'work', latest.chargeState || 'Unknown'), ' ', chip(latest.plug === 'Connected' ? 'ok' : 'bad', latest.plug || 'Unknown')),
      kv([['Power', latest.chargeKw != null ? num(latest.chargeKw, 1, ' kW') : null],
        ['Voltage', pick(x, 'XEV_BATTERY_CHARGER_VOLTAGE_OUTPUT') != null ? num(pick(x, 'XEV_BATTERY_CHARGER_VOLTAGE_OUTPUT'), 0, ' V') : null],
        ['Current', pick(x, 'XEV_BATTERY_CHARGER_CURRENT_OUTPUT') != null ? num(pick(x, 'XEV_BATTERY_CHARGER_CURRENT_OUTPUT'), 0, ' A') : null],
        ['Time to full', pick(x, 'XEV_BATTERY_TIME_TO_FULL_CHARGE') != null ? duration(0, pick(x, 'XEV_BATTERY_TIME_TO_FULL_CHARGE') * 60) : null],
        ['Station type', pick(x, 'XEV_CHARGE_STATION_POWER_TYPE')]])));

    out.push(card('Vehicle',
      kv([['Ignition', latest.ignition || '—'], ['Gear', pick(x, 'GEAR_LEVER_POSITION')], ['Odometer', dist(latest.odometerKm, 0)],
        ['Outside', temp(latest.outsideC)], ['Speed', speed(latest.speedKph)], ['12V battery', latest.battery12v != null ? num(latest.battery12v, 1, ' V') : null],
        ['Locks', latest.locked == null ? null : latest.locked ? 'Locked' : 'Unlocked'], ['Parking brake', pick(x, 'PARKING_BRAKE_STATUS')]])));

    if (latest.lat != null && latest.lon != null) {
      out.push(card('Location', h('a', { href: mapLink(latest.lat, latest.lon), target: '_blank', rel: 'noopener' }, `${latest.lat.toFixed(5)}, ${latest.lon.toFixed(5)}`),
        h('p', { class: 'muted' }, `Heading ${pick(x, 'HEADING', 'direction', 'COMPASS_DIRECTION') ?? '—'} · seen ${ago(latest.ts)}`)));
    }

    const tires = (latest.tires || []).some((t) => t != null) ? latest.tires :['FL', 'FR', 'RL', 'RR'].map((k) => pick(x, `TIRE_PRESSURE_${k}`));
    if (tires.some((t) => t != null) || pick(x, 'tirePressureWarning') != null) {
      out.push(card('Tires', h('div', { class: 'grid' }, ['Front L', 'Front R', 'Rear L', 'Rear R'].map((l, i) => stat(l, pressure(tires[i])))),
        pick(x, 'tirePressureWarning') ? h('p', { class: 'chip bad' }, 'Low pressure warning') : null));
    }

    const t = trips[0];
    if (t) out.push(card(t.open ? 'Trip in progress' : 'Last trip', h('a', { href: `#/trip/${t.id}` }, `${dist(t.distanceKm)} · ${t.category}`),
      h('p', { class: 'muted' }, `${when(t.startTs, t.tz)} · ${duration(t.startTs, t.endTs)}`)));
    if (!full) {
      out.push(h('div', { class: 'card' }, h('p', { class: 'muted' }, 'Data saver is on: showing the essentials. Tires, battery details and other fields are skipped.'),
        h('button', { class: 'sec', onclick: () => { full = true; load(); } }, 'Load full details once')));
    }
    box.replaceChildren(...out);
  }
  await load();
  return load;
}
