import { h, chip } from '../ui.js';
import { api } from '../api.js';

const standalone = () => window.matchMedia('(display-mode: standalone)').matches || navigator.standalone === true;

function step(n, done, title, body, action) {
  return h('section', { class: 'card' },
    h('div', { class: 'row', style: 'border:0;padding:0' }, h('h2', { class: 'grow', style: 'margin:0' }, `${n}. ${title}`),
      done ? chip('ok', 'done') : chip('work', 'to do')),
    h('div', { class: 'muted', style: 'margin:6px 0' }, body), action || null);
}

export default async function start(root, _a, session) {
  const ob = await api('/api/onboarding');
  const admin = session.me.role === 'admin';
  const demo = ob.provider === 'demo';

  const steps = [
    step(1, true, 'Create your account', 'Done. You are signed in as an admin. Family members can get their own read-only accounts in step 7.'),
    step(2, ob.remote, 'Set up remote access (before linking Ford)',
      "Ford's sign-in only sends you back to an https address (or http://localhost). A free Cloudflare tunnel gives CarTrak a public https address, so linking Ford finishes by itself and you can use CarTrak away from home. Skip this only if you will link Ford with the http://localhost paste step.",
      admin ? h('a', { class: 'btn', href: '#/remote' }, ob.remote ? 'Remote access settings' : 'Set up remote access') : null),
    step(3, ob.linked, demo ? 'Connect your Ford (demo mode)' : 'Connect your Ford account',
      demo ? 'CarTrak is running with a simulated car. To use your real car, set the provider to fordconnect in your settings (see the setup guide on GitHub).'
        : 'You need your own Ford developer app (CarTrak includes no Ford credentials). Tip: use a separate FordPass account just for CarTrak, with the car shared to it from your main account. Enter your client ID and secret, set the redirect URL to the https address from step 2 (and register that same address on your Ford app), then connect. You sign in on the Ford page; CarTrak never sees your Ford password.',
      !demo && admin && ob.canLink ? (ob.configured ? h('a', { class: 'btn', href: '#/connect' }, ob.linked ? 'Reconnect Ford' : 'Connect Ford') : h('a', { class: 'btn', href: '#/admin' }, 'Enter your Ford settings first')) : null),
    step(4, ob.hasReading, 'Wait for the first reading',
      ob.hasReading ? 'Data is coming in. Open Home to see it.' : 'After connecting, the first reading arrives within a minute or two. The Home page fills in by itself.'),
    step(5, standalone(), 'Install it on your phone',
      'Android (Chrome): menu ⋮ → Install app. iPhone (Safari): Share → Add to Home Screen. It then opens like a normal app.',
      session.installPrompt ? h('button', { onclick: async () => { session.installPrompt.prompt(); session.installPrompt = null; } }, 'Install now') : null),
    step(6, ob.rules > 0, 'Set up work and personal trips',
      'A rule labels trips for you, for example "Mon–Fri 8am–5pm = work". You can always change a single trip by hand, and your choice is never overwritten.',
      admin ? h('a', { class: 'btn', href: '#/rules' }, 'Add a rule') : null),
    step(7, ob.users > 1, 'Add family (optional)',
      'Give family read-only accounts so they can see the dashboard without changing anything.',
      admin ? h('a', { class: 'btn', href: '#/people' }, 'Add people') : null),
    // Optional, to get more from CarTrak.
    step(8, ob.places > 0, 'Name your places (optional)', 'Name Home and the office. Trips then read "Home → Office", and you can get alerts when the car arrives or leaves.',
      admin ? h('a', { class: 'btn', href: '#/places' }, 'Name a place') : null),
    step(9, ob.priceSet, 'Set your electricity price (optional)', 'See what charging costs and your cost per mile. You can add your charger\'s own kWh to measure charging loss.',
      admin ? h('a', { class: 'btn', href: '#/admin' }, 'Open Settings') : null),
    step(10, ob.alerts > 0, 'Turn on alerts (optional)', 'Charging complete, plug in tonight, tires, service due and more, on this phone.',
      h('a', { class: 'btn', href: '#/notifications' }, 'Choose alerts')),
    step(11, ob.maintenance > 0, 'Add a service reminder (optional)', 'Tire rotation, cabin filter and so on, by distance or time.',
      admin ? h('a', { class: 'btn', href: '#/maintenance' }, 'Add a reminder') : null),
    step(12, ob.backedUp, 'Make a backup (optional)', 'Download a copy of your data so a failed disk or a mistake cannot lose it.',
      admin ? h('a', { class: 'btn', href: '#/backup' }, 'Open Backup') : null),
  ];

  root.append(h('h2', {}, 'Getting started'),
    h('p', { class: 'muted' }, 'A few quick steps, then some optional ones to get more from CarTrak. You can come back to this any time from More.'), ...steps,
    admin ? h('p', { class: 'seg' }, h('button', { onclick: async () => { await api('/api/onboarding/dismiss', { method: 'POST' }); location.hash = '#/'; } }, 'Finish and go Home')) : h('p', {}, h('a', { class: 'btn', href: '#/' }, 'Go Home')));
}
