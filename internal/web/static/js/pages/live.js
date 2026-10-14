import { h, chip, nice, dist, speed, num, temp, ago, duration } from '../ui.js';
import { api } from '../api.js';
import { effective } from '../mode.js';
import { reserveKm, getReserve, chargeEta, getTarget, setTarget, minutesText, clockAt, PRESETS } from '../charge.js';
import { gpsCard } from '../gpscard.js';
import { efficiency } from '../energy.js';
import { batteryRing } from '../ring.js';

// Which six tiles to show, in order, for what the car is doing right now. The battery ring is always there.
const MODES = {
  driving: ['speed', 'trip', 'tripTime', 'reserve', 'range', 'today'],
  charging: ['power', 'eta', 'added', 'range', 'energy', 'outside'],
  parked: ['range', 'energy', 'lock', 'today', 'outside', 'odo'],
};
const MODE_LABEL = { driving: 'Driving', charging: 'Charging', parked: 'Parked' };

// Full-screen "dock" view for a phone, a tablet on a car mount or a laptop. It fits one screen and changes with
// what the car is doing: speed and trip while driving, power and time to target while charging, and the
// essentials while parked. It updates the moment the server gets a new reading, and keeps the screen awake.
export default async function live(root, _a, session) {
  const admin = session.me.role === 'admin';
  const el = {};
  const tile = (key, label) => {
    el[key] = { l: h('div', { class: 'tl muted' }, label), v: h('div', { class: 'tv' }, '—'), s: h('div', { class: 'ts muted' }) };
    el[key].box = h('div', { class: 'tile', hidden: true }, el[key].l, el[key].v, el[key].s);
    return el[key].box;
  };
  const conn = chip('work', 'Connecting…');
  const modeChip = chip('work', '');
  modeChip.hidden = true;
  const age = h('span', { class: 'muted' });
  const road = h('span', { class: 'muted', role: 'status' });
  const toggle = h('button', { class: 'sec' }, 'Road trip mode');
  const ring = batteryRing();
  const heroSub = h('div', { class: 'ts muted' });
  const hero = h('div', { class: 'tile big' }, h('div', { class: 'tl muted' }, 'Charge'), ring.el, heroSub);
  const gps = admin ? gpsCard() : null;
  const dock = h('div', { class: 'dock' },
    h('div', { class: 'dock-top' }, conn, modeChip, age, h('span', { class: 'grow' }),
      admin ? toggle : null,
      h('button', { class: 'sec', onclick: () => (document.fullscreenElement ? document.exitFullscreen() : root.requestFullscreen && root.requestFullscreen()) }, 'Full screen')),
    h('div', { class: 'tiles' }, hero,
      tile('speed', 'Speed'), tile('trip', 'This trip'), tile('tripTime', 'Trip time'), tile('reserve', 'To reserve'), tile('range', 'Range'), tile('today', 'Driving today'),
      tile('power', 'Charging power'), tile('eta', 'Time to target'), tile('added', 'Added this session'), tile('energy', 'Energy left'), tile('outside', 'Outside'),
      tile('lock', 'Doors'), tile('odo', 'Odometer')),
    road);

  let lastTs = 0;
  let lastX = null;
  let lastCharge = null;
  let mode = null;
  let carTs = 0;
  let lastTrip = null;
  let wake = null;
  let es = null;
  let timer = null;
  const set = (k, v, s) => { el[k].v.textContent = v; el[k].s.textContent = s || ''; };

  const etaBox = el.eta.box;
  etaBox.style.cursor = 'pointer';
  etaBox.setAttribute('role', 'button');
  etaBox.setAttribute('tabindex', '0');
  const cycleEta = () => { const t = getTarget(); setTarget(PRESETS[(PRESETS.indexOf(t) + 1) % PRESETS.length] || PRESETS[0]); if (lastX) showEta(lastX); };
  etaBox.onclick = cycleEta;
  etaBox.onkeydown = (e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); cycleEta(); } };

  const charging = (x) => x.chargeKw > 0.2 || (/charging/i.test(x.chargeState || '') && !/not|complete|stop/i.test(x.chargeState || ''));

  function pickMode(ev) {
    const x = ev.sample;
    if (!x) return 'parked';
    if (ev.trip || x.speedKph > 1) return 'driving';
    return charging(x) ? 'charging' : 'parked';
  }

  function showMode(m) {
    if (m === mode) return;
    mode = m;
    modeChip.hidden = false;
    modeChip.className = `chip ${m === 'driving' ? 'personal' : m === 'charging' ? 'ok' : 'work'}`;
    modeChip.textContent = MODE_LABEL[m];
    const shown = MODES[m];
    for (const [key, e] of Object.entries(el)) {
      if (!e.box) continue;
      const i = shown.indexOf(key);
      e.box.hidden = i < 0;
      e.box.style.order = String(i);
    }
  }

  function update(ev) {
    const x = ev.sample;
    lastCharge = ev.charge || null;
    showMode(pickMode(ev));
    if (x) {
      lastTs = x.ts;
      lastX = x;
      ring.set(x.socPct, x.rangeKm != null ? `${dist(x.rangeKm, 0)} range` : '');
      heroSub.textContent = mode === 'charging' && x.chargeKw > 0.2 ? `Charging at ${num(x.chargeKw, 1, ' kW')}` : nice(x.chargeState) || '';
      set('range', dist(x.rangeKm, 0));
      set('energy', x.energyKwh != null ? num(x.energyKwh, 1, ' kWh') : '—', x.capacityKwh ? `of about ${num(x.capacityKwh, 0, ' kWh')}` : '');
      showReserve(x);
      showEta(x);
      set('speed', speed(x.speedKph), x.ignition ? `Ignition ${nice(x.ignition).toLowerCase()}` : '');
      set('power', x.chargeKw > 0.2 ? num(x.chargeKw, 1, ' kW') : '—', nice(x.plug) || '');
      set('outside', temp(x.outsideC));
      set('odo', dist(x.odometerKm, 0));
      set('lock', x.locked == null ? '—' : x.locked ? 'Locked' : 'Unlocked');
      showAdded(x);
    }
    const t = ev.trip;
    const used = t && t.startKwh != null && x && x.energyKwh != null ? t.startKwh - x.energyKwh : null;
    const eff = t ? efficiency(t.distanceKm, used) : null;
    set('trip', t ? dist(t.distanceKm) : '—', t ? `top ${speed(t.maxSpeedKph)}${eff ? ` · ${eff}` : ''}` : 'No trip in progress');
    carTs = ev.carTs || 0;
    lastTrip = { trip: t, today: ev.drivingToday || 0, at: Math.floor(Date.now() / 1000) };
    showTimes();
    showRoad(ev.status);
    tickAge();
  }

  // Energy and time since this charging session began.
  function showAdded(x) {
    const c = lastCharge;
    let kwh = null;
    if (c && c.startKwh != null && x.energyKwh != null) kwh = x.energyKwh - c.startKwh;
    else if (c && c.startSoc != null && x.socPct != null && x.capacityKwh) kwh = ((x.socPct - c.startSoc) / 100) * x.capacityKwh;
    set('added', kwh != null && kwh > 0 ? num(kwh, 1, ' kWh') : '—', c ? `since ${new Date(c.startTs * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}` : 'no session');
  }

  // Time on this trip and time driving today. Both keep counting between readings while a trip is going.
  function showTimes() {
    if (!lastTrip) return;
    const now = Math.floor(Date.now() / 1000);
    const { trip } = lastTrip;
    set('tripTime', trip ? duration(trip.startTs, now) : '—', trip ? `since ${new Date(trip.startTs * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}` : 'No trip in progress');
    const today = lastTrip.today + (trip ? now - lastTrip.at : 0);
    const m = Math.round(today / 60);
    set('today', m < 60 ? `${m} min` : `${Math.floor(m / 60)} h ${m % 60} min`, 'since midnight');
  }

  // Miles until the battery falls to this device's reserve, and the time to the charging target.
  function showReserve(x) {
    const r = getReserve();
    el.reserve.l.textContent = `To ${r}%`;
    const km = reserveKm(x, r);
    set('reserve', km == null ? '—' : km > 0 ? dist(km, 0) : 'below', km === 0 ? `already under ${r}%` : 'at the current range estimate');
  }
  function showEta(x) {
    const t = getTarget();
    el.eta.l.textContent = `Time to ${t}%`;
    const e = chargeEta(x, t);
    const text = !e ? '—' : e.done ? 'reached' : e.idle ? '—' : minutesText(e.minutes);
    set('eta', text, e && e.minutes ? `about ${clockAt(e.minutes)} · tap to change` : e && e.idle ? 'not charging' : 'tap to change');
  }
  function showRoad(st) {
    if (!st) return;
    toggle.textContent = st.live ? 'Road trip: on' : 'Road trip mode';
    toggle.title = 'Road trip mode asks Ford for readings more often while driving. It turns itself off after 12 hours.';
    toggle.onclick = async () => {
      const next = await api('/api/live-mode', { method: 'POST', body: { enabled: !st.live } });
      showRoad(next);
    };
    road.textContent = [st.live ? `Fast updates while driving, until ${new Date(st.liveUntil * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}.` : '', st.rateLimited ? 'Ford asked CarTrak to slow down; it is waiting.' : ''].filter(Boolean).join(' ');
  }

  // Time since the car itself last reported (CarTrak asks Ford more often than a sleeping car has news).
  function tickAge() { age.textContent = lastTs ? `car reported ${ago(carTs || lastTs)}` : ''; showTimes(); }

  async function keepAwake() {
    try { if ('wakeLock' in navigator && !document.hidden) wake = await navigator.wakeLock.request('screen'); } catch (e) { /* not allowed */ }
  }

  function start() {
    es = new EventSource('/api/live');
    es.addEventListener('reading', (e) => { try { update(JSON.parse(e.data)); } catch (err) { /* ignore a bad frame */ } });
    es.onopen = () => { conn.className = 'chip ok'; conn.textContent = 'Live'; };
    es.onerror = () => { conn.className = 'chip bad'; conn.textContent = 'Reconnecting…'; };
    timer = setInterval(tickAge, 1000);
    keepAwake();
  }
  const onVisible = () => { if (!document.hidden && !wake) keepAwake(); };
  document.addEventListener('visibilitychange', onVisible);

  root.append(dock);
  if (gps) root.append(h('details', { class: 'livegps' }, h('summary', { class: 'muted' }, 'Phone GPS'), gps.el));
  if (effective() === 'saver') {
    // The live feed keeps a connection open; on a limited plan ask first.
    const gate = h('div', { class: 'card' }, h('p', { class: 'muted' }, 'Data saver is on for this device. The live view keeps a connection open and uses more data.'),
      h('button', { onclick: () => { gate.remove(); start(); } }, 'Start live view anyway'));
    root.append(gate);
  } else {
    start();
  }

  return {
    cleanup() {
      if (es) es.close();
      if (gps) gps.stop();
      clearInterval(timer);
      document.removeEventListener('visibilitychange', onVisible);
      if (wake) { wake.release().catch(() => {}); wake = null; }
      if (document.fullscreenElement) document.exitFullscreen().catch(() => {});
    },
  };
}
