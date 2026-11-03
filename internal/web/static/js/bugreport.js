// "Send a bug report" at the foot of every page (administrators): captures what is on screen, takes a note, and sends the lot to the
// private repository set up in Settings, so it can be read later instead of describing the page in a chat. The report shows the
// screen it came from: a phone screen from a phone, a window from a computer.
import { h } from './ui.js';
import { api } from './api.js';
import { deviceInfo, pageText, screenshot } from './pagecapture.js';
import { phoneSection } from './errlog.js';
import { trapFocus } from './focustrap.js';

const HIDDEN_ON = ['#/live', '#/login'];

async function debugText() {
  const d = await api('/api/debug-report?hours=24');
  return d.text + phoneSection();
}

function sheet(info, configured) {
  const close = () => { bg.remove(); document.removeEventListener('keydown', onKey); };
  const onKey = (e) => { if (e.key === 'Escape') close(); };
  const msg = h('p', { class: 'muted', role: 'status' });
  let body;
  if (!configured) {
    body = [h('p', {}, 'Bug reports are sent to a private GitHub repository that you set up once: a repository name and a token, under More, Settings (Bug report repository, Bug report token). The steps are in docs/bug-reports.md.'),
      h('p', {}, h('a', { class: 'btn', href: '#/admin', onclick: close }, 'Open Settings'), ' ', h('button', { class: 'text', type: 'button', onclick: close }, 'Close'))];
  } else {
    const note = h('textarea', { rows: 4, maxlength: 4000, placeholder: 'What went wrong? What did you expect?', 'aria-label': 'What went wrong', style: 'width:100%' });
    const withDebug = h('input', { type: 'checkbox', class: 'check', id: 'br-debug', checked: true });
    const send = h('button', { type: 'button', onclick: async () => {
      send.disabled = true;
      msg.textContent = 'Sending…';
      try {
        const debug = withDebug.checked ? await debugText() : '';
        const r = await api('/api/bug-report', { method: 'POST', body: { note: note.value, page: info.page, device: info.device, text: info.text, screenshot: info.shot || '', debug } });
        msg.textContent = `Sent as report ${r.number ? `#${r.number}` : r.id}. You can tell Claude to look at it.`;
        send.hidden = true;
      } catch (e) { msg.textContent = e.message; send.disabled = false; }
    } }, 'Send');
    body = [
      h('p', { class: 'muted' }, `Captured from this page (${info.page}): ${info.shot ? 'a picture, ' : 'no picture (this browser could not draw it), '}its text and your screen: ${info.device.split(';')[0]}.`),
      info.shot ? h('img', { src: info.shot, alt: 'The page as it will be sent', style: 'display:block;max-width:100%;max-height:180px;margin:0 0 var(--sp-3);border-radius:var(--r-sm);border:1px solid var(--line)' }) : null,
      note,
      h('label', { for: 'br-debug', style: 'display:flex;align-items:center;gap:10px;margin-top:10px' }, withDebug, 'Include the debug report (the last 24 hours)'),
      h('p', {}, send, ' ', h('button', { class: 'text', type: 'button', onclick: close }, 'Close')), msg];
  }
  const dlg = h('div', { class: 'sheet', role: 'dialog', 'aria-modal': 'true', 'aria-label': 'Send a bug report', tabindex: '-1' }, h('h2', {}, 'Send a bug report'), ...body.filter(Boolean));
  const bg = h('div', { class: 'sheetbg' }, dlg);
  bg.addEventListener('click', (e) => { if (e.target === bg) close(); });
  document.addEventListener('keydown', onKey);
  trapFocus(dlg);
  document.body.append(bg);
  dlg.focus({ preventScroll: true });
}

export function mountBugReport(session) {
  const foot = document.getElementById('bugfoot');
  if (!foot || session.me.role !== 'admin') return;
  const btn = h('button', { class: 'text sm', type: 'button', onclick: async () => {
    btn.disabled = true;
    try {
      const [shot, status] = await Promise.all([screenshot(), api('/api/bug-report/status').catch(() => ({ configured: false }))]); // before the sheet opens, so it is not in the picture
      sheet({ page: location.hash || '#/', device: deviceInfo(), text: pageText(), shot }, !!status.configured);
    } finally { btn.disabled = false; }
  } }, 'Send a bug report');
  foot.replaceChildren(btn);
  const place = () => { foot.hidden = HIDDEN_ON.some((p) => location.hash.startsWith(p)); };
  addEventListener('hashchange', place);
  place();
}
