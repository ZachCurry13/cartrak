import { h } from './ui.js';
import { api } from './api.js';
import { setBirthday, applySeason } from './seasons.js';

// A person's own birthday and holiday touches, kept with their sign-in so they follow them to every device.

export const loadPrefs = () => api('/api/me/prefs').catch(() => ({}));

// savePrefs changes some of the choices and keeps the rest.
export async function savePrefs(patch) {
  const p = await api('/api/me/prefs', { method: 'PUT', body: { ...(await loadPrefs()), ...patch } });
  setBirthday(p.birthday);
  applySeason();
  return p;
}

const MONTHS = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'];

// The birthday card on Display: a month and a day, never the year.
export async function birthdayCard(session) {
  const p = await loadPrefs();
  const [m0, d0] = (p.birthday || '').split('-');
  const month = h('select', { id: 'bm' }, h('option', { value: '' }, 'Not set'), ...MONTHS.map((n, i) => h('option', { value: String(i + 1).padStart(2, '0') }, n)));
  const day = h('select', { id: 'bd' }, ...Array.from({ length: 31 }, (_, i) => h('option', { value: String(i + 1).padStart(2, '0') }, String(i + 1))));
  month.value = m0 || '';
  day.value = d0 || '01';
  const fit = () => { // only the days the month has (February keeps the 29th)
    const n = [31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31][(Number(month.value) || 1) - 1];
    [...day.options].forEach((o, i) => { o.hidden = o.disabled = i >= n; });
    if (day.selectedIndex >= n) day.selectedIndex = n - 1;
  };
  month.addEventListener('change', fit);
  fit();
  const msg = h('p', { class: 'muted', role: 'status' });
  const save = h('button', { type: 'button', onclick: async () => {
    try {
      const b = month.value ? `${month.value}-${day.value}` : '';
      await savePrefs({ birthday: b });
      if (session && session.me) session.me.birthday = b;
      msg.textContent = b ? `Saved ✓ ${MONTHS[Number(month.value) - 1]} ${Number(day.value)}.` : 'Saved ✓ No birthday set.';
    } catch (e) { msg.textContent = e.message; }
  } }, 'Save birthday');
  return h('section', { class: 'card' }, h('h2', {}, 'Your birthday 🎂'),
    h('p', { class: 'muted' }, 'On your birthday CarTrak dresses up for you, with a happy birthday on your phone in the morning. Just the month and day; only you see it, on every device you sign in on.'),
    h('div', { class: 'cols2' }, h('div', {}, h('label', { for: 'bm' }, 'Month'), month), h('div', {}, h('label', { for: 'bd' }, 'Day'), day)),
    h('p', {}, save), msg);
}
