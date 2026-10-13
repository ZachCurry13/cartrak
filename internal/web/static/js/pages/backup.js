import { h, when } from '../ui.js';
import { api } from '../api.js';

const size = (b) => (b >= 1e9 ? `${(b / 1e9).toFixed(1)} GB` : `${Math.max(1, Math.round(b / 1e6))} MB`);

// Download a copy of everything CarTrak has stored.
export default async function backup(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const d = await api('/api/backup/info');
  const link = (href, text, sec) => h('a', { class: `btn${sec ? ' sec' : ''}`, href, download: '' }, text);
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Backup')),
    h('section', { class: 'card' }, h('h2', {}, 'Your data'),
      h('p', {}, `CarTrak's database is ${size(d.dbBytes)}${d.firstReading ? `, kept since ${when(d.firstReading)}` : ''}. Nothing is deleted as it ages.`),
      h('p', { class: 'muted' }, 'Most of the size is the raw copy of every reading from Ford. Trips, charging, places, settings and the charts are small.'),
      h('p', { class: 'seg' }, link('/api/backup?small=1', 'Download small backup'), link('/api/backup', 'Download full backup', true)),
      h('p', { class: 'muted' }, 'The small backup keeps everything except the raw Ford readings (Car data history for old readings will be empty). Making a backup can take a while on a large database, and the app is slow while it runs. Sign-in sessions are never included.'),
      h('p', { class: 'banner' }, 'A backup contains your Ford sign-in, account password hashes and notification keys. Keep it private, like a password.')),
    h('section', { class: 'card' }, h('h2', {}, 'Restoring'),
      h('p', { class: 'muted' }, 'Stop CarTrak, replace cartrak.db in the data folder with the backup (deleting any cartrak.db-wal and cartrak.db-shm next to it), and start CarTrak again. See docs/backup.md.')));
}
