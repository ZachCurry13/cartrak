import { h } from '../ui.js';

// **bold** inside a line, as text nodes and <b> elements: never HTML.
const inline = (text) => text.split('**').map((part, i) => (i % 2 ? h('b', {}, part) : part)).filter((x) => x !== '');

// Minimal markdown for our own CHANGELOG.md: ## version, ### group, - bullet. Text only, never HTML.
export function renderChangelog(md, current) {
  const out = [];
  let list = null;
  for (const line of md.split('\n')) {
    if (line.startsWith('# ')) continue;
    if (line.startsWith('## ')) {
      list = null;
      const v = line.slice(3).trim();
      out.push(h('h2', { style: 'margin-top:20px' }, v, v === current ? h('span', { class: 'chip ok', style: 'margin-left:8px' }, 'installed') : null));
    } else if (line.startsWith('### ')) {
      list = null;
      out.push(h('h3', { style: 'margin:12px 0 4px;font-size:1rem' }, line.slice(4).trim()));
    } else if (line.startsWith('- ')) {
      if (!list) { list = h('ul', { style: 'margin:0 0 8px;padding-left:20px' }); out.push(list); }
      list.append(h('li', {}, ...inline(line.slice(2).trim())));
    } else if (line.trim()) {
      list = null;
      out.push(h('p', {}, ...inline(line.trim())));
    }
  }
  return out;
}

export default async function changelog(root, _a, session) {
  const r = await fetch('/api/changelog', { credentials: 'same-origin' });
  const text = await r.text();
  try { localStorage.setItem('seenVersion', session.about.version); } catch (e) { /* ignore */ }
  session.about.justUpdated = false;
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('h2', {}, "What's new"),
    h('p', { class: 'muted' }, `You are on v${session.about.version}.`),
    h('section', { class: 'card' }, ...renderChangelog(text, session.about.version)));
}
