import { h, chip, ago } from './ui.js';
import { api } from './api.js';

// "About" card on the More page: version, update status, changelog, help.
export function aboutCard(session, admin) {
  const a = session.about;
  const u = a.update;
  const check = h('input', { type: 'checkbox', id: 'uc', disabled: !admin });
  check.checked = a.updateCheck;
  check.addEventListener('change', async () => {
    await api('/api/settings/update-check', { method: 'POST', body: { enabled: check.checked } });
    a.updateCheck = check.checked;
  });
  const status = !a.updateCheck ? 'Update checks are off.'
    : u.error ? `Could not check for updates (${u.error}).`
      : u.checkedAt ? `Checked ${ago(u.checkedAt)}.` : 'Not checked yet.';
  return h('section', { class: 'card' },
    h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow' }, `CarTrak v${a.version}`),
      u.available ? chip('work', `v${u.latest} available`) : a.updateCheck && u.checkedAt && !u.error ? chip('ok', 'up to date') : null),
    h('p', { class: 'muted' }, status),
    h('p', {}, h('a', { href: '#/changelog' }, "What's new"), ' · ', h('a', { href: '#/start' }, 'Getting started'), ' · ',
      h('a', { href: a.releasesUrl, target: '_blank', rel: 'noopener' }, 'Releases on GitHub')),
    ...(a.components || []).map((c) => h('div', { class: 'hrow' }, h('span', { class: 'hl' }, c.name), h('span', { class: 'hv' }, c.version || 'unknown', ' ', c.update ? chip('work', `${c.latest} available`) : null), c.note ? h('span', { class: 'hn muted' }, c.note) : null)),
    h('label', { for: 'uc', style: 'display:flex;gap:8px;align-items:center;color:var(--text)' }, check,
      'Check GitHub once a day for new versions (sends no vehicle or personal data)'));
}
