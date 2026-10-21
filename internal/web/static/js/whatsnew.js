import { h } from './ui.js';
import { renderChangelog } from './pages/changelog.js';

// After an update, the first time the app opens: a sheet with what changed in every version since this device last opened it,
// so skipping a few updates still shows them all. "Got it" closes it until the next update; the full list stays in More.
export async function showWhatsNew(session) {
  const a = session.about;
  let seen = '';
  try { seen = localStorage.getItem('seenVersion') || ''; } catch (e) { return; }
  if (!a || !a.justUpdated || !seen) return;
  const text = await fetch('/api/changelog', { credentials: 'same-origin' }).then((r) => r.text()).catch(() => '');
  // the sections newer than the one this device last saw (the changelog is newest first)
  const parts = text.split(/\n(?=## )/).filter((p) => p.startsWith('## '));
  const fresh = [];
  for (const p of parts) {
    if (p.slice(3).split('\n')[0].trim() === seen) break;
    fresh.push(p);
    if (fresh.length >= 6) break; // a phone left for months gets the latest six; the rest are in What's new
  }
  const done = () => { try { localStorage.setItem('seenVersion', a.version); } catch (e) { /* ignore */ } a.justUpdated = false; bg.remove(); };
  const sheet = h('div', { class: 'sheet more', role: 'dialog', 'aria-label': "What's new" },
    h('div', { class: 'mhandle', 'aria-hidden': 'true' }),
    h('div', { class: 'mhead' }, h('h2', {}, `✨ What's new in v${a.version}`)),
    h('div', { class: 'wnew' }, ...renderChangelog(fresh.join('\n') || `## ${a.version}\n- Small improvements and fixes.`, '')),
    h('p', { class: 'seg', style: 'margin-top:14px' }, h('button', { type: 'button', class: 'on', onclick: done }, 'Got it'), h('a', { class: 'btn sec', href: '#/changelog', onclick: done }, 'Everything new')));
  const bg = h('div', { class: 'sheetbg bottom' }, sheet);
  bg.addEventListener('click', (e) => { if (e.target === bg) done(); });
  document.body.append(bg);
}
