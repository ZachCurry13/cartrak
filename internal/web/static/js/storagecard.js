import { h } from './ui.js';
import { api } from './api.js';

const size = (b) => (b >= 1e9 ? `${(b / 1e9).toFixed(1)} GB` : `${Math.max(1, Math.round(b / 1e6))} MB`);

// Optional: free up space by removing the full raw detail of old readings. The database does not shrink on its
// own, but the freed space is reused, so it stops growing for a while.
export function storageCard() {
  const box = h('section', { class: 'card' });
  const msg = h('p', { class: 'muted', role: 'status' });
  let days = 90;
  let timer = null;

  async function load() {
    const d = await api(`/api/storage?days=${days}`);
    if (d.trim.running) { progress(d); return; }
    if (timer) { clearInterval(timer); timer = null; }
    const sel = h('select', { id: 'td', 'aria-label': 'Keep the full detail for' }, ...[30, 90, 180, 365].map((n) => h('option', { value: n, selected: n === days }, `Keep the last ${n} days`)));
    sel.addEventListener('change', () => { days = Number(sel.value); load(); });
    const word = h('input', { id: 'tw', placeholder: 'Type TRIM', autocomplete: 'off', 'aria-label': 'Type TRIM to confirm' });
    const go = h('button', { class: 'danger', disabled: true, onclick: start }, 'Free up space');
    word.addEventListener('input', () => { go.disabled = word.value.trim() !== 'TRIM'; });
    async function start() {
      go.disabled = true;
      try { await api('/api/storage/trim', { method: 'POST', body: { days, confirm: 'TRIM' } }); load(); } catch (e) { msg.textContent = e.message; go.disabled = false; }
    }
    box.replaceChildren(h('h2', {}, 'Free up space (optional)'),
      h('p', { class: 'muted' }, `The database is ${size(d.dbBytes)}. Most of it is the full raw copy of every reading from Ford. You can remove that detail for older readings and keep everything CarTrak uses: trips, charging, places, charts, and each reading's main numbers. Car data for those older readings will show less.`),
      sel,
      h('p', {}, d.samples ? `${d.samples} readings and ${d.snapshots} raw copies are older than that.` : 'Nothing is older than that yet.'),
      d.samples ? h('div', {}, h('p', { class: 'banner' }, 'This cannot be undone, except by restoring a backup. Make a full backup first if you may want the old detail. The file does not get smaller by itself; the freed space is reused for new readings, so it stops growing for a while.'),
        h('label', { for: 'tw' }, 'To confirm, type TRIM'), word, h('p', {}, go)) : null, msg, d.trim.message ? h('p', { class: 'muted' }, d.trim.message) : null);
  }

  function progress(d) {
    box.replaceChildren(h('h2', {}, 'Free up space (optional)'), h('p', {}, `Working: ${d.trim.done} of about ${d.trim.total} done. You can leave this page; it keeps going.`));
    if (!timer) timer = setInterval(() => load().then(() => {}).catch(() => {}), 3000);
  }

  box.append(h('h2', {}, 'Free up space (optional)'), msg);
  load().catch((e) => { msg.textContent = e.message; });
  return box;
}
