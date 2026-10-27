import { api } from './api.js';

// The checks only a person with the car and a phone can finish. Each has what to do and what should happen. The automatic ones
// run when the page opens and say what they found.
export const CHECKS = [
  { id: 'alert-sound', title: 'Alerts arrive once, with the right sound', steps: 'Notifications → send a test alert. Then lock the car and leave it a while, or wait for the next real alert.',
    expect: 'The test alert arrives once, with a sound. Trip start, trip end, plug in and plug out arrive silently. An old "left unlocked" report is not repeated after the app updates.' },
  { id: 'badge', title: 'App icon badge', steps: 'Add CarTrak to the home screen, then let an alert arrive while the app is closed.',
    expect: 'The icon shows a count that clears when you open the app.' },
  { id: 'gas-cars', title: 'Live gas prices on every car', steps: 'Switch to each car and open Charging → Live gas prices.',
    expect: 'Each car shows when prices were last updated and no error. If one says no key, note which car.' },
  { id: 'stop-start', title: 'A work stop starts when the car stopped', steps: 'Drive somewhere, sit in the car with it on for about 15 minutes, then switch it off. Open More → Work stops after the next trip.',
    expect: 'The stop starts when the car stopped moving, not when you switched it off, and the minutes add up.' },
  { id: 'stop-label', title: 'Label a stop', steps: 'Under Work stops, open Label this stop, pick a category, add a note and save. Export CSV.',
    expect: 'The label stays after a reload, and the CSV has the note in its own column.' },
  { id: 'fix-day', title: 'Fix a day', steps: 'Open the day you wanted corrected (Oct 22 to 24) in Day view and run Fix a day.',
    expect: 'The trips for that day come out as you remember them: the right number, no split or merged drives.' },
  { id: 'late-drive', title: 'A drive the car reported late', steps: 'After a drive where the car had no signal, look at Trips.',
    expect: 'The drive is there, starting at the last reading before it. Note the date if it is missing.' },
  { id: 'restore', title: 'Restore from a backup', steps: 'Data → Backup: download a backup, then open the Restore check.',
    expect: 'The check says the backup can be read and shows the counts of trips and charges.' },
  { id: 'anything', title: 'Anything slow, odd or confusing', steps: 'Use the app as usual for a day.', expect: 'Write what you noticed and which page it was on.' },
];

const ms = async (path) => { const t = performance.now(); await api(path); return Math.round(performance.now() - t); };
const safe = async (f) => { try { return await f(); } catch (e) { return { ok: false, text: `could not check (${e.message || e})` }; } };
const ymd = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

// AUTO is what the app can see for itself on this device: [name, function(session) -> { ok, text }].
export const AUTO = [
  ['This version', async (session) => ({ ok: true, text: String((session.about && session.about.version) || 'unknown') })],
  ['Alerts on this device', async () => {
    const perm = typeof Notification === 'undefined' ? 'not supported here' : Notification.permission;
    const reg = 'serviceWorker' in navigator ? await navigator.serviceWorker.getRegistration() : null;
    const sub = reg && reg.pushManager ? await reg.pushManager.getSubscription() : null;
    return { ok: perm === 'granted' && !!sub, text: `permission ${perm}, ${sub ? 'subscribed' : 'not subscribed'}` };
  }],
  ['Savings page speed', async () => { const t = await ms('/api/savings'); const t2 = await ms('/api/savings'); return { ok: t2 < 300, text: `${t} ms first, ${t2} ms again` }; }],
  ['Live gas prices (this car)', async () => { const d = await api('/api/savings'); return { ok: !!d.live, text: d.live ? 'on' : 'off (needs a key, US units and city names on)' }; }],
  ['Work stops (last 30 days)', async () => {
    const to = new Date(), from = new Date();
    from.setDate(from.getDate() - 29);
    const d = await api(`/api/stops?from=${ymd(from)}&to=${ymd(to)}`);
    return { ok: true, text: `${d.stops.length} stop${d.stops.length === 1 ? '' : 's'}, ${Math.round(d.minutes)} min parked` };
  }],
].map(([name, fn]) => [name, (session) => safe(() => fn(session))]);
