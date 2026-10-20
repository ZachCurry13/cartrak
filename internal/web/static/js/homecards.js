import { h, dist, num } from './ui.js';
import { api } from './api.js';

const money = (cur, v) => `${cur}${v.toFixed(2)}`;
const hm = (sec) => (sec < 3600 ? `${Math.max(1, Math.round(sec / 60))} min` : `${Math.floor(sec / 3600)} h ${Math.round((sec % 3600) / 60)} min`);
const ymd = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
const link = (href, title, ...kids) => h('a', { class: 'card', href, style: 'display:block;text-decoration:none;color:inherit' }, h('h2', {}, title), ...kids);

// The day so far, in a line, with a way into the day's map and details.
async function todayCard() {
  const x = await api('/api/day').catch(() => null);
  if (!x || !x.events || !x.events.length) return null;
  const s = x.summary || {}, t = x.totals || {}, e = x.energy || {};
  const bits = [];
  if (s.trips) bits.push(`${s.trips} ${s.trips === 1 ? 'trip' : 'trips'}, ${dist(t.km, 0)} in ${hm(t.drivingSec)}`);
  if (e.addedKwh > 0) bits.push(`charged ${num(e.addedKwh, 0, ' kWh')}${e.cost > 0 ? ` (${money(x.currency, e.cost)})` : ''}`);
  if (!bits.length) return null;
  const yest = new Date(); yest.setDate(yest.getDate() - 1);
  return h('section', { class: 'card' }, h('h2', {}, 'Today'), h('p', {}, bits.join(' · ')),
    h('p', { class: 'seg' }, h('a', { class: 'btn sec', href: `#/day/${x.date}` }, 'Open today'), h('a', { class: 'btn sec', href: `#/day/${ymd(yest)}` }, 'Yesterday')));
}

// What charging has added up to this month, and the next service coming.
async function monthCard(maint) {
  const sum = await api('/api/charges/summary').catch(() => null);
  const m = sum && sum.thisMonth;
  const next = ((maint && maint.items) || []).filter((i) => i.state === 'ok' && (i.kmLeft != null || i.daysLeft != null))
    .sort((a, b) => (a.kmLeft ?? 1e9) - (b.kmLeft ?? 1e9) || (a.daysLeft ?? 1e9) - (b.daysLeft ?? 1e9))[0];
  if (!(m && m.kwh > 0) && !next) return null;
  const left = next ? [next.kmLeft != null ? `in ${dist(next.kmLeft, 0)}` : null, next.daysLeft != null ? `in ${next.daysLeft} days` : null].filter(Boolean).join(' or ') : '';
  return link('#/stats', 'This month',
    m && m.kwh > 0 ? h('p', {}, `${num(m.kwh, 0, ' kWh')} charged${m.cost > 0 ? `, ${money(sum.currency, m.cost)}` : ''}`) : null,
    next ? h('p', { class: 'muted' }, `Next service: ${next.name} ${left}`) : null);
}

export async function homeCards(maint) {
  const out = await Promise.all([todayCard(), monthCard(maint)]);
  return out.filter(Boolean);
}
