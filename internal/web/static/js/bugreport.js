// "Send a bug report" at the foot of every page (administrators): keeps a text copy of the page with a note, in the app (More, Bug
// reports), so the problem can be read later instead of described. It holds the page's words, the device and screen, facts about
// the layout and the debug report. A link made on the Bug reports page lets Claude read them.
import { h } from './ui.js';
import { api } from './api.js';
import { deviceInfo, pageText } from './pagecapture.js';
import { layoutFacts } from './layoutfacts.js';
import { pageSnapshot } from './pagesnapshot.js';
import { phoneSection } from './errlog.js';
import { trapFocus } from './focustrap.js';

const HIDDEN_ON = ['#/live', '#/login'];

async function debugText() {
  const d = await api('/api/debug-report?hours=24');
  return d.text + phoneSection();
}

function sheet(info, admin) {
  const close = () => { bg.remove(); document.removeEventListener('keydown', onKey); };
  const onKey = (e) => { if (e.key === 'Escape') close(); };
  const msg = h('p', { class: 'muted', role: 'status' });
  const note = h('textarea', { rows: 4, maxlength: 4000, placeholder: 'What went wrong? What did you expect?', 'aria-label': 'What went wrong', style: 'width:100%;box-sizing:border-box' });
  const withDebug = h('input', { type: 'checkbox', class: 'check', id: 'br-debug' });
  const send = h('button', { type: 'button', onclick: async () => {
    send.disabled = true;
    const began = Date.now();
    const tick = setInterval(() => { if (msg.dataset.phase) msg.textContent = `${msg.dataset.phase} ${Math.round((Date.now() - began) / 1000)} s`; }, 1000);
    const phase = (t) => { msg.dataset.phase = t; msg.textContent = t; };
    try {
      phase('Saving…');
      let debug = '';
      if (withDebug.checked) { phase('Collecting the debug report (this part can be slow)…'); debug = await debugText(); phase('Saving…'); }
      const r = await api('/api/bug-reports', { method: 'POST', body: { note: note.value, page: info.page, device: info.device, text: info.text, layout: info.layout, html: info.html, debug } });
      if (admin) msg.replaceChildren(`Saved as report #${r.id}. `, h('a', { href: '#/bugreports', onclick: close }, 'Open Bug reports'), ' to make a link for Claude.');
      else msg.textContent = `Saved as report #${r.id}. Thank you.`;
      send.hidden = true;
    } catch (e) { delete msg.dataset.phase; msg.textContent = e.message; send.disabled = false; }
    clearInterval(tick); delete msg.dataset.phase;
  } }, 'Save report');
  const dlg = h('div', { class: 'sheet', role: 'dialog', 'aria-modal': 'true', 'aria-label': 'Send a bug report', tabindex: '-1' }, h('h2', {}, 'Send a bug report'),
    h('p', { class: 'muted' }, `Saved in the app as text from this page (${info.page}): its words, a copy of how it looks, and your screen (${info.device.split(';')[0]}).`), note,
    admin ? h('label', { for: 'br-debug', style: 'display:flex;align-items:center;gap:10px;margin-top:10px' }, withDebug, 'Include the debug report (only for a problem with the numbers; it is long)') : null,
    h('p', {}, send, ' ', h('button', { class: 'text', type: 'button', onclick: close }, 'Close')), msg);
  const bg = h('div', { class: 'sheetbg' }, dlg);
  bg.addEventListener('click', (e) => { if (e.target === bg) close(); });
  document.addEventListener('keydown', onKey);
  trapFocus(dlg);
  document.body.append(bg);
  dlg.focus({ preventScroll: true });
}

export function mountBugReport(session) {
  const foot = document.getElementById('bugfoot');
  if (!foot) return; // everyone signed in can send one; only an administrator reads them
  const btn = h('button', { class: 'text sm', type: 'button', onclick: () => sheet({ page: location.hash || '#/', device: deviceInfo(), text: pageText(), layout: layoutFacts(), html: pageSnapshot() }, session.me.role === 'admin') }, 'Send a bug report');
  foot.replaceChildren(btn);
  const place = () => { foot.hidden = HIDDEN_ON.some((p) => location.hash.startsWith(p)); };
  addEventListener('hashchange', place);
  place();
}
