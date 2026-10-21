import { h, chip, ago, when } from '../ui.js';
import { api } from '../api.js';
import { unsupported, current, enable, disable } from '../push.js';
import { loadPrefs, savePrefs } from '../myprefs.js';

const guessLabel = () => (/android/i.test(navigator.userAgent) ? 'Android phone' : /iphone|ipad/i.test(navigator.userAgent) ? 'iPhone' : /windows/i.test(navigator.userAgent) ? 'Windows computer' : /mac/i.test(navigator.userAgent) ? 'Mac' : 'This device');

export default async function notifications(root, _a, session) {
  let cfg = await api('/api/push/config').catch(() => null);
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Notifications')));
  if (!cfg) return root.append(h('p', { class: 'muted' }, 'Notifications are not available on this server.'));

  const why = unsupported();
  let here = why ? null : await current();
  let mine = () => cfg.devices.find((d) => d.endpoint === here);
  // The phone still has its push address but the server lost it (a reinstall, or a device removed elsewhere): register it
  // again with the alerts it had, so choices made here are saved.
  let rejoined = false;
  if (here && !mine() && typeof Notification !== 'undefined' && Notification.permission === 'granted') {
    try { cfg = await enable(cfg.publicKey, guessLabel(), cfg.defaults, ''); here = await current(); rejoined = true; } catch (e) { /* shown as off below */ }
  }
  const chosen = new Set((mine() || { events: cfg.defaults }).events);
  const q0 = ((mine() || {}).quiet || '').split('-');
  const label = h('input', { id: 'nl', value: (mine() || {}).label || guessLabel(), maxlength: 40 });
  const qOn = h('input', { id: 'qon', type: 'checkbox', checked: !!q0[0] });
  const qFrom = h('input', { id: 'qf', type: 'time', value: q0[0] || '22:00' });
  const qTo = h('input', { id: 'qt', type: 'time', value: q0[1] || '07:00' });
  const prefs = await loadPrefs();
  const holidayOn = h('input', { id: 'hol', type: 'checkbox', checked: !prefs.holidayOff, onchange: () => dirty() });
  const msg = h('p', { class: 'muted', role: 'status' });
  const box = h('div');
  const quiet = () => (qOn.checked ? `${qFrom.value}-${qTo.value}` : '');
  for (const el of [qOn, qFrom, qTo]) el.addEventListener('change', () => dirty());
  const say = (t) => { msg.textContent = t; };

  async function run(f) {
    try { say(''); await f(); } catch (e) { say(e.message); }
  }
  // One save for the alerts and quiet hours together. A device the server lost is registered again with these choices; then
  // the page shows what the server really kept.
  const saveBar = h('div', { class: 'savebar', hidden: true });
  const dirty = () => { saveBar.hidden = false; say(''); };
  async function save() {
    const touches = holidayOn.checked === !!prefs.holidayOff;
    if (touches) Object.assign(prefs, await savePrefs({ holidayOff: !holidayOn.checked })); // follows the person, not the device
    if (!mine() && !(here && typeof Notification !== 'undefined' && Notification.permission === 'granted')) {
      say(`${touches ? 'Holiday touches saved ✓ ' : ''}Turn on alerts on this device (above) to save which alerts it gets.`);
      return;
    }
    try {
      cfg = mine() ? await api('/api/push/prefs', { method: 'POST', body: { endpoint: here, events: [...chosen], quiet: quiet() } }) : null;
    } catch (e) {
      if (!/not registered/.test(e.message)) throw e;
      cfg = null;
    }
    if (!cfg) { cfg = await enable((await api('/api/push/config')).publicKey, label.value.trim() || guessLabel(), [...chosen], quiet()); here = await current(); }
    const kept = mine();
    saveBar.hidden = true;
    draw();
    say(kept ? `Saved ✓ ${kept.events.length} alert${kept.events.length === 1 ? '' : 's'} on this device${kept.quiet ? `, quiet ${kept.quiet}` : ''}.` : 'Could not save: this device is not registered. Turn alerts on above.');
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
          const box = h('input', { type: 'checkbox', id: `k-${k.id}`, checked: chosen.has(k.id), onchange: (e) => { if (e.target.checked) chosen.add(k.id); else chosen.delete(k.id); dirty(); } });
          return h('div', { class: 'hrow' }, h('label', { for: `k-${k.id}`, class: 'hl' }, box, ' ', k.label), h('span', { class: 'hn muted' }, k.help));
        }),
        h('h2', {}, 'Quiet hours'),
        h('p', {}, h('label', { for: 'qon' }, qOn, ' No alerts between')),
        h('div', { class: 'cols2' }, h('div', {}, h('label', { for: 'qf' }, 'From'), qFrom), h('div', {}, h('label', { for: 'qt' }, 'To'), qTo)),
        h('p', { class: 'muted' }, 'Quiet hours use the car time zone from Settings.'),
        h('h2', {}, 'Holiday touches'),
        h('p', {}, h('label', { for: 'hol' }, holidayOn, ' 🎃 Dress up friendly alerts for holidays and my birthday')),
        h('p', { class: 'muted' }, 'Around a holiday, alerts like Trip ended and Charging complete start with its emoji and wish you a happy holiday on the day. On your birthday (set it under Display) you also get a happy birthday in the morning. Warnings always stay plain. This follows you to every device.'),
        h('p', { class: 'seg' }, h('button', { type: 'button', class: 'on', onclick: () => run(save) }, 'Save alert choices')),
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

  // Admins can see what CarTrak decided to send and who it reached, to find out why an alert did or did not arrive.
  const log = h('div');
  const logLine = (x) => {
    const who = x.wanted ? `Sent to ${x.sent} of ${x.wanted} device${x.wanted === 1 ? '' : 's'}.` : x.devices ? 'No device has this alert turned on.' : 'No devices are registered.';
    return h('div', { class: 'hrow' }, h('span', { class: 'hl' }, x.title), h('span', { class: 'hv muted' }, when(x.ts)),
      h('span', { class: 'hn muted' }, `${x.body} ${who}${x.quiet ? ` ${x.quiet} in quiet hours.` : ''}`));
  };
  if (session && session.me.role === 'admin') {
    api('/api/alerts/log').then((xs) => {
      log.replaceChildren(h('section', { class: 'card' }, h('h2', {}, 'Recent alerts'),
        h('p', { class: 'muted' }, 'Every alert CarTrak decided to send, and who it reached. If an alert you expected is missing here, CarTrak never decided to send it.'),
        ...(xs.length ? xs.map(logLine) : [h('p', { class: 'muted' }, 'None yet.')])));
    }).catch(() => { /* not available */ });
  }
  root.append(h('p', { class: 'muted' }, 'Get a notification on this phone or computer when something happens. Alerts say things like "Charging complete" and never include a location. They are encrypted so only your device can read them, but the push service of the phone maker (Google or Apple) can see that one was sent.'), box);
  root.append(log);
  saveBar.append(h('span', { class: 'grow' }, 'Unsaved changes'), h('button', { type: 'button', class: 'on', onclick: () => run(save) }, 'Save'));
  root.append(saveBar);
  draw();
  if (rejoined) say('This device was registered again, so its alerts are saved here. Check the ones you want.');
}
