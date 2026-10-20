import { h, when } from '../ui.js';
import { api } from '../api.js';
import { restoreCard } from '../restorecard.js';
import { storageCard } from '../storagecard.js';

const size = (b) => (b >= 1e9 ? `${(b / 1e9).toFixed(1)} GB` : `${Math.max(1, Math.round(b / 1e6))} MB`);

// Download a copy of everything CarTrak has stored.
export default async function backup(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const d = await api('/api/backup/info');
  const link = (href, text, sec) => h('a', { class: `btn${sec ? ' sec' : ''}`, href, download: '' }, text);
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Backup')),
    h('section', { class: 'card' }, h('h2', {}, session.cars.length > 1 ? 'This car’s data' : 'Your data'),
      h('p', {}, `CarTrak's database is ${size(d.dbBytes)}${d.firstReading ? `, kept since ${when(d.firstReading)}` : ''}. Nothing is deleted as it ages.`),
      h('p', { class: 'muted' }, 'Most of the size is the raw copy of every reading from Ford. Trips, charging, places, settings and the charts are small.'),
      h('p', { class: 'seg' }, link('/api/backup?small=1', 'Download small backup'), link('/api/backup', 'Download full backup', true)),
      h('p', { class: 'muted' }, 'The small backup keeps everything except the raw Ford readings (Car data history for old readings will be empty). Making a backup can take a while on a large database, and the app is slow while it runs. Sign-in sessions are never included.'),
      h('p', { class: 'banner' }, 'A backup contains the Ford sign-in of this car. Keep it private, like a password.')),
    session.me.siteRole === 'admin' && session.cars.length > 1 ? h('section', { class: 'card' }, h('h2', {}, 'Every car and every person'),
      h('p', { class: 'muted' }, 'One zip with the people and the database of each car. A backup of one car (above) holds only that car and no accounts. To put the whole zip back, stop CarTrak, replace core.db and the database file of each car in the data folder with the ones in the zip, and start it again.'),
      h('p', { class: 'seg' }, link('/api/backup/site', 'Download everything (all cars)')),
      h('p', { class: 'banner' }, 'It contains the password hash of every account and the Ford sign-in of every car. Keep it private.')) : '',
    h('section', { class: 'card' }, h('h2', {}, 'Export to spreadsheets'),
      h('p', { class: 'muted' }, 'One zip with every trip, every charging session, your named places and your service reminders as spreadsheets (CSV), to open in Excel or Sheets. It is for reading your data elsewhere; it is not a backup and cannot be restored. It shows where the car has been, so keep it private.'),
      h('p', { class: 'seg' }, link('/api/export/all.zip', 'Download everything as a zip'))),
    storageCard(), restoreCard());
}
