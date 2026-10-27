import { h } from './ui.js';
import { api } from './api.js';

// Tracking, on the phone's side. When a tracking session is running (started on the Developer page), the pages opened, the buttons
// tapped (by their label, never what was typed) and the errors are sent to the server, which records them with every request the
// app makes. The server decides whether a session is running; this only asks now and then.
let on = false;
let queue = [];
let wired = false;
let checkedAt = 0;
let pill = null;

const page = () => location.hash || '#/';
const push = (kind, a) => {
  if (!on) return;
  queue.push({ ts: Math.floor(Date.now() / 1000), kind, a: String(a).slice(0, 200), b: page() });
  if (queue.length >= 40) flush();
};

async function flush() {
  if (!on || !queue.length) return;
  const events = queue.splice(0, 300);
  try { await api('/api/track/events', { method: 'POST', body: { events } }); } catch (e) { queue = events.slice(0, 100).concat(queue); }
}

const labelOf = (el) => {
  const t = el.closest('button, a, [role="button"], select, summary, label, input, textarea');
  if (!t) return '';
  if (t.matches('input, textarea')) return `${t.type || 'field'}: ${t.getAttribute('aria-label') || t.placeholder || t.id || ''}`; // never the value
  return (t.getAttribute('aria-label') || t.innerText || t.title || t.getAttribute('href') || '').trim().replace(/\s+/g, ' ');
};

function wire() {
  if (wired) return;
  wired = true;
  document.addEventListener('click', (e) => { const l = e.target instanceof Element ? labelOf(e.target) : ''; if (l) push('tap', l); }, true);
  window.addEventListener('hashchange', () => push('page', location.hash));
  window.addEventListener('error', (e) => push('error', e.message || 'error'));
  window.addEventListener('unhandledrejection', (e) => push('error', (e.reason && e.reason.message) || 'rejected promise'));
  document.addEventListener('visibilitychange', () => { if (document.hidden) flush(); });
  setInterval(flush, 5000);
}

function showPill(active) {
  if (active && !pill) {
    pill = h('a', { href: '#/developer', 'aria-label': 'Tracking is on', style: 'position:fixed;right:12px;bottom:calc(76px + env(safe-area-inset-bottom,0px));z-index:60;background:#b91c1c;color:#fff;padding:8px 14px;border-radius:999px;font-weight:600;text-decoration:none;font-size:.85rem;min-height:36px;display:flex;align-items:center' }, '● Tracking');
    document.body.append(pill);
  } else if (!active && pill) { pill.remove(); pill = null; }
}

// trackCheck asks the server whether a session is running (at most every 20 seconds unless forced) and starts or stops recording.
export async function trackCheck(force) {
  const now = Date.now();
  if (!force && now - checkedAt < 20000) return;
  checkedAt = now;
  try {
    const s = await api('/api/track', { noOffline: true });
    if (s.active && !on) { on = true; wire(); push('page', location.hash || '#/'); }
    if (!s.active && on) { await flush(); on = false; }
    showPill(s.active);
  } catch (e) { /* not an admin, or offline: nothing to record */ }
}

export const trackingOn = () => on;
