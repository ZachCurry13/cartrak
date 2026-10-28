import { h } from '../ui.js';
import { api } from '../api.js';
import { withCar } from '../car.js';
import { trackCheck } from '../tracker.js';

const HOURS = [1, 2, 4, 8, 12];
const clock = (ts) => new Date(ts * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });

// Developer: the tools for working out what went wrong, in one place. Tracking records everything while the owner uses the app and
// exports it when they stop.
export default async function developer(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const box = h('section', { class: 'card' });
  const msg = h('p', { class: 'muted', role: 'status' });
  const priv = h('input', { type: 'checkbox', id: 'trk-private', checked: true });
  const hours = h('select', { id: 'trk-hours', 'aria-label': 'Stop by itself after' }, ...HOURS.map((n) => h('option', { value: n, selected: n === 4 }, `${n} hour${n === 1 ? '' : 's'}`)));
  let timer = null;
  // An administrator of the whole installation with more than one car can track every car at once.
  const many = session.me.siteRole === 'admin' && (session.cars || []).length > 1;
  const allBox = many ? h('input', { type: 'checkbox', id: 'trk-all' }) : null;
  const allRow = many ? h('label', { for: 'trk-all', style: 'display:flex;gap:10px;align-items:center;min-height:44px' }, allBox, ' Every car, not only this one') : null;
  const q = () => (allBox && allBox.checked ? 'all=1' : '');

  async function act(path, body) {
    msg.textContent = '';
    try { await api(path + (q() ? `?${q()}` : ''), { method: 'POST', body: body || {} }); await trackCheck(true); await draw(); } catch (e) { msg.textContent = e.message; }
  }
  async function draw() {
    const s = await api('/api/track', { noOffline: true });
    const kids = [h('h2', {}, 'Tracking'), allRow];
    if (s.active) {
      kids.push(h('p', {}, `On since ${clock(s.from)}. It stops by itself at ${clock(s.until)}. ${s.events} thing${s.events === 1 ? '' : 's'} recorded so far.`),
        h('p', { class: 'seg' }, h('button', { onclick: () => act('/api/track/stop') }, 'Stop tracking'), h('button', { class: 'sec', onclick: () => act('/api/track/extend') }, 'Add an hour')));
    } else {
      kids.push(h('p', { class: 'muted' }, 'Start it, use the app, then stop it. You can then download everything that happened in that time. While it is on, a red Tracking button shows on every page of this phone.'),
        h('p', {}, h('label', { for: 'trk-hours' }, 'Stop by itself after '), hours), h('p', {}, h('button', { onclick: () => act('/api/track/start', { hours: Number(hours.value) }) }, 'Start tracking')));
    }
    if (s.has && !s.active) {
      kids.push(h('h3', {}, 'Last session'), h('p', {}, `${clock(s.from)} to ${clock(s.to)}, ${s.events} app event${s.events === 1 ? '' : 's'}.`),
        h('label', { for: 'trk-private', style: 'display:flex;gap:10px;align-items:center;min-height:44px' }, priv, ' Include positions and the VIN (keep the file private)'),
        h('p', {}, h('a', { class: 'btn', href: '#', onclick: (e) => { e.preventDefault(); location.href = withCar(`/api/track/export?private=${priv.checked ? 1 : 0}${q() ? `&${q()}` : ''}`); } }, 'Download everything (.zip)')));
    }
    kids.push(msg);
    box.replaceChildren(...kids.filter(Boolean));
  }

  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Developer')),
    h('p', { class: 'muted' }, 'Tools for finding out why something looks wrong.'),
    h('section', { class: 'card' }, h('h2', {}, 'Reports'),
      h('a', { class: 'row', href: '#/debug' }, h('span', { class: 'grow' }, 'Debug report'), '›'),
      h('a', { class: 'row', href: '#/fieldcheck' }, h('span', { class: 'grow' }, 'Field check'), '›'),
      h('a', { class: 'row', href: '#/fields' }, h('span', { class: 'grow' }, 'Field watcher'), '›')), box);
  await draw();
  timer = setInterval(() => { if (!document.body.contains(box)) clearInterval(timer); else draw().catch(() => {}); }, 15000);
}
