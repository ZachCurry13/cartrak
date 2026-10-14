import { h, chip } from '../ui.js';
import { api } from '../api.js';
import { copyButton } from '../phonehelp.js';
import { checkCard } from '../mileagecheck.js';

const store = {
  get: (k, d) => { try { return localStorage.getItem(`mileage.${k}`) || d; } catch (e) { return d; } },
  set: (k, v) => { try { localStorage.setItem(`mileage.${k}`, v); } catch (e) { /* ignore */ } },
};

const pad = (n) => String(n).padStart(2, '0');
const iso = (d) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
const monday = (d) => { const x = new Date(d.getFullYear(), d.getMonth(), d.getDate()); x.setDate(x.getDate() - ((x.getDay() + 6) % 7)); return x; };
const addDays = (d, n) => { const x = new Date(d); x.setDate(x.getDate() + n); return x; };
// The calendar week number (week 1 is the week with January 4th), as used by the weeks box.
function isoWeek(dateStr) {
  const d = new Date(`${dateStr}T12:00:00`);
  const th = new Date(d.getFullYear(), d.getMonth(), d.getDate() + 3 - ((d.getDay() + 6) % 7));
  const wk1 = new Date(th.getFullYear(), 0, 4);
  return { year: th.getFullYear(), week: 1 + Math.round(((th - wk1) / 86400000 - 3 + ((wk1.getDay() + 6) % 7)) / 7) };
}
// Ready-made periods, as from and to dates (both included).
const QUICK = [
  ['This week', () => { const a = monday(new Date()); return [a, addDays(a, 6)]; }],
  ['Last week', () => { const a = addDays(monday(new Date()), -7); return [a, addDays(a, 6)]; }],
  ['This month', () => { const n = new Date(); return [new Date(n.getFullYear(), n.getMonth(), 1), new Date(n.getFullYear(), n.getMonth() + 1, 0)]; }],
  ['Last month', () => { const n = new Date(); return [new Date(n.getFullYear(), n.getMonth() - 1, 1), new Date(n.getFullYear(), n.getMonth(), 0)]; }],
];

const field = (id, label, input) => h('div', {}, h('label', { for: id }, label), input);

export default async function mileage(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const year = new Date().getFullYear();
  const weeks = h('input', { id: 'mw', placeholder: 'For example 26-28', value: store.get('weeks', ''), inputmode: 'numeric', autocomplete: 'off' });
  const yr = h('input', { id: 'my', type: 'number', min: 2000, max: 2100, value: year });
  const from = h('input', { id: 'mf', type: 'date' });
  const to = h('input', { id: 'mt', type: 'date' });
  const desc = h('input', { id: 'md', value: store.get('desc', 'Customer Visits'), maxlength: 60 });
  const which = h('select', { id: 'mc' }, h('option', { value: 'work' }, 'Work trips only'), h('option', { value: 'all' }, 'All trips'), h('option', { value: 'personal' }, 'Personal trips only'));
  const rounding = h('select', { id: 'mr' }, h('option', { value: 'up' }, 'Round each trip up to the next mile'), h('option', { value: 'nearest' }, 'Round the day to the nearest mile'));
  rounding.value = store.get('rounding', 'up');
  const msg = h('p', { class: 'muted', role: 'status' });
  const out = h('div');
  const go = h('button', { onclick: build }, 'Build the log');
  const quick = h('p', { class: 'seg' }, ...QUICK.map(([label, range]) => h('button', { class: 'sec', onclick: () => {
    const [a, b] = range();
    weeks.value = '';
    from.value = iso(a);
    to.value = iso(b);
    build();
  } }, label)));

  async function build() {
    const p = new URLSearchParams({ category: which.value, description: desc.value.trim(), rounding: rounding.value });
    store.set('rounding', rounding.value);
    if (weeks.value.trim()) { p.set('weeks', weeks.value.trim()); p.set('year', yr.value); } else { p.set('from', from.value); p.set('to', to.value); }
    store.set('weeks', weeks.value.trim());
    store.set('desc', desc.value.trim());
    go.disabled = true;
    out.replaceChildren();
    msg.textContent = 'Building. The first time, each new place is looked up (about a second each), then remembered.';
    try {
      show(await api(`/api/mileage?${p}`));
      out.append(checkCard(p, build));
      msg.textContent = '';
    } catch (e) { msg.textContent = e.message; }
    go.disabled = false;
  }

  function show(d) {
    const r = d.report;
    const table = h('div', { class: 'tablewrap' }, h('table', { class: 'mtable' },
      h('thead', {}, h('tr', {}, ...['Date', 'Origin City', 'Destination City', 'Description', 'Total Miles Driven'].map((t) => h('th', {}, t)))),
      h('tbody', {}, ...r.rows.map((x) => h('tr', {}, h('td', {}, x.date), h('td', {}, x.origin), h('td', {}, x.destinations), h('td', {}, x.description), h('td', { class: 'num' }, String(x.miles))))),
      h('tfoot', {}, h('tr', {}, h('td', { colspan: 4 }, 'Total Mileage Recorded:'), h('td', { class: 'num' }, String(r.totalMiles))))));
    // Miles by calendar week, to check against a weekly sheet.
    const byWeek = new Map();
    for (const x of r.rows) { const w = isoWeek(x.date); const k = `${w.year}-${pad(w.week)}`; byWeek.set(k, (byWeek.get(k) || 0) + x.miles); }
    const weekly = byWeek.size > 1 ? h('p', { class: 'muted' }, 'By calendar week: ', [...byWeek].map(([k, m]) => `week ${Number(k.split('-')[1])}: ${m}`).join(' · ')) : null;
    out.replaceChildren(h('section', { class: 'card' }, h('h2', {}, `${d.from} to ${d.to}`),
      r.rows.length ? table : h('p', { class: 'muted' }, 'No finished trips of that kind in this period. CarTrak only has trips from after it was linked to your car.'),
      weekly,
      ...r.notes.map((n) => h('p', { class: 'banner' }, n)),
      r.rows.length && d.rate ? h('p', {}, 'Estimated deduction at ', `${d.currency}${d.rate.toFixed(2)} a mile: `, h('b', {}, `${d.currency}${(r.totalMiles * d.rate).toFixed(2)}`), h('span', { class: 'muted' }, ' (an estimate for your own planning, not tax advice)')) : null,
      r.rows.length ? h('p', { class: 'seg' }, copyButton(d.tsv, 'Copy rows to paste into your sheet'), copyButton(d.tsvFull, 'Copy with headings and total')) : null,
      r.pending ? h('p', {}, chip('work', `${r.pending} place${r.pending === 1 ? '' : 's'} left`), ' ', h('button', { class: 'sec', onclick: build }, 'Run again to finish')) : null,
      d.lookupOn ? null : h('p', { class: 'muted' }, 'City lookup is switched off in Settings, so only places already remembered are named.')));
  }

  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Mileage log')),
    h('p', { class: 'muted' }, 'A day-by-day business mileage log from your trips: origin city, each city where you parked (once, in order, without your starting city), a description and the miles. Copy it and paste it into your sheet.'),
    h('section', { class: 'card' }, h('h2', {}, 'Which period'), quick,
      h('div', { class: 'cols2' }, field('mw', 'Calendar weeks', weeks), field('my', 'Year', yr)),
      h('p', { class: 'muted' }, 'Calendar weeks are the numbered weeks of the year, Monday to Sunday (week 1 is the week with January 4th). Or leave weeks empty and choose dates:'),
      h('div', { class: 'cols2' }, field('mf', 'From', from), field('mt', 'To (included)', to)),
      h('h2', {}, 'Format'),
      h('div', { class: 'cols2' }, field('md', 'Description column', desc), field('mc', 'Which trips', which)),
      field('mr', 'Miles', rounding),
      h('p', {}, go), msg,
      h('p', { class: 'muted' }, 'To name cities, CarTrak asks OpenStreetMap from your server, only when you build a log, with locations rounded to about 100 m. Every answer is remembered. You can switch this off in Settings.')),
    out);
}
