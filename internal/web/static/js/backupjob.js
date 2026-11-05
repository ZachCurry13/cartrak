import { h } from './ui.js';
import { api } from './api.js';

// A backup of a big database takes minutes to make. The button starts the copy on the server, shows that it is working, and
// starts the download when the copy is ready, so the page never looks as if nothing happened.
export function backupButton(text, small, sec) {
  const note = h('p', { class: 'muted', role: 'status', hidden: true });
  const btn = h('button', { class: sec ? 'sec' : '', type: 'button' }, text);
  const say = (msg) => { note.hidden = !msg; note.textContent = msg || ''; };
  btn.onclick = async () => {
    btn.disabled = true;
    say('Making the backup. A large database can take a few minutes, and you can leave this page open.');
    try {
      const { id } = await api(`/api/backup/start${small ? '?small=1' : ''}`, { method: 'POST' });
      for (;;) {
        await new Promise((ok) => setTimeout(ok, 2000));
        const s = await api(`/api/backup/status?id=${id}`);
        if (s.state === 'failed') throw new Error(s.error);
        if (s.state === 'ready') {
          const mb = Math.max(1, Math.round(s.bytes / 1e6));
          say(`Ready (${mb} MB). The download is starting. If nothing happens, check your browser’s downloads.`);
          const a = h('a', { href: `/api/backup/file?id=${id}`, download: '' });
          document.body.append(a); a.click(); a.remove();
          break;
        }
        say(`Making the backup… ${s.seconds} s so far. A large database can take a few minutes.`);
      }
    } catch (e) {
      say(`The backup did not finish: ${e.message}`);
    }
    btn.disabled = false;
  };
  return h('div', {}, btn, note);
}
