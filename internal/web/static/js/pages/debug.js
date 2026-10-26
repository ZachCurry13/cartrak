import { h } from '../ui.js';
import { api } from '../api.js';
import { phoneSection } from '../errlog.js';
import { currentCar } from '../car.js';

// A plain-text report to copy and send when something looks wrong. It has no positions, sign-ins, keys or VIN.
export default async function debug(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const hours = h('select', { id: 'dh', 'aria-label': 'How far back' },
    ...[[6, 'Last 6 hours'], [24, 'Last 24 hours'], [48, 'Last 2 days'], [72, 'Last 3 days'], [168, 'Last 7 days'], [336, 'Last 14 days']].map(([v, t]) => h('option', { value: v, selected: v === 24 }, t)));
  const box = h('textarea', { readonly: true, rows: 16, 'aria-label': 'Debug report', style: 'width:100%;box-sizing:border-box;font-family:ui-monospace,Menlo,Consolas,monospace;font-size:.75rem;white-space:pre;overflow:auto' });
  // Any car the person can open, one by one (an administrator of the whole installation can open every car), or all of them together.
  const cars = session.cars || [];
  const site = session.me.siteRole === 'admin';
  const which = h('select', { id: 'dw', 'aria-label': 'Which car' }, h('option', { value: '' }, 'The car on screen'),
    ...cars.map((c) => h('option', { value: `car:${c.id}` }, `${c.name}${c.theirs ? ' (not yours)' : ''}`)), site && cars.length > 1 ? h('option', { value: 'all' }, 'Every car, together') : null);
  const many = cars.length > 1;
  const msg = h('p', { class: 'muted', role: 'status' });
  const copy = h('button', { class: 'sec', disabled: true, onclick: async () => { try { await navigator.clipboard.writeText(box.value); msg.textContent = 'Copied'; } catch (e) { box.select(); msg.textContent = 'Select all and copy (the browser blocked the button)'; } } }, 'Copy');
  const save = h('a', { class: 'btn sec', hidden: true, download: 'cartrak-debug-report.txt' }, 'Download .txt');
  // cartrak-debug-<car>-<date>-<time>-<how far back>.txt, so a pile of reports sorts by day and says which car it is about
  function fileName() {
    const id = which.value === 'all' ? null : which.value ? Number(which.value.slice(4)) : Number(currentCar());
    const car = which.value === 'all' ? 'all-cars' : (cars.find((c) => c.id === id) || {}).name || 'car';
    const slug = String(car).toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'car';
    const d = new Date(), p = (n) => String(n).padStart(2, '0');
    return `cartrak-debug-${slug}-${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}-${hours.value}h.txt`;
  }
  async function make() {
    msg.textContent = 'Building the report…';
    try {
      const d = await api(`/api/debug-report?hours=${encodeURIComponent(hours.value)}${which.value === 'all' ? '&all=1' : which.value ? `&car=${which.value.slice(4)}` : ''}`);
      const text = d.text + phoneSection();
      box.value = text; copy.disabled = false;
      save.download = fileName();
      save.href = URL.createObjectURL(new Blob([text], { type: 'text/plain' })); save.hidden = false;
      msg.textContent = `${d.readings} readings. Read it before you send it, and delete anything you would rather not share.`;
    } catch (e) { msg.textContent = e.message; }
  }
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Debug report')),
    h('p', { class: 'muted' }, 'When something looks wrong (a trip that is too long, a false "unlocked", a charge that does not add up), make a report right after, copy it and send it. It lists what the car reported, the trips and charging CarTrak made from it, and the alerts it sent. It leaves out where the car was, sign-ins, keys and the VIN. Choose how far back the problem was.'),
    h('section', { class: 'card' }, h('p', {}, many ? which : null, many ? ' ' : null, hours, ' ', h('button', { onclick: make }, 'Make report')), box, h('p', { class: 'seg' }, copy, save), msg));
}
