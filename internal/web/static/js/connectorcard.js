import { h, chip, ago } from './ui.js';
import { api } from './api.js';

// The Cloudflare connector's version, and a button to install a newer one without breaking remote access.
export function connectorCard() {
  const box = h('section', { class: 'card' });
  const msg = h('p', { class: 'muted', role: 'status' });
  let timer = null;

  async function load() {
    const i = await api('/api/connector');
    paint(i);
    const working = i.update && i.update.state === 'working';
    if (working && !timer) timer = setInterval(load, 2500);
    if (!working && timer) { clearInterval(timer); timer = null; }
  }

  async function check() {
    msg.textContent = 'Checking GitHub…';
    try { paint(await api('/api/connector/check', { method: 'POST' })); msg.textContent = ''; } catch (e) { msg.textContent = e.message; }
  }

  async function update() {
    msg.textContent = 'Starting…';
    try { await api('/api/connector/update', { method: 'POST' }); load(); } catch (e) { msg.textContent = e.message; }
  }

  function paint(i) {
    const u = i.update || {};
    const working = u.state === 'working';
    box.replaceChildren(...[h('h2', {}, 'Cloudflare connector'),
      h('div', { class: 'row', style: 'border:0;padding:0' }, h('b', { class: 'grow' }, i.current ? `Version ${i.current}` : 'Version unknown'),
        i.available ? chip('work', `${i.latest} available`) : i.current && i.latest && !i.error ? chip('ok', 'up to date') : null),
      h('p', { class: 'muted' }, i.error ? `Could not check (${i.error}).` : i.checkedAt ? `Checked ${ago(i.checkedAt)}.` : 'Not checked yet.'),
      i.failed && u.state !== 'failed' ? h('p', { class: 'banner' }, `Version ${i.failed} did not connect to Cloudflare, so the previous one was put back. It will not be offered again.`) : null,
      u.message && u.state !== 'idle' ? h('p', { class: u.state === 'failed' ? 'banner' : 'muted' }, u.message) : null,
      h('p', { class: 'seg' },
        i.available && i.supported ? h('button', { onclick: update, disabled: working }, working ? 'Updating…' : `Update to ${i.latest}`) : null,
        h('button', { class: 'sec', onclick: check, disabled: working }, 'Check now')),
      i.available && !i.supported ? h('p', { class: 'muted' }, 'Installing from the app works only on the Linux Docker image. Update CarTrak itself to get the newer connector.') : null,
      msg,
      h('p', { class: 'muted' }, 'The new connector is downloaded from GitHub, checked against the checksum GitHub publishes, and must run. If remote access is on, it drops for a few seconds while the connector restarts, and if the tunnel does not reconnect within 90 seconds the old connector is put back. CarTrak itself updates through TrueNAS, not here.')].filter(Boolean));
  }

  box.append(h('h2', {}, 'Cloudflare connector'), msg);
  load().catch((e) => { msg.textContent = e.message; });
  return { el: box, cleanup: () => { if (timer) clearInterval(timer); } };
}
