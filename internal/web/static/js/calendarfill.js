import { h } from './ui.js';
import { api } from './api.js';

// The route planner's calendar buttons: fill the stops with a day's appointments (those with a place), in the order of the day.
// Home's "Plan this route" leaves the day here (planDay) and the planner fills it in when it opens.

const KEY = 'cartrak.planDay';
export const ymd = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
const localInput = (d) => new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16);

export function planDay(date) {
  try { sessionStorage.setItem(KEY, date); } catch { /* storage blocked: the planner opens empty */ }
  location.hash = '#/plan';
}

// st: { stops, when, best, back, draw(), note(text) } from the route planner.
export function calendarButtons(st) {
  const box = h('div');
  async function fill(date) {
    st.note('Reading the calendar...');
    try {
      const r = await api(`/api/calendar/day?date=${date}`);
      const timed = r.events.filter((e) => !e.allDay);
      const placed = timed.filter((e) => e.place);
      if (!placed.length) { st.note(timed.length ? 'None of the appointments has a place CarTrak could find. Add them by hand.' : 'No appointments that day.'); return; }
      st.stops.splice(1, st.stops.length - 1, ...placed.slice(0, 10).map((e) => ({ name: e.title, label: e.place.address ? `${e.title}, ${e.place.address}` : e.title, lat: e.place.lat, lon: e.place.lon })));
      st.best.checked = false; // the appointments' own order
      st.back.checked = true;
      const leave = new Date((placed[0].start - 30 * 60) * 1000);
      if (leave.getTime() > Date.now()) st.when.value = localInput(leave);
      st.draw();
      const missing = timed.length - placed.length;
      st.note(`${placed.length} appointment${placed.length === 1 ? '' : 's'} filled in, in the order of the day${missing ? `. ${missing} had no place CarTrak could find: add ${missing === 1 ? 'it' : 'them'} by hand` : ''}. Plan the route to see the battery and the weather.`);
    } catch (e) { st.note(e.message); }
  }
  api('/api/calendar/day').then((r) => {
    if (!r || !r.connected) return;
    const today = new Date(), tomorrow = new Date(Date.now() + 86400000);
    box.replaceChildren(h('p', { class: 'seg' }, h('button', { class: 'sec', type: 'button', onclick: () => fill(ymd(today)) }, "Today's appointments"),
      h('button', { class: 'sec', type: 'button', onclick: () => fill(ymd(tomorrow)) }, "Tomorrow's")));
  }).catch(() => {});
  let asked = null;
  try { asked = sessionStorage.getItem(KEY); sessionStorage.removeItem(KEY); } catch { /* storage blocked */ }
  if (asked) fill(asked);
  return box;
}
