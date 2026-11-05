import { h, chip, ago } from './ui.js';
import { api } from './api.js';

// The lists CarTrak downloads and keeps (the chargers, the gas prices) and the version of CarTrak itself. CarTrak looks once a day on
// its own; nothing changes until you tap Update. A new version of CarTrak cannot install itself: it runs in a container that is stopped
// and started on the server.
export function dataUpdatesCard(onChange) {
  const box = h('section', { class: 'card' });
  const msg = h('p', { class: 'muted', role: 'status' });
  async function draw(view) {
    const d = view || await api('/api/data-updates').catch(() => null);
    if (!d) return box.replaceChildren();
    const run = async (it, btn) => {
      btn.disabled = true; msg.textContent = `Updating ${it.name}…`;
      try { draw(await api(`/api/data-updates/${it.id}/update`, { method: 'POST' })); msg.textContent = `${it.name} is up to date.`; onChange(); } catch (e) { msg.textContent = e.message; btn.disabled = false; }
    };
    const app = d.app || {};
    const row = (it) => {
      const btn = h('button', { class: it.available ? '' : 'sec', type: 'button' }, it.updatedAt ? (it.available ? 'Update' : 'Update now') : 'Download');
      btn.onclick = () => run(it, btn);
      return h('div', { class: 'row', style: 'flex-wrap:wrap;gap:8px' },
        h('div', { class: 'grow', style: 'flex:1 1 12em;min-width:0' }, h('b', {}, it.name), ' ', it.available ? chip('work', 'Update available') : it.updatedAt ? chip('ok', 'Up to date') : null,
          h('div', { class: 'muted' }, it.updatedAt ? `${it.source} · updated ${ago(it.updatedAt)}${it.count ? ` · ${it.count.toLocaleString()} entries` : ''}` : `${it.source} · not downloaded yet`), it.note ? h('div', { class: 'muted' }, it.note) : null), btn);
    };
    box.replaceChildren(h('h2', {}, 'Data updates'), h('p', { class: 'muted' }, 'CarTrak checks once a day for newer lists. Nothing changes until you tap Update.'),
      h('div', { class: 'row', style: 'flex-wrap:wrap;gap:8px' }, h('div', { class: 'grow', style: 'flex:1 1 12em;min-width:0' }, h('b', {}, `CarTrak ${app.current || ''}`), ' ',
        !app.enabled ? chip('work', 'Checks are off') : app.available ? chip('work', `${app.latest} is out`) : chip('ok', 'Latest'),
        h('div', { class: 'muted' }, !app.enabled ? 'Update checks are switched off (More, About).' : app.available ? 'It cannot install itself: stop and start the app on your server to update.' : 'You have the newest version.')),
        app.available && app.url ? h('a', { class: 'btn', href: app.url, target: '_blank', rel: 'noopener' }, "See what's new") : null),
      ...d.items.map(row),
      h('p', { class: 'seg' }, h('button', { class: 'sec', type: 'button', onclick: async (e) => { e.currentTarget.disabled = true; msg.textContent = 'Checking…'; try { draw(await api('/api/data-updates/check', { method: 'POST' })); msg.textContent = ''; onChange(); } catch (x) { msg.textContent = x.message; } } }, 'Check now'),
        h('span', { class: 'muted' }, d.checkedAt ? `Last checked ${ago(d.checkedAt)}` : 'Not checked yet')), msg);
  }
  draw();
  return box;
}
