import { h, chip, dist, isMetric, temp, tempBandLabel } from '../ui.js';
import { api } from '../api.js';
import { fitSpeedModel } from '../bestspeed.js';
import { bestSpeedCard } from '../bestspeedcard.js';
import { getReserve, getTarget } from '../charge.js';
import { planTrip } from '../tripplan.js';

const KEY = 'cartrak.planKw';
const saved = () => { try { return Number(localStorage.getItem(KEY)) || 0; } catch (e) { return 0; } };
const write = (v) => { try { localStorage.setItem(KEY, String(v)); } catch (e) { /* ignore */ } };
const toKm = (v) => (isMetric() ? v : v * 1.609344);
const hm = (m) => (m == null ? '—' : m < 60 ? `${m} min` : `${Math.floor(m / 60)} h ${m % 60} min`);

// How many charging stops does a trip need, and about where?
export default async function plan(root) {
  const latest = await api('/api/latest?summary=1').catch(() => null);
  const low = getReserve();
  const high = getTarget();
  const spd = await api('/api/speed-efficiency').catch(() => null); // your energy use at each speed, for the best-speed card
  const speedModel = spd ? fitSpeedModel(spd.bands) : null;
  const fast = await api('/api/charging/fast-average').catch(() => null); // what the car really averages at fast chargers
  const histKw = fast && fast.avgKw ? Math.round(fast.avgKw) : 0;
  const band = await api(`/api/range-band?low=${low}&high=${high}&days=60`).catch(() => null);
  const avgKmPerPct = band && band.bandKm ? band.bandKm / (high - low) : null;
  // Cold weather uses much more battery, so the plan can use what you really get at the temperature you choose.
  const weather = h('select', { id: 'pw' },
    ...(band && band.nowBand ? [h('option', { value: 'now' }, `Today's temperature (${temp(band.outsideC)})`)] : []),
    h('option', { value: 'avg' }, 'Recent average'),
    ...((band && band.byTemp) || []).map((b) => h('option', { value: b.name }, tempBandLabel(b.name))));
  const kmPerPct = () => {
    const v = weather.value;
    if (v === 'now' && band.nowBand) return band.nowBand.kmPerPct;
    const hit = ((band && band.byTemp) || []).find((b) => b.name === v);
    return hit ? hit.kmPerPct : avgKmPerPct;
  };
  const unit = isMetric() ? 'km' : 'mi';
  const distance = h('input', { id: 'pd', type: 'number', min: 1, inputmode: 'decimal', placeholder: unit === 'mi' ? '420' : '680' });
  const start = h('input', { id: 'ps', type: 'number', min: 1, max: 100, inputmode: 'numeric', value: latest && latest.socPct != null ? Math.round(latest.socPct) : 100 });
  const kw = h('input', { id: 'pk', type: 'number', min: 10, max: 350, inputmode: 'numeric', value: saved() || histKw || 100 });
  const out = h('div');
  const field = (id, label, input, help) => h('div', {}, h('label', { for: id }, label), input, help ? h('p', { class: 'muted' }, help) : null);

  function show() {
    write(Number(kw.value) || 100);
    const r = planTrip({ distanceKm: toKm(Number(distance.value)), startPct: Number(start.value), reservePct: low, targetPct: high, kmPerPct: kmPerPct(),
      capacityKwh: latest && latest.capacityKwh, chargeKw: Number(kw.value) });
    if (!r.ok) return out.replaceChildren(h('p', { class: 'muted' }, r.reason));
    const where = (km) => dist(km, 0);
    out.replaceChildren(...[h('section', { class: 'card' }, h('h2', {}, 'The plan'),
      h('p', {}, r.stops.length ? chip('work', `${r.stops.length} charging stop${r.stops.length === 1 ? '' : 's'}`) : chip('ok', 'No charging stop needed'), ' ',
        h('span', { class: 'muted' }, r.stops.length ? `About ${hm(r.extraMinutes)} charging in total.` : `You arrive with about ${Math.round(r.arrivePct)}%.`)),
      ...r.stops.map((s, i) => h('div', { class: 'row' }, h('div', { class: 'grow' }, h('b', {}, `Stop ${i + 1}, about ${where(s.atKm)} in`),
        h('div', { class: 'muted' }, `Arrive at ${Math.round(s.arrivePct)}%, charge to ${Math.round(s.chargeToPct)}%`)), h('b', {}, hm(s.minutes)))),
      r.stops.length ? h('p', { class: 'muted' }, `You arrive with about ${Math.round(r.arrivePct)}%.`) : null,
      h('p', { class: 'muted' }, 'A guide, not a route. Hills, highway speed, wind and cold use more battery, and fast chargers slow down above 80%. Leave extra room, and pick real chargers near each mark.')),
      speedModel && latest && latest.capacityKwh && kmPerPct() ? bestSpeedCard({ model: speedModel, distanceKm: toKm(Number(distance.value)), startPct: Number(start.value), reservePct: low, targetPct: high,
        chargeKw: Number(kw.value), capacityKwh: latest.capacityKwh, cChosen: latest.capacityKwh / 100 / kmPerPct() }, show) : null].filter(Boolean));
  }
  for (const i of [distance, start, kw, weather]) i.addEventListener('input', show);

  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Road-trip planner')),
    h('section', { class: 'card' }, h('h2', {}, 'Your trip'),
      avgKmPerPct ? h('p', { class: 'muted' }, `Your battery goes about ${dist(avgKmPerPct, 1)} per percent${band.source === 'trips' ? ', from your recent driving' : ", from the car's own range figure"}. The reserve (${low}%) and charge target (${high}%) are your settings.`)
        : h('p', { class: 'banner' }, 'CarTrak does not know how far your battery goes yet. It needs a reading with the range, or a few trips.'),
      field('pd', `Trip distance (${unit})`, distance), field('ps', 'Starting charge (%)', start),
      band && ((band.byTemp && band.byTemp.length) || band.nowBand) ? field('pw', 'Weather', weather, 'Based on how far you really got on trips in that weather. Cold and hot weather both use more.') : null, field('pk', 'Average charging power (kW)', kw, (saved() || !histKw ? '' : `Started from your own fast charges (${fast.sessions} session${fast.sessions === 1 ? '' : 's'} averaged ${histKw} kW). `) + 'The average over a stop, between the reserve and the target. Many cars average well below their peak.')),
    out);
  show();
}
