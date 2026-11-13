import { h, chip } from './ui.js';
import { api } from './api.js';
import { carWarnings, warningLine } from './wxalerts.js';
import { planDay, ymd } from './calendarfill.js';

const clock = (ts) => new Date(ts * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });

// Home's "Today" card (from 6 pm, "Tomorrow"): the calendar's appointments with the place each one goes to, and the weather warnings
// where the car is. Nothing at all when there is no calendar and no warning.
export function todayCard(admin) {
  const box = h('div');
  (async () => {
    const evening = new Date().getHours() >= 18;
    const day = evening ? new Date(Date.now() + 86400000) : new Date();
    const [cal, wx] = await Promise.all([admin ? api(`/api/calendar/day?date=${ymd(day)}`).catch(() => null) : null, carWarnings().catch(() => [])]);
    const events = cal && cal.connected ? cal.events : null;
    if (!events && !wx.length) return;
    const placed = (events || []).filter((e) => !e.allDay && e.place);
    const row = (e) => h('div', { class: 'row', style: 'gap:8px;align-items:flex-start;flex-wrap:wrap;padding:6px 0' },
      h('b', { style: 'flex:0 0 5.5em' }, e.allDay ? 'All day' : clock(e.start)),
      h('div', { style: 'flex:1 1 10em;min-width:0;overflow-wrap:anywhere' }, e.title || '(no title)',
        e.allDay ? null : h('div', { class: 'muted' }, e.place ? (e.place.address || e.place.name) : e.location ? `${e.location} (place not found)` : 'No place given')));
    box.replaceChildren(h('section', { class: 'card' }, h('h2', {}, evening ? 'Tomorrow' : 'Today'),
      ...wx.map((a) => warningLine(a, 'the car')),
      events ? (events.length ? h('div', {}, ...events.map(row)) : h('p', { class: 'muted' }, 'No appointments on the calendar.')) : null,
      cal && cal.error ? h('p', { class: 'muted' }, `The calendar could not be read just now: ${cal.error}`) : null,
      placed.length ? h('p', {}, h('button', { class: 'sec', type: 'button', onclick: () => planDay(ymd(day)) }, `Plan ${evening ? "tomorrow's" : "today's"} route`), ' ', chip('ok', `${placed.length} stop${placed.length === 1 ? '' : 's'}`)) : null));
  })().catch(() => box.replaceChildren());
  return box;
}
