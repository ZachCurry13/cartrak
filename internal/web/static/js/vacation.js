import { h } from './ui.js';
import { api } from './api.js';
import { categoryList, catLabel } from './categories.js';

const fmt = (d) => new Date(`${d}T12:00:00`).toLocaleDateString([], { month: 'short', day: 'numeric' });
const addDays = (day, n) => { const d = new Date(`${day}T12:00:00`); d.setDate(d.getDate() + n); return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`; };

// What each kind of mode is called: Vacation makes trips Personal; every other category is a mode of its own, so Work trip mode
// and Medical mode are there, and so is one for each category the owner adds.
const MODE_NAME = { vacation: 'Vacation', work: 'Work trip', medical: 'Medical' };
export const modeName = (kind) => MODE_NAME[kind] || catLabel(kind);
const modeOptions = () => [['vacation', 'Vacation (trips are Personal)'], ...categoryList().filter((c) => c.key !== 'personal').map((c) => [c.key, `${modeName(c.key)} mode (trips are ${c.label})`])];
const outcome = (kind) => (kind === 'vacation' ? 'Personal' : catLabel(kind));

// Modes: for a stretch of days every trip gets the same label (Personal on vacation, Work on a work trip, Medical, or one of your
// own), without changing each one by hand. Rules still apply the rest of the time, and trips you set by hand keep what you chose.
export async function vacationCard(admin) {
  const box = h('section', { class: 'card' });
  async function draw() {
    const [d, named] = await Promise.all([api('/api/time-off'), admin ? api('/api/named-places').catch(() => []) : Promise.resolve([])]);
    const kind = h('select', { id: 'vk', 'aria-label': 'Mode' }, ...modeOptions().map(([v, t]) => h('option', { value: v }, t)));
    const from = h('input', { type: 'date', id: 'vf', value: d.today, 'aria-label': 'First day' });
    const to = h('input', { type: 'date', id: 'vt', value: addDays(d.today, 6), 'aria-label': 'Last day' });
    const label = h('input', { id: 'vl', maxlength: 60, placeholder: 'Where or why (optional)', 'aria-label': 'Label' });
    const msg = h('p', { class: 'muted', role: 'alert' });
    const now = d.items.find((x) => x.from <= d.today && x.to >= d.today);
    box.replaceChildren(...[
      h('h2', {}, 'Modes'),
      h('p', { class: 'muted' }, 'Choose the days and every trip in them gets one label: Personal on vacation, Work on a work trip, Medical, or a category of your own. You do not have to change them one by one. Trips you set by hand keep what you chose.'),
      now ? h('p', { class: 'banner notice' }, `${modeName(now.kind)} mode until ${fmt(now.to)}${now.label ? ` (${now.label})` : ''}.`) : null,
      ...d.items.map((x) => h('div', { class: 'row' }, h('span', { class: 'grow', style: 'flex-basis:12em' }, `${fmt(x.from)} to ${fmt(x.to)}`, h('span', { class: 'muted' }, ` · ${modeName(x.kind)}${x.label ? `, ${x.label}` : ''}`)),
        admin && x.to >= d.today ? baseSelect(x, named) : null,
        admin ? h('button', { class: 'sec', onclick: async () => { await api(`/api/time-off/${x.id}`, { method: 'DELETE' }); draw(); } }, x.from <= d.today && x.to >= d.today ? 'End now' : 'Remove') : null)),
      admin ? h('div', {}, h('label', { for: 'vk' }, 'Mode'), kind, h('label', { for: 'vf' }, 'First day'), from, h('label', { for: 'vt' }, 'Last day'), to, label,
        h('p', {}, h('button', { onclick: async () => {
          try { await api('/api/time-off', { method: 'POST', body: { kind: kind.value, from: from.value, to: to.value, label: label.value } }); draw(); } catch (e) { msg.textContent = e.message; }
        } }, 'Start mode')), msg,
        h('p', { class: 'muted' }, 'If two modes cover the same day, the one you added last wins. On a trip away, CarTrak finds your base from where the car spends the nights: time parked there is not a stop, and drives start from and return to it. Choose one of your places as the base if you prefer.')) : null,
    ].filter(Boolean));
  }
  await draw();
  return box;
}

// Where you are staying on a trip away. CarTrak finds it from where the car spends the nights; a named place can be chosen instead.
function baseSelect(x, named) {
  const sel = h('select', { 'aria-label': `Where you are staying, ${fmt(x.from)} to ${fmt(x.to)}`, style: 'max-width:100%' },
    h('option', { value: '' }, 'Base: where the car sleeps'), ...named.map((p) => h('option', { value: String(p.id) }, `Base: ${p.name}`)));
  sel.value = x.baseId ? String(x.baseId) : '';
  sel.addEventListener('change', async () => {
    try { await api(`/api/time-off/${x.id}`, { method: 'PATCH', body: { baseId: sel.value ? Number(sel.value) : null } }); } catch (e) { sel.value = x.baseId ? String(x.baseId) : ''; }
  });
  return sel;
}

// A short note for Home while a mode is on, or null.
export async function vacationNote() {
  const d = await api('/api/time-off').catch(() => null);
  const now = d && d.items.find((x) => x.from <= d.today && x.to >= d.today);
  return now ? h('a', { class: 'banner notice', href: '#/rules', style: 'display:block;color:inherit;text-decoration:none' }, `${modeName(now.kind)} mode until ${fmt(now.to)}: trips count as ${outcome(now.kind)}.`) : null;
}
