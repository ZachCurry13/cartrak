import { remoteChip } from '../remotestart.js';
import { homeCards } from '../homecards.js';
import { h, stat, chip, nice, dist, speed, temp, pressure, pct, num, ago, duration, when } from '../ui.js';
import { api } from '../api.js';
import { notices } from '../notice.js';
import { isSaver, effective } from '../mode.js';
import { cachedEx } from '../offline.js';
import { healthStrip } from '../healthstrip.js';
import { doorsCard, healthCard } from '../cards.js';
import { createMap, mapConfig } from '../map.js';
import { reserveKm, getReserve } from '../charge.js';
import { chargeEtaBox } from '../chargecard.js';
import { vacationNote } from '../vacation.js';
import { setupCards } from '../setup.js';
import { serviceCards } from '../servicecard.js';
import { shortOdometer } from '../tripdistance.js';
import { isCharging } from '../chargestate.js';
import { bandRow } from '../rangeband.js';
import { batteryRing } from '../ring.js';
import { googleMapsUrl, mapButtons } from '../maplinks.js';
import { effSpan, kwhText } from '../energy.js';
import { greeting } from '../seasons.js';

let homeMap = null; // kept between refreshes
let homeLink = null; // the map is a link: tap it to open the spot in Google Maps

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


// How far the car can go before the battery falls to this device's reserve percent.
function reserveRow(latest) {
  const km = reserveKm(latest);
  return km == null ? null : h('p', { style: 'margin:10px 0 0' }, h('span', { class: 'muted' }, `Until ${getReserve()}%: `), h('b', {}, km > 0 ? dist(km, 0) : 'already below'));
}

// The last seven days at a glance. Tap the efficiency to change its unit.
function weekCard(w) {
  const all = Object.values(w);
  const km = all.reduce((n, x) => n + x.distanceKm, 0);
  const trips = all.reduce((n, x) => n + x.trips, 0);
  const kwh = all.reduce((n, x) => n + x.kwh, 0);
  return card('Last 7 days', h('div', { class: 'grid' }, stat('Distance', dist(km, 0)), stat('Trips', trips),
    ...(kwh > 0 ? [stat('Energy used', kwhText(kwh)), stat('Efficiency', effSpan(all.reduce((n, x) => n + x.energyKm, 0), kwh))] : [])),
    h('p', { style: 'margin:10px 0 0' }, h('a', { href: '#/trips' }, 'See all trips')));
}

function card(title, ...kids) { return h('section', { class: 'card' }, h('h2', {}, title), ...kids); }

function wrappedTeaser() {
  const d = new Date(new Date().getFullYear(), new Date().getMonth() - 1, 1);
  const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
  return h('a', { class: 'wteaser w-teal', href: `#/wrapped/${key}`, style: 'min-height:110px;margin-bottom:12px' }, h('span', { class: 'kick' }, 'Wrapped'), h('b', {}, `Your ${d.toLocaleDateString([], { month: 'long' })} is ready ✨`), h('span', {}, 'Tap to play ▶'));
}

// Ford sends some values only when the car wakes, so say when a value is old.
function oldNote(x, metric) {
  const t = Date.parse(pick(x, `${metric}.updateTime`) || '') / 1000;
  return t && Date.now() / 1000 - t > 7200 ? h('p', { class: 'muted' }, `These battery values are from ${ago(t)}; the car reports them when it wakes up.`) : null;
}

function kv(rows) {
  return h('div', { class: 'kv' }, rows.filter((r) => r[1] != null).flatMap(([k, v]) => [h('span', {}, k), h('span', {}, v)]));
}

export default async function dashboard(root, _args, session) {
  const mc = await mapConfig();
  const box = h('div', { class: 'dash' });
  root.append(box);
  let cache = null; // status, last trip, checklist: refetched every time only in Normal mode
  let full = !isSaver(); // Data saver starts with the small summary until asked for more
  async function load() {
    const { value: latest, staleSince } = await cachedEx(`latest${full ? '' : '-s'}`, () => api(`/api/latest${full ? '' : '?summary=1'}`));
    if (staleSince) cache = cache || { status: { poller: { linked: true }, canLink: false }, trips: [], ob: null }; // offline: nothing to refetch
    else if (!cache || effective() === 'normal') {
      const [status, trips, ob, health] = await Promise.all([api('/api/status'), api('/api/trips?limit=1'),
        session.me.role === 'admin' ? api('/api/onboarding') : null, api('/api/health').catch(() => null)]);
      const week = await api(`/api/summary?from=${Math.floor(Date.now() / 1000) - 7 * 86400}`).catch(() => null);
      const maint = await api('/api/maintenance').catch(() => null);
      const recent = session.me.role === 'admin' ? await api('/api/trips?limit=200').catch(() => []) : [];
      cache = { status, trips, ob, health, week, maint, unlabelled: recent.filter((x) => !x.open && x.categorySource === 'default').length, short: recent.filter((x) => x.startTs > Date.now() / 1000 - 30 * 86400 && shortOdometer(x)) };
    }
    const { status, trips, ob, health, week, maint, unlabelled, short = [] } = cache;
    const out = [...(staleSince ? [h('div', { class: 'banner' }, h('b', {}, 'You are offline. '), `Showing the last readings saved on this device (${ago(Math.floor(staleSince / 1000))}).`)] : []), ...notices(session)];
    if (ob) {
      // Only while the one essential step (linking Ford) is missing. The optional steps stay on the Getting started page.
      if (!ob.dismissed && ob.canLink && !ob.linked) out.push(h('a', { class: 'card', href: '#/start', style: 'display:block;text-decoration:none' }, h('b', {}, 'Getting started'), h('div', { class: 'muted' }, 'Connect your Ford account to start seeing your car →')));
    }
    const hi = greeting();
    if (hi) out.push(h('p', { class: 'greeting' }, hi));
    const strip = healthStrip(health, session.me.role === 'admin');
    if (strip) out.push(strip);
    const vac = await vacationNote();
    if (vac) out.push(vac);
    if (!latest) {
      out.push(h('p', { class: 'muted' }, 'No readings yet. Data appears after the first successful sync.'));
      return box.replaceChildren(...out);
    }
    const x = latest.extra || {};
    const soc = latest.socPct;
    const veh = session.vehicle || { battery: true, plug: true, engine: true }; // unknown type: show everything the car reports
    // A hybrid that reports no battery level (most do not) leads with its fuel tank instead of an empty battery ring.
    const fuelFirst = veh.engine && latest.fuelPct != null && (!veh.battery || soc == null);
    const t0 = trips[0];
    const driving = (t0 && t0.open) || latest.speedKph > 1;
    const isChargingNow = isCharging(latest);
    const plugged = veh.plug && latest.plug ? /^connected$/i.test(latest.plug) : null;
    const chips = h('div', { style: 'display:flex;gap:6px;flex-wrap:wrap;justify-content:center;margin:-4px 0 12px' },
      chip(driving ? 'personal' : isChargingNow ? 'ok' : 'work', driving ? 'Driving' : isChargingNow ? 'Charging' : 'Parked'),
      plugged == null ? null : chip(plugged ? 'ok' : 'work', plugged ? 'Plugged in' : 'Not plugged in'),
      latest.locked == null ? null : chip(latest.locked ? 'ok' : 'work', latest.locked ? 'Locked' : 'Unlocked'),
      remoteChip(latest));
    if (fuelFirst) {
      const ring = batteryRing('Fuel');
      ring.set(latest.fuelPct, pick(x, 'fuelRange') != null ? `${dist(pick(x, 'fuelRange'), 0)} range` : '');
      out.push(card('Fuel', ring.el, chips, h('div', { class: 'grid' }, stat('Oil life left', latest.oilLifePct != null ? pct(latest.oilLifePct) : '—'))));
    } else if (veh.battery) {
      const ring = batteryRing();
      ring.set(soc, latest.rangeKm != null ? `${dist(latest.rangeKm, 0)} range` : '');
      out.push(card('Battery', ring.el, chips,
        h('div', { class: 'grid' }, stat('Energy left', num(pick(x, 'XEV_BATTERY_ENERGY_REMAINING'), 1, ' kWh')),
          stat('Battery temp', temp(pick(x, 'XEV_BATTERY_TEMPERATURE'))),
          pick(x, 'xevBatteryMaximumRange') != null ? stat('Range at 100%', dist(pick(x, 'xevBatteryMaximumRange'), 0)) : null),
        reserveRow(latest),
        bandRow(),
        oldNote(x, 'xevBatteryStateOfCharge')));
    }
    const due = ((maint && maint.items) || []).filter((m) => m.state !== 'ok');
    if (due.length || unlabelled || short.length) {
      out.push(card('Needs attention',
        ...due.map((m) => h('a', { class: 'row', href: '#/maintenance', style: 'color:inherit;text-decoration:none' }, h('span', { class: 'grow' }, m.name), chip(m.state === 'overdue' ? 'bad' : 'work', m.state === 'overdue' ? 'service due' : 'due soon'))),
        unlabelled ? h('a', { class: 'row', href: '#/review', style: 'color:inherit;text-decoration:none' }, h('span', { class: 'grow' }, `${unlabelled} trip${unlabelled === 1 ? '' : 's'} to label`), chip('work', 'review')) : null,
        ...short.slice(0, 3).map((x) => h('a', { class: 'row', href: `#/trip/${x.id}`, style: 'color:inherit;text-decoration:none' }, h('span', { class: 'grow' }, `${when(x.startTs, x.tz)}: the distance looks too short`), chip('work', 'check')))));
    }
    if (!staleSince) out.push(...await serviceCards(session), ...await setupCards(session));
    if (!staleSince && new Date().getDate() <= 7) out.push(wrappedTeaser()); // the first week of a month: last month's Wrapped
    if (full) out.push(...await homeCards(maint)); // the day so far and the month: skipped in Data saver
    if (week && Object.values(week).reduce((n, x) => n + x.trips, 0) > 0) out.push(weekCard(week));
    const lastTrip = trips[0];
    if (lastTrip) out.push(card(lastTrip.open ? 'Trip in progress' : 'Last trip', h('a', { href: `#/trip/${lastTrip.id}` }, `${dist(lastTrip.distanceKm)} · ${lastTrip.category}`),
      h('p', { class: 'muted' }, `${lastTrip.startPlace || lastTrip.endPlace ? `${lastTrip.startPlace || '…'} → ${lastTrip.endPlace || '…'} · ` : ''}${when(lastTrip.startTs, lastTrip.tz)} · ${duration(lastTrip.startTs, lastTrip.endTs)}`),
      session.me.role === 'admin' && !lastTrip.open ? h('p', {}, h('a', { class: 'btn sec', href: `#/trip/${lastTrip.id}/share` }, 'Share this trip')) : null));
    if (veh.engine && latest.fuelPct != null && !fuelFirst) {
      out.push(card('Fuel and oil', h('div', { class: 'grid' }, stat('Fuel', pct(latest.fuelPct)), stat('Fuel range', dist(pick(x, 'fuelRange'), 0)),
        stat('Oil life left', latest.oilLifePct != null ? pct(latest.oilLifePct) : '—')),
        h('div', { class: 'bar', style: 'margin-top:10px' }, h('i', { style: `width:${Math.max(0, Math.min(100, latest.fuelPct))}%` }))));
    }

    const charging = isCharging(latest);
    if (veh.plug) out.push(card('Charging', h('div', { style: 'margin-bottom:8px' }, chip(charging ? 'ok' : 'work', nice(latest.chargeState) || 'Unknown'), ' ', chip(/^connected$/i.test(latest.plug || '') ? 'ok' : 'work', nice(latest.plug) || 'Unknown')),
      kv([['Power', latest.chargeKw != null ? num(latest.chargeKw, 1, ' kW') : null],
        ['Voltage', pick(x, 'XEV_BATTERY_CHARGER_VOLTAGE_OUTPUT') != null ? num(pick(x, 'XEV_BATTERY_CHARGER_VOLTAGE_OUTPUT'), 0, ' V') : null],
        ['Current', pick(x, 'XEV_BATTERY_CHARGER_CURRENT_OUTPUT') != null ? num(pick(x, 'XEV_BATTERY_CHARGER_CURRENT_OUTPUT'), 0, ' A') : null],
        ['Time to full', pick(x, 'XEV_BATTERY_TIME_TO_FULL_CHARGE') != null ? duration(0, pick(x, 'XEV_BATTERY_TIME_TO_FULL_CHARGE') * 60) : null],
        ['Station type', nice(pick(x, 'XEV_CHARGE_STATION_POWER_TYPE'))]]),
      charging ? chargeEtaBox(latest, pick(x, 'XEV_BATTERY_TIME_TO_FULL_CHARGE')) : null));

    out.push(card('Vehicle',
      kv([['Ignition', nice(latest.ignition) || '—'], ['Gear', nice(pick(x, 'GEAR_LEVER_POSITION'))], ['Odometer', dist(latest.odometerKm, 0)],
        ['Outside', temp(latest.outsideC)], ['Speed', speed(latest.speedKph)], ['12V battery', latest.battery12v != null ? num(latest.battery12v, 1, ' V') : null],
        ['Locks', latest.locked == null ? null : latest.locked ? 'Locked' : 'Unlocked'], ['Parking brake', pick(x, 'PARKING_BRAKE_STATUS')]])));

    if (latest.lat != null && latest.lon != null) {
      // A small fixed map of where the car is. One map is kept and moved, so a refresh does not reload the tiles.
      let here = null;
      if (mc.enabled && effective() !== 'saver') {
        homeMap = homeMap || createMap(mc, { height: 170, interactive: false });
        const d = 0.004; // about half a kilometer around the car
        homeMap.show({ markers: [{ lat: latest.lat, lon: latest.lon }], fit: [[latest.lat - d, latest.lon - d * 1.5], [latest.lat + d, latest.lon + d * 1.5]] });
        homeLink = homeLink || h('a', { target: '_blank', rel: 'noopener', 'aria-label': "Open the car's location in Google Maps", style: 'display:block;color:inherit' }, homeMap.el);
        homeLink.href = googleMapsUrl(latest.lat, latest.lon);
        here = homeLink;
      }
      out.push(card('Location', here, h('a', { href: googleMapsUrl(latest.lat, latest.lon), target: '_blank', rel: 'noopener' }, `${latest.lat.toFixed(5)}, ${latest.lon.toFixed(5)}`), mapButtons(latest.lat, latest.lon, 'the car'),
        h('p', { class: 'muted' }, `Heading ${pick(x, 'HEADING', 'direction', 'COMPASS_DIRECTION') ?? '—'} · seen ${ago(latest.ts)}`)));
    }

    const tires = (latest.tires || []).some((t) => t != null) ? latest.tires :['FL', 'FR', 'RL', 'RR'].map((k) => pick(x, `TIRE_PRESSURE_${k}`));
    if (tires.some((t) => t != null) || pick(x, 'tirePressureWarning') != null) {
      out.push(card('Tires', h('div', { class: 'grid' }, ['Front L', 'Front R', 'Rear L', 'Rear R'].map((l, i) => stat(l, pressure(tires[i])))),
        pick(x, 'tirePressureWarning') ? h('p', { class: 'chip bad' }, 'Low pressure warning') : null));
    }

    const doors = doorsCard(latest, x, pick);
    if (doors) out.push(doors);
    const hc = healthCard(x);
    if (hc) out.push(hc);

    if (!full) {
      out.push(h('div', { class: 'card' }, h('p', { class: 'muted' }, 'Data saver is on: showing the essentials. Tires, battery details and other fields are skipped.'),
        h('button', { class: 'sec', onclick: () => { full = true; load(); } }, 'Load full details once')));
    }
    box.replaceChildren(...out);
  }
  await load();
  return load;
}
