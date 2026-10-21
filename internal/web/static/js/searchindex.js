import { api } from './api.js';
import { GROUPS } from './pages/guide.js';

// Things people look for by a word that is not the name of the page they live on. Each says where it is:
// [title, where, href, what it is, extra words people might type].
const EXTRA = [
  ['Vacation', 'More → Trip rules', '#/rules', 'Pick the days you are away and every trip is Personal, so you do not change them one by one.', 'vacation holiday time off away trip personal work'],
  ['Work or personal', 'More → Trip rules', '#/rules', 'Rules that label trips by day, time and place.', 'work personal business label category rule hours'],
  ['Electricity price', 'Charging → Charging settings', '#/charging-settings', 'Price per kWh at home and away, and the currency, for charging costs.', 'electricity price cost kwh rate currency money bill utility'],
  ['Cheaper night rates', 'Charging → Charging settings', '#/charging-settings','A cheaper home price for certain hours, such as 22:00-06:00.', 'night off peak offpeak time of use tou cheaper rate'],
  ['Mileage deduction rate', 'More → Settings', '#/admin', 'What a business mile is worth, for the estimate in the Mileage log and Year in review.', 'mileage deduction tax irs rate per mile business'],
  ['Time zone', 'More → Settings', '#/admin', 'The time zone the app uses for rules, reports and the weekly summary.', 'time zone timezone clock hour local'],
  ['Ford sign-in and credentials', 'More → Ford connection', '#/ford', 'Link or relink your Ford account.', 'ford account link login connect credentials client id secret relink'],
  ['Miles or kilometers, °F or °C', 'More → Display and data', '#/display', 'Units, theme, trip times and data use for this device.', 'units miles kilometers km metric imperial fahrenheit celsius temperature psi kpa'],
  ['Dark or light theme', 'More → Display and data', '#/display', 'Choose the look for this device.', 'theme dark light mode appearance color color'],
  ['Export every car field', 'More → Data → All fields', '#/data', 'A zip of spreadsheets: every field the car sends, which never changed, and every moment something did. Choose how far back.', 'export download csv spreadsheet fields raw data history changes zip over time'],
  ['Trophies', 'Insights → Trophies', '#/trophies', 'Every trophy you can earn, the ones you have and how far along the rest are.', 'trophies achievements badges awards collection milestones streak goals'],
  ['Most visited places', 'Insights → Highlights', '#/analytics', 'The places and towns you went to most in the last 12 months.', 'popular most visited places towns favorite regular where'],
  ['Fuel and fill-ups', 'More → Fuel', '#/fuel', 'Fill-ups with the gallons and the price per gallon, the tank size CarTrak learns, and fuel cost, for a gas or hybrid car.', 'fuel gas gasoline petrol fill-up fill up fillup gallons liters price per gallon tank mpg fuel economy refuel hybrid'],
  ['Your birthday', 'More → Display and data', '#/display', 'Set your birthday, and CarTrak dresses up for you that day with a happy birthday alert in the morning.', 'birthday birth day party cake celebrate'],
  ['Holiday themes', 'More → Display and data', '#/display', 'Seasonal colors, an ornament and a short effect on Home that follow the holidays, or off.', 'holiday holidays seasonal season theme christmas halloween easter decorations off'],
  ['Holiday touches on alerts', 'More → Notifications', '#/notifications', 'Friendly alerts get the holiday emoji and a happy holiday on the day. Warnings stay plain.', 'holiday alerts emoji notifications festive birthday touches'],
  ['All cars together', 'Car picker → All cars', '#/fleet', 'Every car side by side, the month and year together, and one trip list.', 'all cars fleet together combined household both compare side by side'],
  ['Data saver', 'More → Display and data', '#/display','Use less mobile data on this device.', 'data saver mobile data cellular bandwidth live updates'],
  ['Efficiency unit (mi/kWh, Wh/mi)', 'Tap any efficiency number', '#/insights', 'Tap an efficiency figure anywhere to switch between mi/kWh, Wh/mi and mi per percent.', 'efficiency unit mi/kwh wh/mi miles per percent consumption'],
  ['Quiet hours for alerts', 'More → Notifications', '#/notifications', 'Pick hours when this device gets no alerts.', 'quiet hours do not disturb night mute silence'],
  ['Phone tracking and GPS', 'More → Phone tracking', '#/phone', 'Trip start from your phone and a finer route.', 'phone tracking gps owntracks shortcut automation bluetooth location'],
  ['Share a trip', 'Trips → open a trip → Share', '#/trips', 'Make a public link to a trip or journey that you can turn off.', 'share link public journey send show friend'],
  ['Home Assistant', 'More → Settings', '#/admin', 'A read-only feed of the car state for Home Assistant.', 'home assistant ha feed integration smart home sensor'],
  ['Cloudflare tunnel', 'More → Remote access', '#/remote', 'Reach CarTrak away from home.', 'cloudflare tunnel remote access away outside domain https'],
  ['Add family or a viewer', 'More → People', '#/people', 'Give family read-only accounts.', 'family viewer account user people password admin invite'],
  ['Reports and insights', 'More → Reports', '#/reports', 'Year in review, statistics, efficiency insights, mileage and spreadsheets in one place.', 'reports report year review statistics insights efficiency mileage export csv spreadsheet summary'],
  ['Charging settings', 'Charging → Charging settings', '#/charging-settings', 'Prices, cheaper night rates, where home is, and charging alerts.', 'charging price cost kwh rate electricity night home currency budget efficiency settings'],
  ['Trip categories and modes', 'More → Trip rules → Trip categories', '#/categories', 'Work, Personal, Medical and your own labels, with a color and a mileage rate for each.', 'category categories label medical work personal charity custom mode vacation work trip medical mode mileage rate'],
  ['Add or switch cars', 'More → Cars', '#/cars', 'Follow another car, switch between cars, and share a car.', 'car cars second vehicle switch add truck family another own'],
  ['See recent alerts (the bell)', 'Header → bell', '#/inbox', 'Every alert about your cars, newest first, even with push off.', 'inbox notifications alerts bell history unread messages'],
  ['Help and quick settings', 'Header → ?', '#/help', 'Units, theme, data use and a getting-started checklist.', 'help getting started quick settings units theme data install home screen'],
  ['Insights', 'More → Insights', '#/analytics', 'Wrapped for a month or the year, everywhere you have been, achievements, where the energy went and detailed charts.', 'insights wrapped recap year month map heatmap everywhere achievements badges charts graphs analytics records personality infographic'],
  ['Service history', 'More → Maintenance → Service history', '#/service', 'Every service visit with what was done and what it cost; log one, or answer one CarTrak noticed at a service place.', 'service history maintenance dealership dealer tire shop repair visit oil rotation cost log'],
  ['Setup checklist', 'More → Setup checklist', '#/setup', 'Suggested next steps, all optional: install, alerts, places, prices and more.', 'setup checklist getting started todo to do suggestions new'],
  ['Restore or download a backup', 'More → Backup', '#/backup', 'Save a copy of your data or put one back.', 'backup restore export download zip csv spreadsheet save copy'],
  ['Delete a bad trip', 'Trips → open the trip → Fix this trip', '#/trips', 'Delete a duplicate or close a trip that never ended.', 'delete remove bad trip duplicate close stuck'],
  ['Tire pressure', 'Health', '#/health', 'Each tire against the placard and a slow-leak check.', 'tire tyre pressure psi leak flat placard'],
  ['12 V battery', 'Health', '#/health', 'The small battery that wakes the car.', '12v 12 volt battery small battery dead jump'],
  ['Battery health', 'Health', '#/health', 'Battery size over time and the true battery level.', 'battery health degradation capacity usable true level soc'],
  ['Charging curve', 'Charging → a session → Charging curve', '#/charging', 'Power against battery level for a session.', 'charging curve speed kw power peak fast charger supercharger taper'],
  ['Best speed for a trip', 'Trips → Plan a trip', '#/plan', 'Compare two cruising speeds on a long trip: driving time saved against charging time added.', 'best speed ideal speed 70 75 compare faster slower road trip planner charging time stops'],
  ['Day view', 'Trips → Day view', '#/day', 'All of a day on one map: every trip, the stops between them and the totals.', 'day view map route whole day combined stops charging trip stops'],
  ['Field watcher', 'Data → Field watcher', '#/fields', 'Every reading the car sends, how often it changes and what CarTrak uses it for.', 'fields watcher every field unused data changes api readings explore'],
  ['History', 'Trips → History feed', '#/history', 'Driving, charging and parked time as one feed, by month.', 'history timeline feed log parked driving charging month'],
  ['Statistics', 'Charging → Statistics', '#/stats', 'Distance, cost, energy and time by day, month and year; charging at home, work and public.', 'statistics stats chart bar month year cost kwh distance home work public donut'],
];

const TABS = new Set(['#/', '#/trips', '#/live', '#/charging', '#/health', '#/data']); // pages with a tab along the bottom
const norm = (s) => String(s || '').toLowerCase().replace(/[^a-z0-9+/ ]+/g, ' ');
const stem = (w) => (w.length > 3 ? w.replace(/(ing|es|s)$/, '') : w);

let cached = null;

// Everything searchable: the page map, the extras above, the real setting names and the real alert names.
export async function buildIndex(admin) {
  if (cached) return cached;
  const out = [];
  for (const [group, items] of GROUPS) {
    for (const [href, name, text] of items) out.push({ title: name, where: TABS.has(href) ? `The ${name} tab` : `More → ${name}`, href, text, keys: group });
  }
  for (const [title, where, href, text, keys] of EXTRA) out.push({ title, where, href, text, keys });
  const settings = admin ? await api('/api/admin/settings').catch(() => null) : null;
  for (const s of (settings && settings.settings) || []) {
    if (s.secret) continue;
    out.push({ title: s.label, where: 'More → Settings', href: '#/admin', text: s.help || '', keys: String(s.key).replace(/_/g, ' ') });
  }
  const cs = admin ? await api('/api/admin/settings?group=charging').catch(() => null) : null;
  for (const s of (cs && cs.settings) || []) out.push({ title: s.label, where: 'Charging → Charging settings', href: '#/charging-settings', text: s.help || '', keys: `${String(s.key).replace(/_/g, ' ')} price cost charging electricity` });
  const push = await api('/api/push/config').catch(() => null);
  for (const k of (push && push.kinds) || []) out.push({ title: `${k.label} (alert)`, where: 'More → Notifications', href: '#/notifications', text: k.help || '', keys: 'alert notification notify push' });
  cached = out;
  return out;
}

// The best matches for what was typed: every word must appear, titles count most.
export function search(index, query, limit = 6) {
  const words = norm(query).split(/\s+/).filter(Boolean).map(stem);
  if (!words.length) return [];
  const scored = [];
  for (const e of index) {
    const t = norm(e.title), k = norm(e.keys), x = norm(e.text);
    let score = 0;
    let all = true;
    for (const w of words) {
      const inT = t.includes(w), inK = k.includes(w), inX = x.includes(w);
      if (!inT && !inK && !inX) { all = false; break; }
      score += (inT ? 6 : 0) + (inK ? 3 : 0) + (inX ? 1 : 0) + (t.startsWith(w) ? 2 : 0);
    }
    if (all) scored.push([score, e]);
  }
  scored.sort((a, b) => b[0] - a[0]);
  const seen = new Set();
  const res = [];
  for (const [, e] of scored) {
    const id = `${e.title}|${e.href}`;
    if (seen.has(id)) continue;
    seen.add(id);
    res.push(e);
    if (res.length >= limit) break;
  }
  return res;
}
