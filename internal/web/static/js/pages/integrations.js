import { h, chip, ago } from '../ui.js';
import { api } from '../api.js';
import { refreshAttention } from '../attention.js';

const STATE = { on: ['ok', 'On'], off: ['work', 'Off'], needs: ['bad', 'Set up'], info: ['work', 'Own page'] };

// A setting's field: a drop-down for a list of choices, a text box otherwise (a key is a password box that keeps what is saved).
function field(s) {
  const el = s.options
    ? h('select', { id: `i-${s.key}`, disabled: s.locked }, ...s.options.map((o) => h('option', { value: o.value }, o.label)))
    : h('input', { id: `i-${s.key}`, type: s.secret ? 'password' : 'text', autocomplete: 'off', autocapitalize: 'none', spellcheck: 'false', disabled: s.locked,
      value: s.secret ? '' : s.value, placeholder: s.secret ? (s.isSet ? '•••••••• saved (leave blank to keep)' : '') : s.placeholder });
  if (s.options) el.value = s.value || '';
  return el;
}

// Every outside service CarTrak talks to, one card each: what it is for, what it sends, whether it is on, and its settings to change
// right there, with a button to check it answers.
export default async function integrations(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Integrations')),
    h('p', { class: 'muted' }, 'The outside services CarTrak uses, all in one place. Keys stay on your server and are never shown again. A value set in an environment variable shows as locked.'), box);

  async function load() {
    const [d, st] = await Promise.all([api('/api/integrations'), api('/api/admin/settings')]);
    const byKey = Object.fromEntries(st.settings.map((s) => [s.key, s]));
    box.replaceChildren(...d.items.map((it) => card(it, it.keys ? it.keys.map((k) => byKey[k]).filter(Boolean) : [])));
    refreshAttention();
  }

  function card(it, settings) {
    const msg = h('p', { class: 'muted', role: 'status' });
    const inputs = {};
    const rows = settings.map((s) => {
      inputs[s.key] = field(s);
      return h('div', {}, h('label', { for: `i-${s.key}` }, s.label.replace(' (advanced)', ''), ' ', s.locked ? chip('work', 'locked') : null), inputs[s.key], h('div', { class: 'muted', style: 'margin-top:4px' }, s.help));
    });
    const [kind, label] = STATE[it.state] || STATE.off;
    const test = it.testable ? h('button', { class: 'sec', type: 'button', onclick: async () => {
      msg.textContent = 'Asking…';
      try { const r = await api('/api/integrations/test', { method: 'POST', body: { id: it.id } }); msg.textContent = `${r.ok ? 'Works' : 'Did not work'}: ${r.msg}`; load(); } catch (e) { msg.textContent = e.message; }
    } }, 'Test connection') : null;
    const save = settings.length ? h('button', { type: 'button', onclick: async () => {
      const values = {};
      for (const s of settings) if (!s.locked && !(s.secret && !inputs[s.key].value.trim())) values[s.key] = inputs[s.key].value.trim();
      try { await api('/api/admin/settings', { method: 'PUT', body: { values } }); msg.textContent = 'Saved. It applies right away.'; load(); } catch (e) { msg.textContent = e.message; }
    } }, 'Save') : null;
    const last = it.lastTest ? h('p', { class: 'muted' }, `Last test ${ago(it.lastTest.ts)}: ${it.lastTest.ok ? 'worked' : 'failed'}${it.lastTest.ok ? '' : `. ${it.lastTest.msg}`}`) : null;
    const open = settings.length > 0 && (it.state === 'needs');
    return h('section', { class: 'card' },
      h('div', { class: 'row' }, h('h3', { class: 'grow' }, it.name), chip(kind, label)),
      h('p', { class: 'muted' }, it.purpose), h('p', { class: 'muted' }, h('b', {}, 'Sends: '), it.sends), it.note ? h('p', {}, it.note) : null, last,
      settings.length ? h('details', { open }, h('summary', {}, 'Settings'), ...rows) : null,
      h('p', { class: 'seg' }, save, test, it.link ? h('a', { class: 'btn sec', href: it.link }, it.linkText || 'Open') : null), msg);
  }
  await load();
}
