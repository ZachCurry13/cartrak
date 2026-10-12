import { h } from './ui.js';

// Ford sends the browser here after sign-in. The code can arrive in the query
// (?code=) or after a # (the fragment), which servers never receive, so this page
// reads the whole address in the browser and hands it to CarTrak.
const out = document.getElementById('out');
const show = (...kids) => out.replaceChildren(...kids.filter(Boolean));

const q = new URLSearchParams(location.search);
const f = new URLSearchParams(location.hash.replace(/^#/, ''));
const names = [...new Set([...q.keys(), ...f.keys()])];
const has = (k) => q.has(k) || f.has(k);

async function finish() {
  if (!has('code') && !has('error')) {
    show(h('div', { class: 'banner' }, h('b', {}, 'Ford sent you back without a code.'),
      h('p', {}, names.length ? `Ford sent these fields: ${names.join(', ')}.` : 'Ford sent no fields at all.'),
      h('p', { class: 'muted' }, 'Start again from More → Connect Ford. If it keeps happening, check that the redirect URL in CarTrak matches the one registered on your Ford app exactly.'),
      h('a', { class: 'btn', href: '/#/connect' }, 'Back to Connect Ford')));
    return;
  }
  const full = location.href;
  const r = await fetch('/api/ford/complete', {
    method: 'POST', credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'cartrak' },
    body: JSON.stringify({ url: full }),
  });
  const body = await r.json().catch(() => ({}));
  if (r.status !== 401) history.replaceState(null, '', location.pathname); // drop the one-time code from the address bar and history
  if (r.ok) {
    show(h('div', { class: 'card' }, h('h2', {}, 'Ford is connected'), h('p', {}, 'The first reading arrives within a minute or two.'),
      h('a', { class: 'btn', href: '/#/' }, 'Go to CarTrak')));
    setTimeout(() => location.replace('/#/'), 2500);
  } else if (r.status === 401) {
    show(h('div', { class: 'banner' }, h('b', {}, 'Sign in to CarTrak first. '),
      'This browser is not signed in to CarTrak. Sign in as admin, then either press Connect Ford again, or paste the address of this page into More → Connect Ford, step 2 (copy it from the address bar first).',
      h('p', {}, h('a', { class: 'btn', href: '/' }, 'Open CarTrak'))));
  } else {
    show(h('div', { class: 'banner' }, h('b', {}, 'Could not finish connecting. '), body.error || `HTTP ${r.status}`,
      h('p', {}, h('a', { class: 'btn', href: '/#/connect' }, 'Back to Connect Ford'))));
  }
}
finish().catch((e) => show(h('div', { class: 'banner' }, `Something went wrong: ${e.message}`)));
