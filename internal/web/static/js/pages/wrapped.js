import { h, dist, temp, isMetric } from '../ui.js';
import { api } from '../api.js';
import { compareDistance, compareEnergy, compareDriver, milesOrKm, distUnit, personalityLine, periodTitle, valueText, clockText } from '../funwords.js';
import { flowChart, dayRing, ringKey } from '../funcharts.js';
import { drawInfographic, shareImage } from '../infographic.js';
import { efficiency } from '../energy.js';
import { economy, volText, priceText } from '../fuelunits.js';
import { activeTheme, THEMES } from '../seasons.js';

// Wrapped: a month or a year as full-screen slides. Tap the right side for the next one, the left side to go back. The last slide
// is the infographic, to share or save.

const SKIN = ['w-teal', 'w-violet', 'w-amber', 'w-blue', 'w-rose', 'w-green', 'w-violet', 'w-teal'];

function countUp(el, to) {
  const t0 = performance.now(), dur = 1100;
  const step = (t) => { const k = Math.min(1, (t - t0) / dur), e = 1 - (1 - k) ** 3; el.textContent = Math.round(to * e).toLocaleString(); if (k < 1) requestAnimationFrame(step); };
  requestAnimationFrame(step);
}

const plural = (n, w) => `${n} ${w}${n === 1 ? '' : 's'}`;

export default async function wrapped(root, args) {
  const period = args[0] || '';
  const w = await api(`/api/analytics/wrapped?period=${encodeURIComponent(period)}`);
  const r = w.recap, cur = w.currency || '$', title = periodTitle(r.period);
  const season = activeTheme();
  const routes = (await api(`/api/analytics/everywhere?period=${encodeURIComponent(r.period)}`).catch(() => ({ routes: [] }))).routes;
  if (!r.trips) {
    root.append(h('p', {}, h('a', { href: '#/analytics' }, '← Insights')), h('div', { class: 'empty' }, h('b', {}, `Nothing to wrap for ${title}`), h('p', { class: 'muted' }, 'There were no trips in it.')));
    return;
  }
  const big = h('div', { class: 'huge' }, '0');
  const prev = r.prevKm > 0 ? Math.round(((r.km - r.prevKm) / r.prevKm) * 100) : null;
  const p = w.personalities || [];
  const vs = compareDriver(r.km, w.ring && w.ring.days); // per day since CarTrak began, if that was later than the period's start
  const moments = (w.moments || []).filter((m) => ['longest', 'coldest', 'hottest', 'peakKw', 'efficient', 'topSpeed', 'bestMpg', 'longestTank', 'cheapestGas'].includes(m.key)).slice(0, 4);
  const canvas = drawInfographic(w, routes, season ? [THEMES[season.key][6], THEMES[season.key][7]] : null);
  const img = h('img', { src: canvas.toDataURL('image/png'), alt: `${title} infographic`, class: 'wposter' });
  const shareMsg = h('p', { class: 'small', role: 'status' });
  const slides = [
    [h('div', { class: 'kick' }, 'CarTrak Wrapped'), h('div', { class: 'grow' }), h('div', { class: 'huge', style: 'font-size:3.4rem' }, title), h('div', { class: 'line' }, 'Your car’s story. Tap to begin.'), h('div', { class: 'wemoji' }, season ? `${THEMES[season.key][1]}🚗💨` : '🚗💨')],
    [h('div', { class: 'kick' }, `Your ${title}`), h('div', { class: 'grow' }), big, h('div', { class: 'line' }, `${distUnit()}. That’s ${compareDistance(r.km)}.`),
      h('div', { class: 'small' }, `${r.trips} ${r.trips === 1 ? 'trip' : 'trips'} · ${r.hours < 1 ? plural(Math.round(r.hours * 60), 'minute') : plural(Math.round(r.hours), 'hour')} behind the wheel · ${plural(r.daysDriven, 'day')} on the road${prev != null ? ` · ${prev >= 0 ? '▲' : '▼'} ${Math.abs(prev)}% on the one before` : ''}`)],
    vs ? [h('div', { class: 'kick' }, 'How you compare'), h('div', { class: 'grow' }), h('div', { class: 'huge' }, vs.big), h('div', { class: 'line' }, vs.line),
      h('div', { class: 'small' }, `${milesOrKm(r.km / w.ring.days)} ${distUnit()} a day on average, against about ${milesOrKm(59.5)} for the average American driver (US Department of Transportation).`)] : null,
    p.length ? [h('div', { class: 'kick' }, 'Your car’s personality'), h('div', { class: 'wemoji big' }, p[0].emoji), h('div', { class: 'huge', style: 'font-size:2.6rem' }, p[0].name), h('div', { class: 'line' }, `${personalityLine(p[0])}.`),
      p[1] ? h('div', { class: 'small' }, `Runner-up: ${p[1].name} ${p[1].emoji}`) : null] : null,
    r.kwhCharged > 0 ? [h('div', { class: 'kick' }, 'Energy'), h('div', { class: 'huge' }, valueText('kwh', r.kwhCharged)), h('div', { class: 'line' }, `charged, ${compareEnergy(r.kwhCharged)}.`),
      h('div', { class: 'small' }, `${Math.round(r.homeShare * 100)}% at home · ${r.sessions} charges · ${cur}${r.cost.toFixed(2)}${r.kmPerKwh ? ` · ${efficiency(r.kmPerKwh * 100, 100)}` : ''}`),
      r.saved != null ? h('div', { class: 'wsave' }, h('b', {}, `${cur}${Math.round(r.saved)}`), ' saved against a gas car', r.kgCo2Saved > 0 ? h('div', {}, `and ${Math.round(r.kgCo2Saved)} kg of CO₂ kept out of the air`) : null) : null] : null,
    w.fuel && !(r.kwhCharged > 0) ? [h('div', { class: 'kick' }, 'Fuel'), h('div', { class: 'huge' }, `${cur}${Math.round(w.fuel.spent).toLocaleString()}`), h('div', { class: 'line' }, `on fuel${w.fuel.galBought > 0 ? `, ${volText(w.fuel.galBought)} in ${w.fuel.fills} fill-up${w.fuel.fills === 1 ? '' : 's'}` : ''}.`),
      h('div', { class: 'small' }, [w.fuel.kmPerGal > 0 ? `${economy(w.fuel.kmPerGal, 1)} on average` : '', w.fuel.avgPrice > 0 ? `${priceText(cur, w.fuel.avgPrice)} on average` : '', w.fuel.costPerKm > 0 ? `${cur}${(w.fuel.costPerKm * (isMetric() ? 1 : 1.609344)).toFixed(2)} a ${isMetric() ? 'km' : 'mile'}` : ''].filter(Boolean).join(' · '))] : null,
    r.kwhCharged > 0 || w.flow.homeKwh + w.flow.publicKwh > 0 ? [h('div', { class: 'kick' }, 'Where every kWh went'), h('div', { class: 'wpanel' }, flowChart(w.flow))] : null,
    [h('div', { class: 'kick' }, 'Out and about'), ...(w.towns ? [h('div', { class: 'huge' }, String(w.towns)), h('div', { class: 'line' }, `town${w.towns === 1 ? '' : 's'} visited.`)]
      : r.longest ? [h('div', { class: 'huge' }, dist(r.longest.km, 0)), h('div', { class: 'line' }, 'your longest drive.')] : []),
      (w.places || []).length ? h('div', { class: 'small' }, `Your regulars: ${w.places.map((x) => `${x.name} (${x.visits})`).join(', ')}`) : null,
      r.longest && w.towns ? h('div', { class: 'small' }, `Longest drive: ${dist(r.longest.km, 0)}.`) : null,
      r.busiestDay ? h('div', { class: 'small' }, `Busiest day: ${new Date(`${r.busiestDay.date}T12:00`).toLocaleDateString([], { weekday: 'long', month: 'short', day: 'numeric' })}, ${dist(r.busiestDay.km, 0)}.`) : null],
    moments.length ? [h('div', { class: 'kick' }, 'Highlights'), ...moments.map((m) => h('div', { class: 'wmoment' }, h('b', {}, valueText(m.kind, m.value, cur)), h('span', {}, m.title)))] : null,
    w.service && w.service.visits ? [h('div', { class: 'kick' }, 'Car care'), h('div', { class: 'wemoji big' }, '🔧'), h('div', { class: 'huge', style: 'font-size:2.6rem' }, `${w.service.visits} service visit${w.service.visits === 1 ? '' : 's'}`),
      h('div', { class: 'line' }, w.service.items.slice(0, 5).join(', ')), w.service.cost > 0 ? h('div', { class: 'small' }, `${cur}${w.service.cost.toFixed(2)} spent keeping it in shape.`) : null] : null,
    (w.badges || []).length ? [h('div', { class: 'kick' }, 'Achievements earned'), h('div', { class: 'wbadges' }, ...w.badges.map((b) => h('div', {}, h('span', { class: 'wemoji' }, b.emoji), h('b', {}, b.name))))] : null,
    w.ring && w.ring.days ? [h('div', { class: 'kick' }, 'A day in the life'), h('div', { class: 'wpanel' }, dayRing(w.ring), ringKey()),
      w.ring.leaveAt ? h('div', { class: 'small' }, `Out the door around ${clockText(w.ring.leaveAt)} on weekdays${w.ring.plugInAt ? `, plugged in around ${clockText(w.ring.plugInAt)}` : ''}.`) : null] : null,
    [h('div', { class: 'kick' }, 'Your infographic'), img, h('div', { class: 'seg', style: 'margin-top:12px' },
      h('button', { type: 'button', class: 'on', onclick: async (e) => { e.stopPropagation(); shareMsg.textContent = { shared: 'Shared.', saved: 'Saved to your downloads.', canceled: '' }[await shareImage(canvas, `CarTrak ${title}`)]; } }, 'Share picture'),
      h('a', { class: 'btn sec', href: '#/analytics', onclick: (e) => e.stopPropagation() }, 'Done')), shareMsg],
  ].filter(Boolean);

  let at = 0;
  const bars = h('div', { class: 'dots' }, ...slides.map(() => h('i')));
  const body = h('div', { class: 'wbody' });
  const close = h('a', { class: 'wclose', href: '#/analytics', 'aria-label': 'Close' }, '×');
  const stage = h('section', { class: 'wrapped', role: 'dialog', 'aria-label': `${title} wrapped` }, bars, close, body);
  const show = (i) => {
    at = Math.max(0, Math.min(slides.length - 1, i));
    stage.className = `wrapped ${season && (at === 0 || at === slides.length - 1) ? 'wseason' : SKIN[at % SKIN.length]}`; // the season opens and closes it
    [...bars.children].forEach((b, j) => b.classList.toggle('on', j <= at));
    body.replaceChildren(...slides[at].filter(Boolean));
    if (slides[at].includes(big)) countUp(big, Number(milesOrKm(r.km).replace(/,/g, '')));
  };
  stage.addEventListener('click', (e) => { if (e.target.closest('a,button')) return; show(at + (e.clientX < innerWidth / 3 ? -1 : 1)); });
  const keys = (e) => { if (e.key === 'ArrowRight') show(at + 1); if (e.key === 'ArrowLeft') show(at - 1); if (e.key === 'Escape') location.hash = '#/analytics'; };
  addEventListener('keydown', keys);
  root.append(stage);
  show(0);
  return { cleanup: () => removeEventListener('keydown', keys) };
}
