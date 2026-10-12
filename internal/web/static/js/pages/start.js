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
    step(1, true, 'Create your account', 'Done. You are signed in as an admin. Family members can get their own read-only accounts in step 6.'),
    step(2, ob.linked, demo ? 'Connect your Ford (demo mode)' : 'Connect your Ford account',
      demo ? 'CarTrak is running with a simulated car. To use your real car, set the provider to fordconnect in your settings (see the setup guide on GitHub).'
        : 'Tip: use a separate FordPass account just for CarTrak, with the car shared to it from your main account, so your main account is never at risk. You sign in on Ford\'s own page and approve the vehicle. CarTrak never sees your Ford password and only keeps a token that lets it read your car\'s data.',
      !demo && admin && ob.canLink ? h('a', { class: 'btn', href: '/ford/connect' }, ob.linked ? 'Reconnect Ford' : 'Connect Ford') : null),
    step(3, ob.hasReading, 'Wait for the first reading',
      ob.hasReading ? 'Data is coming in. Open Home to see it.' : 'After connecting, the first reading arrives within a minute or two. The Home page fills in by itself.'),
    step(4, standalone(), 'Install it on your phone',
      'Android (Chrome): menu ⋮ → Install app. iPhone (Safari): Share → Add to Home Screen. It then opens like a normal app.',
      session.installPrompt ? h('button', { onclick: async () => { session.installPrompt.prompt(); session.installPrompt = null; } }, 'Install now') : null),
    step(5, ob.rules > 0, 'Set up work and personal trips',
      'A rule labels trips for you, for example "Mon–Fri 8am–5pm = work". You can always change a single trip by hand, and your choice is never overwritten.',
      admin ? h('a', { class: 'btn', href: '#/rules' }, 'Add a rule') : null),
    step(6, ob.users > 1, 'Add family (optional)',
      'Give family read-only accounts so they can see the dashboard without changing anything.',
      admin ? h('a', { class: 'btn', href: '#/settings' }, 'Add people') : null),
  ];

  root.append(h('h2', {}, 'Getting started'),
    h('p', { class: 'muted' }, 'Six quick steps. You can come back to this any time from More.'), ...steps,
    admin ? h('p', { class: 'seg' }, h('button', { onclick: async () => { await api('/api/onboarding/dismiss', { method: 'POST' }); location.hash = '#/'; } }, 'Finish and go Home')) : h('p', {}, h('a', { class: 'btn', href: '#/' }, 'Go Home')));
}
