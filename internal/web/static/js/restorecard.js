import { h, when } from './ui.js';
import { api } from './api.js';

const size = (b) => (b >= 1e9 ? `${(b / 1e9).toFixed(1)} GB` : `${Math.max(1, Math.round(b / 1e6))} MB`);

// Restore a backup from the app: upload (or pick a file you copied into the restore folder), check what is in it,
// then confirm. The current data is kept beside it.
export function restoreCard() {
  const box = h('section', { class: 'card' });
  const msg = h('p', { class: 'muted', role: 'status' });

  async function load() {
    const d = await api('/api/restore');
    if (d.pending) return confirmView(d.pending);
    pickView(d.files || []);
  }

  function pickView(files) {
    const input = h('input', { type: 'file', id: 'rf', accept: '.db,application/octet-stream', 'aria-label': 'Backup file' });
    const upload = async () => {
      const f = input.files && input.files[0];
      if (!f) { msg.textContent = 'Choose a backup file first.'; return; }
      msg.textContent = `Uploading ${size(f.size)} and checking it. A large file can take a while…`;
      try {
        const r = await fetch('/api/restore/upload', { method: 'POST', credentials: 'same-origin', headers: { 'X-Requested-With': 'cartrak', 'Content-Type': 'application/octet-stream' }, body: f });
        const j = await r.json().catch(() => ({}));
        if (!r.ok) throw new Error(j.error || `HTTP ${r.status}`);
        msg.textContent = '';
        confirmView(j);
      } catch (e) { msg.textContent = e.message; }
    };
    const use = (name) => async () => {
      msg.textContent = 'Checking…';
      try { msg.textContent = ''; confirmView(await api('/api/restore/use', { method: 'POST', body: { name } })); } catch (e) { msg.textContent = e.message; }
    };
    box.replaceChildren(...[h('h2', {}, 'Restore a backup'),
      h('p', { class: 'muted' }, 'Choose a backup you downloaded here. CarTrak checks it first and shows what is in it before anything changes.'),
      h('label', { for: 'rf' }, 'Backup file'), input,
      h('p', {}, h('button', { onclick: upload }, 'Upload and check')),
      files.length ? h('div', {}, h('p', { class: 'muted' }, 'Or use a file you copied into the restore folder in CarTrak\'s data folder (best for very large backups):'),
        ...files.map((n) => h('p', {}, h('button', { class: 'sec', onclick: use(n) }, `Check ${n}`)))) : null,
      msg,
      h('p', { class: 'muted' }, 'Uploads through a Cloudflare tunnel are limited to 100 MB on the free plan. For a larger backup, restore from your home network, or copy the file into the restore folder.')].filter(Boolean));
  }

  function confirmView(b) {
    const word = h('input', { id: 'rw', placeholder: 'Type RESTORE', autocomplete: 'off', 'aria-label': 'Type RESTORE to confirm' });
    const go = h('button', { class: 'danger', disabled: true, onclick: apply }, 'Restore and restart');
    word.addEventListener('input', () => { go.disabled = word.value.trim() !== 'RESTORE'; });
    async function apply() {
      go.disabled = true;
      msg.textContent = 'Restarting CarTrak…';
      try { await api('/api/restore/apply', { method: 'POST', body: { confirm: 'RESTORE' } }); } catch (e) { msg.textContent = e.message; go.disabled = false; return; }
      box.replaceChildren(h('h2', {}, 'Restoring'), h('p', {}, 'CarTrak is restarting with your backup. This page reloads when it is back (usually under a minute). If it does not come back, press Start for the app in TrueNAS.'));
      let down = false;
      const t = setInterval(async () => {
        try {
          const r = await fetch('/healthz', { cache: 'no-store' });
          if (r.ok && down) { clearInterval(t); location.href = '/'; }
        } catch (e) { down = true; }
      }, 2000);
    }
    box.replaceChildren(...[h('h2', {}, 'Ready to restore'),
      h('p', {}, `This backup is ${size(b.bytes)} with ${b.trips} trips, ${b.charges} charging sessions and ${b.readings} readings${b.firstReading ? `, from ${when(b.firstReading)} to ${when(b.lastReading)}` : ''}.`),
      h('p', {}, `Accounts in it: ${b.users.join(', ')}. After restoring, everyone signs in with these accounts and their passwords from that time.`),
      h('p', { class: 'banner' }, 'Restoring replaces everything now in CarTrak and signs everyone out. Your current data is not deleted: it is kept in the data folder as cartrak.db.before-restore-<time>, and you can delete it later.'),
      h('label', { for: 'rw' }, 'To confirm, type RESTORE'), word,
      h('p', { class: 'seg' }, go, h('button', { class: 'sec', onclick: async () => { await api('/api/restore', { method: 'DELETE' }); load(); } }, 'Cancel')), msg].filter(Boolean));
  }

  box.append(h('h2', {}, 'Restore a backup'), msg);
  load().catch((e) => { msg.textContent = e.message; });
  return box;
}
