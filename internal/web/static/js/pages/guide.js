import { h } from '../ui.js';

// A map of everything CarTrak does and where to find it, so features are easy to rediscover.
export const GROUPS = [
  ['Right now', [
    ['#/', 'Home', 'Battery ring, range, what needs attention, the last 7 days, last trip, charging, location, tires.'],
    ['#/live', 'Live', 'One screen that follows the car: driving, charging or parked. Time to your charge target, trip time, driving today.'],
    ['#/charging', 'Charging', 'Every session with the energy added, what it cost, and your measured charging loss.'],
  ]],
  ['Trips and miles', [
    ['#/trips', 'Trips', 'All trips with efficiency, a map view, combining trips into journeys, and range by month and weather.'],
    ['#/review', 'Trips to review', 'Label trips that no rule could, with one tap.'],
    ['#/mileage', 'Mileage log', 'A business mileage log by week or month, rounded up per trip, with a check against the odometer.'],
    ['#/rules', 'Trip rules', 'Label trips Work or Personal by day, time and place.'],
    ['#/places', 'Places', 'Name Home, the office and others. Trips read "Home → Office" and you can get arrive and leave alerts.'],
  ]],
  ['Planning', [
    ['#/plan', 'Road-trip planner', 'How many charging stops a trip needs, and where, for the weather you pick.'],
    ['#/time', 'Time', 'How much of the time the car was parked, driving or charging, and where it was parked.'],
    ['#/insights', 'Efficiency insights', 'What changes your miles per kWh: temperature, month, best and worst trips.'],
    ['#/maintenance', 'Maintenance', 'Service reminders by distance or time.'],
    ['#/trophies', 'Trophies', 'Your collection of achievements: the ones earned and, grayed out, how far along the rest are.'],
    ['#/fuel', 'Fuel', 'For a gas or hybrid car: fill-ups with the gallons and the price per gallon, the tank size CarTrak works out, and what fuel costs.'],
  ]],
  ['The car itself', [
    ['#/health', 'Health', '12 V battery, main battery, tires, trouble codes, parked drain.'],
    ['#/data', 'Data', 'Everything the car reports in plain words, and the car as it was at any earlier time.'],
    ['#/parkedwatch', 'Parked watch', 'Experimental bump and alarm alerts from the car\'s motion sensors.'],
  ]],
  ['Alerts and sharing', [
    ['#/notifications', 'Notifications', 'Charging, plug-in reminders, trips, tires, service due, weekly summary and more, on this phone.'],
    ['#/trips', 'Share a trip', 'Open a trip or journey and use Share to make a public link you can turn off.'],
    ['#/cars', 'Cars', 'Follow more than one car, switch between them, add a car of your own, and choose who can see each one.'],
    ['#/security', 'Security', 'Who is signed in, the sign-in log, and your shared links (admins).'],
  ]],
  ['Looking after CarTrak', [
    ['#/year', 'Year in review', 'A summary of the year: miles, work and personal, energy, charging and cost.'],
    ['#/debug', 'Debug report', 'A copyable report of recent readings, trips, charging and alerts, to send when something looks wrong (admins).'],
    ['#/datacheck', 'Data check', 'Looks for gaps in the readings, an odometer that went back, and overlapping or impossible trips.'],
    ['#/backup', 'Backup', 'Download or restore a backup, and free up space (admins).'],
    ['#/remote', 'Remote access', 'Use CarTrak away from home, and update the Cloudflare connector (admins).'],
    ['#/status', 'Status', 'Is everything working? Ford, the tunnel, readings.'],
    ['#/about', 'About and updates', 'Version, what is new, and the parts CarTrak is built from.'],
  ]],
];

export default async function guide(root) {
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'What CarTrak can do')),
    h('p', { class: 'muted' }, 'Everything in one place. Tap a name to open it. Tap an efficiency number anywhere to change its unit, and tap the ? beside a value on the Data page to see what it means.'),
    ...GROUPS.map(([title, items]) => h('section', { class: 'card' }, h('h2', {}, title),
      ...items.map(([href, name, text]) => h('a', { class: 'row', href }, h('div', { class: 'grow' }, h('b', {}, name), h('div', { class: 'muted' }, text)))))));
}
