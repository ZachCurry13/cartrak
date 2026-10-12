import { h } from '../ui.js';
import { api } from '../api.js';

export function loginPage(needsSetup, done) {
  const err = h('div', { class: 'muted', role: 'alert' });
  const name = h('input', { id: 'u', autocomplete: 'username', required: true, autocapitalize: 'none' });
  const pass = h('input', { id: 'p', type: 'password', autocomplete: needsSetup ? 'new-password' : 'current-password', required: true });
  const form = h('form', { class: 'card login', onsubmit: async (ev) => {
    ev.preventDefault();
    err.textContent = '';
    try {
      await api(needsSetup ? '/api/setup' : '/api/login', { method: 'POST', body: { name: name.value, password: pass.value }, noAuthRedirect: true });
      done();
    } catch (e) { err.textContent = e.message; }
  } },
  h('h2', {}, needsSetup ? 'Create the admin account' : 'Sign in'),
  needsSetup && h('p', { class: 'muted' }, 'First run: choose a name and a password of 10+ characters.'),
  h('label', { for: 'u' }, 'Name'), name,
  h('label', { for: 'p' }, 'Password'), pass,
  h('p', {}, h('button', { type: 'submit', style: 'width:100%' }, needsSetup ? 'Create account' : 'Sign in')),
  err);
  return form;
}
