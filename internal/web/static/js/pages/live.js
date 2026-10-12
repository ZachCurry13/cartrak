import { h, chip, dist, speed, pct, num, temp, ago, duration } from '../ui.js';
import { api } from '../api.js';
import { effective } from '../mode.js';

// Full-screen "dock" view for a tablet on a car mount or a laptop: big tiles,
// updated the moment the server gets a new reading, screen kept awake.
export default async function live(root, _a, session) {
  const admin = session.me.role === 'admin';
  const el = {};
  const tile = (key, label, cls = '') => {
    el[key] = { v: h('div', { class: 'tv' }, '—'), s: h('div', { class: 'ts muted' }) };
    return h('div', { class: `tile ${cls}` }, h('div', { class: 'tl muted' }, label), el[key].v, el[key].s);
  };
  const conn = chip('work', 'Connecting…');
  const age = h('span', { class: 'muted' });
  const road = h('p', { class: 'muted', role: 'status' });
  const toggle = h('button', { class: 'sec' }, 'Road trip mode');
  const bar = h('i');
  const dock = h('div', { class: 'dock' },
    h('div', { class: 'dock-top' }, h('h2', { style: 'margin:0' }, 'Live'), conn, age,
      h('span', { class: 'grow' }),
      h('button', { class: 'sec', onclick: () => (document.fullscreenElement ? document.exitFullscreen() : root.requestFullscreen && root.requestFullscreen()) }, 'Full screen')),
    h('div', { class: 'tiles' },
      h('div', { class: 'tile big' }, h('div', { class: 'tl muted' }, 'Charge'), (el.soc = { v: h('div', { class: 'tv' }, '—'), s: h('div', { class: 'ts muted' }) }).v,
        h('div', { class: 'bar' }, bar), el.soc.s),
      tile('range', 'Range'), tile('speed', 'Speed'), tile('power', 'Charging power'),
      tile('trip', 'This trip'), tile('outside', 'Outside'), tile('odo', 'Odometer')),
    h('div', {}, admin ? h('p', {}, toggle) : null, road));

  let lastTs = 0;
  let wake = null;
  let es = null;
  let timer = null;
  const set = (k, v, s) => { el[k].v.textContent = v; el[k].s.textContent = s || ''; };

  function update(ev) {
    const x = ev.sample;
    if (x) {
      lastTs = x.ts;
      set('soc', pct(x.socPct), x.chargeState || '');
      bar.style.width = `${Math.max(0, Math.min(100, x.socPct || 0))}%`;
      set('range', dist(x.rangeKm, 0));
      set('speed', speed(x.speedKph), x.ignition ? `Ignition ${x.ignition}` : '');
      set('power', x.chargeKw > 0.2 ? num(x.chargeKw, 1, ' kW') : '—', x.plug || '');
      set('outside', temp(x.outsideC));
      set('odo', dist(x.odometerKm, 0));
    }
    const t = ev.trip;
    set('trip', t ? dist(t.distanceKm) : '—', t ? `${duration(t.startTs, Math.floor(Date.now() / 1000))} · top ${speed(t.maxSpeedKph)}` : 'No trip in progress');
    showRoad(ev.status);
    tickAge();
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

  root.append(h('p', {}, h('a', { href: '#/' }, '← Home')), dock);
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
      clearInterval(timer);
      document.removeEventListener('visibilitychange', onVisible);
      if (wake) { wake.release().catch(() => {}); wake = null; }
      if (document.fullscreenElement) document.exitFullscreen().catch(() => {});
    },
  };
}
