import { h, chip } from '../ui.js';
import { api } from '../api.js';

// Which setting sits in which box, in order.
const SECTIONS = [
  ['Prices', 'What a kWh costs you. These work out the cost of every charging session that has no receipt entered.',
    ['CARTRAK_PRICE_HOME', 'CARTRAK_PRICE_PUBLIC', 'CARTRAK_CURRENCY', 'CARTRAK_PRICE_OFFPEAK', 'CARTRAK_OFFPEAK_WINDOW']],
  ['Accuracy', 'How much of the energy from a home charger reaches the battery.', ['CARTRAK_CHARGE_EFFICIENCY']],
  ['Compared with a gas car', 'The gas car you would otherwise drive. With these, CarTrak shows what you save and the CO2 you avoid. Leave them empty to turn the comparison off.',
    ['CARTRAK_FUEL_UNITS', 'CARTRAK_GAS_PRICE', 'CARTRAK_GAS_ECONOMY', 'CARTRAK_GRID_CO2']],
  ['Live gas prices (optional)', 'With a free EIA key, CarTrak compares each charge with the average price of gas where and when it happened. Your gas price above is used until a live price is known, or without a key. The key and the place adjustments start the same for every car; a car can set its own.',
    ['CARTRAK_EIA_KEY', 'CARTRAK_GAS_ADJUST']],
  ['Alerts about charging', 'Each person still chooses which alerts reach their own device under More, Notifications.',
    ['CARTRAK_PLUG_REMINDER_TIME', 'CARTRAK_PLUG_REMINDER_BELOW', 'CARTRAK_LOW_BATTERY', 'CARTRAK_CHARGE_BUDGET']],
];

const HOME_TEXT = {
  place: 'CarTrak knows where home is: you named a place Home. Charging within its size counts as home and uses the home price.',
  marked: 'CarTrak knows where home is: you marked a charging session as Home. Charging near it counts as home.',
  none: 'CarTrak does not know where home is yet, so every charge is priced at the home rate and none counts as home in the statistics. Name a place Home under Places, or mark a session as Home on the Charging page.',
};

const WORDS = new Set(['CARTRAK_EIA_KEY', 'CARTRAK_GAS_ADJUST', 'CARTRAK_CURRENCY', 'CARTRAK_OFFPEAK_WINDOW']); // text, not numbers

function field(s) {
  const el = s.options
    ? h('select', { id: `f-${s.key}`, disabled: s.locked }, ...s.options.map((o) => h('option', { value: o.value }, o.label)))
    : h('input', { id: `f-${s.key}`, type: s.secret ? 'password' : 'text', inputmode: WORDS.has(s.key) ? 'text' : 'decimal', autocomplete: 'off', value: s.value,
      disabled: s.locked, placeholder: s.secret && s.isSet ? 'Saved. Type a new one to replace it.' : s.placeholder || '' });
  if (s.options) el.value = s.value || '';
  return [el, h('div', { class: 'muted', style: 'margin:4px 0 12px' }, s.help)];
}

// Everything about what charging costs and alerts about it, on one page for the owner of the car.
export default async function chargingSettings(root, _a, session) {
  const admin = session.me.role === 'admin';
  root.append(h('p', {}, h('a', { href: '#/charging' }, '← Charging')), h('div', { class: 'page-title' }, h('h2', {}, 'Charging settings')));
  if (!admin) return root.append(h('p', { class: 'muted' }, 'Only the owner of the car can change these. Ask them if a price looks wrong.'));
  const [data, home, ps] = await Promise.all([api('/api/admin/settings?group=charging'), api('/api/charging/home').catch(() => ({ source: 'none' })), api('/api/named-places').catch(() => [])]);
  const byKey = Object.fromEntries(data.settings.map((s) => [s.key, s]));
  const inputs = {};
  const msg = h('p', { class: 'muted', role: 'status' });

  const cards = SECTIONS.map(([title, intro, keys]) => h('section', { class: 'card' }, h('h2', {}, title), h('p', { class: 'muted' }, intro),
    ...keys.filter((k) => byKey[k]).flatMap((k) => {
      const s = byKey[k];
      const [el, help] = field(s);
      inputs[k] = el;
      return [h('label', { for: `f-${k}` }, s.label, ' ', s.locked ? chip('work', 'set in the server settings') : null), el, help];
    })));

  async function save() {
    const values = {};
    for (const [k, el] of Object.entries(inputs)) if (!byKey[k].locked) values[k] = el.value.trim();
    try { await api('/api/admin/settings?group=charging', { method: 'PUT', body: { values } }); msg.textContent = 'Saved. Costs are worked out again with the new prices.'; } catch (e) { msg.textContent = e.message; }
  }

  const special = ps.filter((p) => p.chargePrice != null || p.chargeHome);
  root.append(h('p', { class: 'muted' }, 'Prices, the efficiency of your charger and the charging alerts. They apply to this car only.'),
    h('section', { class: 'card' }, h('h2', {}, 'Where home is'), h('p', {}, HOME_TEXT[home.source] || HOME_TEXT.none),
      h('p', { class: 'muted' }, 'A cabin, a relative\'s house or a workplace charger can have its own price, or be free: open Places and set the price under the place.'),
      special.length ? h('div', {}, ...special.map((p) => h('div', { class: 'row' }, h('span', { class: 'grow' }, p.name),
        chip(p.chargeHome ? 'ok' : 'work', p.chargeHome ? 'counts as home' : p.chargePrice === 0 ? 'free' : `${data.settings.find((s) => s.key === 'CARTRAK_CURRENCY')?.value || '$'}${p.chargePrice.toFixed(2)} per kWh`)))) : null,
      h('p', {}, h('a', { class: 'btn sec', href: '#/places' }, 'Open Places'))),
    ...cards,
    h('p', { class: 'seg' }, h('button', { onclick: save }, 'Save')), msg);
}
