// Fix a day: each change a rebuild would make, as one row: what is stored now on the left, what it becomes on the right, what changed
// written under them, and a box to untick the ones that should stay as they are.
import { h, dist, when, duration, num } from './ui.js';

const mins = (s) => Math.round(Math.abs(s) / 60);
const shift = (label, s) => (Math.abs(s) >= 120 ? `${label} ${mins(s)} min ${s < 0 ? 'earlier' : 'later'}` : null);
const socLine = (o, n) => (o.startSoc != null && n.startSoc != null && (Math.abs(o.startSoc - n.startSoc) >= 1 || Math.abs(o.endSoc - n.endSoc) >= 1)
  ? `Battery ${num(o.startSoc, 0)}% to ${num(o.endSoc, 0)}% becomes ${num(n.startSoc, 0)}% to ${num(n.endSoc, 0)}%` : null);

// What changed, in plain words: how many rows became how many, and for one row that became one, which numbers moved.
export function describe(kind, olds, news) {
  const word = kind === 'trip' ? 'trip' : 'charging session', s = (k) => (k === 1 ? '' : 's');
  if (!olds.length) return [`Found in the readings: ${news.length} ${word}${s(news.length)} that was not recorded`];
  if (!news.length) return [`Not in the readings: ${olds.length} ${word}${s(olds.length)} that should not be there`];
  if (olds.length === 1 && news.length > 1) return [`Split into ${news.length} ${word}s`];
  if (olds.length > 1 && news.length === 1) return [`${olds.length} ${word}s joined into one`];
  if (olds.length > 1) return [`${olds.length} ${word}s become ${news.length}`];
  const o = olds[0], n = news[0];
  const out = [shift('Starts', n.startTs - o.startTs), shift('Ends', (n.endTs || 0) - (o.endTs || 0)), socLine(o, n)];
  if (kind === 'trip' && Math.abs(n.distanceKm - o.distanceKm) >= 0.2) out.push(`Distance ${dist(o.distanceKm)} becomes ${dist(n.distanceKm)}`);
  if (kind === 'charge' && o.endKwh != null && n.endKwh != null && Math.abs((n.endKwh - n.startKwh) - (o.endKwh - o.startKwh)) >= 0.3) out.push(`Energy added ${num(o.endKwh - o.startKwh, 1, ' kWh')} becomes ${num(n.endKwh - n.startKwh, 1, ' kWh')}`);
  return out.filter(Boolean).length ? out.filter(Boolean) : ['Small differences in the figures'];
}

function side(kind, rows) {
  if (!rows.length) return [h('p', { class: 'muted rc-none' }, 'Nothing')];
  return rows.map((r) => h('div', { class: 'rc-item' },
    h('b', {}, kind === 'trip' ? `${dist(r.distanceKm)} · ${duration(r.startTs, r.endTs)}` : `Charging ${num(r.startSoc, 0)}% to ${num(r.endSoc, 0)}%`),
    h('div', { class: 'muted' }, when(r.startTs))));
}

// One change: the box, the old and new side by side, and the notes. onTick runs whenever a box is changed.
export function changeRow(c, p, onTick) {
  const pick = (list, ix) => ix.map((i) => list[i]);
  const olds = pick(c.kind === 'trip' ? p.oldTrips : p.oldCharges, c.old), news = pick(c.kind === 'trip' ? p.newTrips : p.newCharges, c.new);
  const box = h('input', { type: 'checkbox', class: 'check', checked: true, id: `rc-${c.key}`, 'aria-label': 'Apply this change' });
  const first = (olds[0] || news[0]).startTs;
  const row = h('section', { class: 'card rc' },
    h('label', { class: 'rc-head', for: box.id }, box, h('span', { class: 'rc-title' }, `${c.kind === 'trip' ? 'Trip' : 'Charge'} · ${when(first)}`)),
    h('div', { class: 'rc-cols' },
      h('div', {}, h('div', { class: 'tag' }, 'Now'), ...side(c.kind, olds)),
      h('div', { class: 'rc-arrow', 'aria-hidden': 'true' }, '→'),
      h('div', {}, h('div', { class: 'tag' }, 'After'), ...side(c.kind, news))),
    h('p', { class: 'tags' }, ...describe(c.kind, olds, news).map((t) => h('span', { class: 'chip' }, t))));
  box.addEventListener('change', () => { row.classList.toggle('off', !box.checked); onTick(); });
  return { row, box, key: c.key };
}
