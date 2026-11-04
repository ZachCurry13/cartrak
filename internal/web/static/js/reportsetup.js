import { h } from './ui.js';
import { api } from './api.js';

const NEW_REPO = 'https://github.com/new?name=cartrak-reports&visibility=private&description=Bug+reports+from+CarTrak';
const NEW_TOKEN = 'https://github.com/settings/personal-access-tokens/new';

// The one-time setup of bug reports, inside the report sheet: make a private repository, make a token for it, paste both here.
// CarTrak checks with GitHub that the repository exists, is private and can be written to before keeping anything.
export function reportSetup(close, done) {
  const repo = h('input', { placeholder: 'your-name/cartrak-reports', maxlength: 200, autocomplete: 'off', 'aria-label': 'Repository', style: 'width:100%;box-sizing:border-box' });
  const token = h('input', { type: 'password', placeholder: 'Paste the token', maxlength: 255, autocomplete: 'off', 'aria-label': 'Token', style: 'width:100%;box-sizing:border-box' });
  const msg = h('p', { class: 'muted', role: 'status' });
  const go = h('button', { type: 'button', onclick: async () => {
    go.disabled = true; msg.textContent = 'Checking with GitHub…';
    try { await api('/api/bug-report/setup', { method: 'POST', body: { repo: repo.value, token: token.value } }); done(); } catch (e) { msg.textContent = e.message; go.disabled = false; }
  } }, 'Check and save');
  const link = (href, text) => h('a', { href, target: '_blank', rel: 'noopener' }, text);
  return [
    h('p', { class: 'muted' }, 'Reports hold screenshots and your car\'s positions, so they go to a private GitHub repository of your own, never the public CarTrak one. Three quick steps, once:'),
    h('ol', { style: 'padding-left:1.2em;margin:0 0 var(--sp-3)' },
      h('li', {}, link(NEW_REPO, 'Create the private repository'), ' (the name and Private are filled in; tick Add a README, then tap Create).'),
      h('li', {}, link(NEW_TOKEN, 'Create a token'), ': name it CarTrak, choose Only select repositories and pick that repository, then under Repository permissions set ', h('b', {}, 'Contents'), ' and ', h('b', {}, 'Issues'), ' to Read and write. Copy the token.'),
      h('li', {}, 'Paste the repository name and the token here.')),
    repo, h('div', { style: 'height:var(--sp-2)' }), token,
    h('p', {}, go, ' ', h('button', { class: 'text', type: 'button', onclick: close }, 'Close')), msg,
    h('p', { class: 'muted' }, 'The token is kept on your server only and never shown again.'),
  ];
}
