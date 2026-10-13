import { h, chip, nice, dist, speed, pct, num, temp, ago, duration } from '../ui.js';
import { api } from '../api.js';
import { effective } from '../mode.js';
import { reserveKm, getReserve, chargeEta, getTarget, setTarget, minutesText, clockAt, PRESETS } from '../charge.js';
import { gpsCard } from '../gpscard.js';
import { efficiency } from '../energy.js';
import { batteryRing } from '../ring.js';

// Full-screen "dock" view for a tablet on a car mount or a laptop: big tiles,
// updated the moment the server gets a new reading, screen kept awake.
export default async function live(root, _a, session) {
  const admin = session.me.role === 'admin';
  const el = {};
  const tile = (key, label, cls = '') => {
    el[key] = { l: h('div', { class: 'tl muted' }, label), v: h('div', { class: 'tv' }, '—'), s: h('div', { class: 'ts muted' }) };
    return h('div', { class: `tile ${cls}` }, el[key].l, el[key].v, el[key].s);
  };
  const conn = chip('work', 'Connecting…');
  const age = h('span', { class: 'muted' });
  const road = h('p', { class: 'muted', role: 'status' });
  const toggle = h('button', { class: 'sec' }, 'Road trip mode');
  const ring = batteryRing();
  el.soc = { v: h('div', { class: 'tv' }, '—'), s: h('div', { class: 'ts muted' }) };
  const dock = h('div', { class: 'dock' },
    h('div', { class: 'dock-top' }, h('h2', { style: 'margin:0' }, 'Live'), conn, age,
      h('span', { class: 'grow' }),
      h('button', { class: 'sec', onclick: () => (document.fullscreenElement ? document.exitFullscreen() : root.requestFullscreen && root.requestFullscreen()) }, 'Full screen')),
    h('div', { class: 'tiles' },
      h('div', { class: 'tile big' }, h('div', { class: 'tl muted' }, 'Charge'), ring.el, el.soc.s),
      tile('range', 'Range'), tile('energy', 'Energy left'), tile('reserve', 'To reserve'), tile('speed', 'Speed'), tile('power', 'Charging power'), tile('eta', 'Time to target'),
      tile('trip', 'This trip'), tile('outside', 'Outside'), tile('odo', 'Odometer')),
    h('div', {}, admin ? h('p', {}, toggle) : null, road));

  let lastTs = 0;
  let lastX = null;
  const etaTile = el.eta.v.parentElement;
  etaTile.style.cursor = 'pointer';
  etaTile.setAttribute('role', 'button');
  etaTile.setAttribute('tabindex', '0');
  const cycleEta = () => { const t = getTarget(); setTarget(PRESETS[(PRESETS.indexOf(t) + 1) % PRESETS.length] || PRESETS[0]); if (lastX) showEta(lastX); };
  etaTile.onclick = cycleEta;
  etaTile.onkeydown = (e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); cycleEta(); } };
  let wake = null;
  let es = null;
  let timer = null;
  const set = (k, v, s) => { el[k].v.textContent = v; el[k].s.textContent = s || ''; };

  function update(ev) {
    const x = ev.sample;
    if (x) {
      lastTs = x.ts;
      lastX = x;
      set('soc', pct(x.socPct), nice(x.chargeState) || '');
      ring.set(x.socPct, x.rangeKm != null ? `${dist(x.rangeKm, 0)} range` : '');
      set('range', dist(x.rangeKm, 0));
      set('energy', x.energyKwh != null ? num(x.energyKwh, 1, ' kWh') : '—', x.capacityKwh ? `of about ${num(x.capacityKwh, 0, ' kWh')}` : '');
      showReserve(x);
      showEta(x);
      set('speed', speed(x.speedKph), x.ignition ? `Ignition ${nice(x.ignition).toLowerCase()}` : '');
      set('power', x.chargeKw > 0.2 ? num(x.chargeKw, 1, ' kW') : '—', nice(x.plug) || '');
      set('outside', temp(x.outsideC));
      set('odo', dist(x.odometerKm, 0));
    }
    const t = ev.trip;
    const used = t && t.startKwh != null && x && x.energyKwh != null ? t.startKwh - x.energyKwh : null;
    const eff = t ? efficiency(t.distanceKm, used) : null;
    set('trip', t ? dist(t.distanceKm) : '—', t ? `${duration(t.startTs, Math.floor(Date.now() / 1000))} · top ${speed(t.maxSpeedKph)}${eff ? ` · ${eff}` : ''}` : 'No trip in progress');
    showRoad(ev.status);
    tickAge();
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
    toggle.textContent = st.live ? 'Turn off road trip mode' : 'Road trip mode';
    toggle.onclick = async () => {
      const next = await api('/api/live-mode', { method: 'POST', body: { enabled: !st.live } });
      showRoad(next);
    };
    road.textContent = st.live
      ? `Fast updates while driving, until ${new Date(st.liveUntil * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}.`
      : 'Road trip mode asks Ford for readings more often while driving. It turns itself off after 12 hours.';
    if (st.rateLimited) road.textContent += ' Ford asked CarTrak to slow down; it is waiting.';
  }

  function tickAge() { age.textContent = lastTs ? `reading ${ago(lastTs)}` : ''; }

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

  const gps = admin ? gpsCard() : null;
  root.append(h('p', {}, h('a', { href: '#/' }, '← Home')), dock, ...(gps ? [gps.el] : []));
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
