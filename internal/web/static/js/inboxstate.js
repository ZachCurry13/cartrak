import { api } from './api.js';

// The bell's count is the alerts newer than the last time the inbox was opened on this device.
const KEY = 'inboxSeen';
const get = () => { try { return Number(localStorage.getItem(KEY)) || 0; } catch (e) { return 0; } };
const set = (v) => { try { localStorage.setItem(KEY, String(v)); } catch (e) { /* storage blocked */ } };

export const inboxSeenBefore = get;

export function markInboxSeen(items) {
  const top = items.reduce((m, x) => Math.max(m, x.ts), 0);
  set(Math.max(top, Math.floor(Date.now() / 1000)));
  paintBell(0);
}

function paintBell(n) {
  const b = document.getElementById('belln');
  if (!b) return;
  b.hidden = n <= 0;
  b.textContent = n > 99 ? '99+' : String(n);
  const bell = document.getElementById('bell');
  if (bell) bell.setAttribute('aria-label', n > 0 ? `Notifications, ${n} new` : 'Notifications');
}

// Counts what is new and shows it on the bell. The first time on a device nothing counts as new.
export async function refreshBell() {
  try {
    const { items } = await api('/api/inbox');
    let seen = get();
    if (!seen) { set(Math.floor(Date.now() / 1000)); seen = get(); }
    paintBell(items.filter((x) => x.ts > seen).length);
  } catch (e) { /* offline: leave the bell as it was */ }
}
