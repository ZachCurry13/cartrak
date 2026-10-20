import { h, chip, when, isMetric } from './ui.js';
import { api } from './api.js';
import { copyButton } from './phonehelp.js';

// "Share" on a trip or journey: make a public link, choosing how much it shows. The link is shown once.
export function shareCard(kind, id, defaultTitle = '') {
  const title = h('input', { id: 'sht', placeholder: defaultTitle || 'Optional title', maxlength: 80, value: '' });
  const mode = h('select', { id: 'shm' },
    h('option', { value: 'trimmed' }, 'Route, leaving out the first and last quarter mile (recommended)'),
    h('option', { value: 'stats' }, 'Numbers only, no route'),
    h('option', { value: 'full' }, 'The whole route, including where it started and ended'));
  const cities = h('input', { type: 'checkbox', id: 'shc' });
  const time = h('input', { type: 'checkbox', id: 'sht2' });
  const insights = h('input', { type: 'checkbox', id: 'shi', checked: true });
  const expires = h('select', { id: 'she' }, h('option', { value: '7' }, 'In 7 days'), h('option', { value: '30', selected: true }, 'In 30 days'), h('option', { value: '90' }, 'In 90 days'), h('option', { value: '0' }, 'Never'));
  const msg = h('p', { class: 'muted', role: 'status' });
  const result = h('div');
  const check = (box, boxId, text) => h('label', { for: boxId, style: 'display:flex;gap:8px;align-items:center;color:var(--text)' }, box, text);

  async function make() {
    msg.textContent = 'Creating the link…';
    try {
      const r = await api('/api/shares', { method: 'POST', body: { kind, id: Number(id), title: title.value.trim() || defaultTitle, mode: mode.value, showCities: cities.checked, showInsights: insights.checked, showTime: time.checked, units: isMetric() ? 'km' : 'mi', expiresDays: Number(expires.value) } });
      const url = r.publicAddressKnown ? r.url : `${location.origin}${r.path}`;
      msg.textContent = '';
      result.replaceChildren(h('p', { class: 'banner notice' }, 'Anyone with this link can open this page. You will not see the link again, so copy it now. You can turn it off any time under More, Security.'),
        h('pre', { class: 'code' }, url), h('p', { class: 'seg' }, copyButton(url, 'Copy link'), navigator.share ? h('button', { class: 'sec', onclick: () => navigator.share({ title: title.value.trim() || defaultTitle || 'CarTrak', url }).catch(() => {}) }, 'Share…') : null),
        r.publicAddressKnown ? null : h('p', { class: 'muted' }, 'Remote access has no public address yet, so this link only works where this address can be reached. Set up Remote access to share outside your home.'));
    } catch (e) { msg.textContent = e.message; }
  }

  return h('section', { class: 'card' }, h('h2', {}, 'Share'),
    h('p', { class: 'muted' }, 'Make a link anyone can open without signing in. It shows only what you choose here, never your VIN, your live position or your other trips.'),
    h('label', { for: 'sht' }, 'Title'), title,
    h('label', { for: 'shm' }, 'Route'), mode,
    check(cities, 'shc', 'Show the city names where it started and ended'),
    check(time, 'sht2', 'Show the date and time'),
    check(insights, 'shi', kind === 'trip' ? 'Show the efficiency details: speed, temperature and what affected it' : 'Show the speed details'),
    h('label', { for: 'she' }, 'The link stops working'), expires,
    h('p', {}, h('button', { onclick: make }, 'Create link')), msg, result);
}

// The links that exist, for the Security page. A link can be turned off here.
export async function sharesList() {
  const box = h('section', { class: 'card' });
  async function load() {
    const xs = await api('/api/shares');
    const now = Date.now() / 1000;
    box.replaceChildren(...[h('h2', {}, 'Shared links'),
      h('p', { class: 'muted' }, 'Public links you made from a trip or journey. For safety the links themselves are not stored, only a check value, so they cannot be shown again.'),
      ...(xs.length ? xs.map((x) => {
        const dead = x.revoked || (x.expires && x.expires < now);
        const how = x.mode === 'stats' ? 'numbers only' : x.mode === 'trimmed' ? 'route, ends trimmed' : 'whole route';
        return h('div', { class: 'row' }, h('div', { class: 'grow' }, h('b', {}, x.title || (x.kind === 'day' ? 'A day' : x.kind === 'journey' ? 'A journey' : 'A trip')), ' ', dead ? chip('bad', x.revoked ? 'turned off' : 'expired') : chip('ok', 'on'),
          h('div', { class: 'muted' }, `${how} · made ${when(x.created)} · ${x.expires ? `until ${when(x.expires)}` : 'no expiry'} · ${x.views} view${x.views === 1 ? '' : 's'}`)),
          dead ? null : h('button', { class: 'sec', onclick: async () => { await api(`/api/shares/${x.id}`, { method: 'DELETE' }); load(); } }, 'Turn off'));
      }) : [h('p', { class: 'muted' }, 'No links yet. Open a trip or journey and use Share.')])]);
  }
  await load();
  return box;
}
