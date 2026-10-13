import { h, chip, ago } from '../ui.js';
import { api } from '../api.js';
import { unsupported, current, enable, disable } from '../push.js';

const guessLabel = () => (/android/i.test(navigator.userAgent) ? 'Android phone' : /iphone|ipad/i.test(navigator.userAgent) ? 'iPhone' : /windows/i.test(navigator.userAgent) ? 'Windows computer' : /mac/i.test(navigator.userAgent) ? 'Mac' : 'This device');

export default async function notifications(root) {
  let cfg = await api('/api/push/config').catch(() => null);
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Notifications')));
  if (!cfg) return root.append(h('p', { class: 'muted' }, 'Notifications are not available on this server.'));

  const why = unsupported();
  let here = why ? null : await current();
  let mine = () => cfg.devices.find((d) => d.endpoint === here);
  const chosen = new Set((mine() || { events: cfg.defaults }).events);
  const q0 = ((mine() || {}).quiet || '').split('-');
  const label = h('input', { id: 'nl', value: (mine() || {}).label || guessLabel(), maxlength: 40 });
  const qOn = h('input', { id: 'qon', type: 'checkbox', checked: !!q0[0] });
  const qFrom = h('input', { id: 'qf', type: 'time', value: q0[0] || '22:00' });
  const qTo = h('input', { id: 'qt', type: 'time', value: q0[1] || '07:00' });
  const msg = h('p', { class: 'muted', role: 'status' });
  const box = h('div');
  const quiet = () => (qOn.checked ? `${qFrom.value}-${qTo.value}` : '');
  const say = (t) => { msg.textContent = t; };

  async function run(f) {
    try { say(''); await f(); } catch (e) { say(e.message); }
  }
  async function save() {
    if (!mine()) return;
    cfg = await api('/api/push/prefs', { method: 'POST', body: { endpoint: here, events: [...chosen], quiet: quiet() } });
    say('Saved.');
  }
  const toggle = () => run(async () => {
    if (mine()) { cfg = await disable(); here = null; } else { cfg = await enable(cfg.publicKey, label.value.trim(), [...chosen], quiet()); here = await current(); }
    draw();
  });

  function draw() {
    const on = !!mine();
    box.replaceChildren(...[
      why ? h('div', { class: 'banner' }, why) : h('section', { class: 'card' }, h('h2', {}, 'This device'),
        h('p', {}, on ? chip('ok', 'Alerts on') : chip('work', 'Alerts off')),
        on ? null : h('div', {}, h('label', { for: 'nl' }, 'Name for this device'), label),
        h('p', { class: 'seg' }, h('button', { onclick: toggle }, on ? 'Turn off alerts here' : 'Turn on alerts here'),
          on ? h('button', { class: 'sec', onclick: () => run(async () => { await api('/api/push/test', { method: 'POST', body: { endpoint: here } }); say('Test sent. It should arrive in a moment.'); }) }, 'Send a test alert') : null)),
      h('section', { class: 'card' }, h('h2', {}, 'Which alerts'),
        ...cfg.kinds.map((k) => {
          const box = h('input', { type: 'checkbox', id: `k-${k.id}`, checked: chosen.has(k.id), onchange: (e) => { if (e.target.checked) chosen.add(k.id); else chosen.delete(k.id); run(save); } });
          return h('div', { class: 'hrow' }, h('label', { for: `k-${k.id}`, class: 'hl' }, box, ' ', k.label), h('span', { class: 'hn muted' }, k.help));
        }),
        h('h2', {}, 'Quiet hours'),
        h('p', {}, h('label', { for: 'qon' }, qOn, ' No alerts between')),
        h('div', { class: 'cols2' }, h('div', {}, h('label', { for: 'qf' }, 'From'), qFrom), h('div', {}, h('label', { for: 'qt' }, 'To'), qTo)),
        h('p', {}, h('button', { class: 'sec', onclick: () => run(save) }, 'Save quiet hours'), ' ', h('span', { class: 'muted' }, 'Uses the car time zone from Settings.')),
        msg),
      cfg.devices.length ? h('section', { class: 'card' }, h('h2', {}, 'Your devices'),
        ...cfg.devices.map((d) => h('div', { class: 'hrow' }, h('span', { class: 'hl' }, d.label, d.endpoint === here ? ' (this one)' : ''),
          h('span', { class: 'hv' }, d.lastErr ? chip('bad', 'Failed') : d.lastOk ? chip('ok', `Last sent ${ago(d.lastOk)}`) : chip('work', 'Nothing sent yet'),
            ' ', h('button', { class: 'danger', onclick: () => run(async () => {
              if (!confirm(`Stop alerts on ${d.label}?`)) return;
              cfg = d.endpoint === here ? await disable() : await api('/api/push/unsubscribe', { method: 'POST', body: { endpoint: d.endpoint } });
              if (d.endpoint === here) here = null;
              draw();
            }) }, 'Remove')),
          d.lastErr ? h("span", { class: "hn muted" }, d.lastErr) : null))) : null].filter(Boolean));
  }

  root.append(h('p', { class: 'muted' }, 'Get a notification on this phone or computer when something happens. Alerts say things like "Charging complete" and never include a location. They are encrypted so only your device can read them, but the push service of the phone maker (Google or Apple) can see that one was sent.'), box);
  draw();
}
