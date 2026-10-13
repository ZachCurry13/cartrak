import { h } from '../ui.js';
import { aboutCard } from '../aboutcard.js';
import { notices } from '../notice.js';

export default async function about(root, _a, session) {
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'About CarTrak')),
    ...notices(session), aboutCard(session, session.me.role === 'admin'),
    h('p', { class: 'muted foot' }, 'CarTrak is an independent open-source project and is not affiliated with or endorsed by Ford Motor Company.'));
}
