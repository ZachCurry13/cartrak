// The last few things that went wrong in the app on this phone, kept on the phone only, so a debug report can include them.
import { slowestRequests } from './reqtimes.js';

const KEY = 'cartrak.errors';

export function recentErrors() {
  try { return JSON.parse(localStorage.getItem(KEY) || '[]'); } catch (e) { return []; }
}

// where in the code it happened, so an error can be found later: the file and line of an error, or the first line of a rejection's stack
const where = (file, line, stack) => (file ? `${String(file).split('/').pop()}:${line}` : String(stack || '').split('\n').map((l) => l.trim()).find((l) => l.includes('/js/')) || '');

function keep(msg, at = '') {
  try {
    const list = recentErrors();
    list.push({ t: Math.floor(Date.now() / 1000), page: (location.hash.split('/')[1] || 'home').slice(0, 20), msg: String(at ? `${msg} @ ${at}` : msg).slice(0, 240) });
    localStorage.setItem(KEY, JSON.stringify(list.slice(-10)));
  } catch (e) { /* storage blocked: nothing to keep */ }
}

export function watchErrors() {
  addEventListener('error', (e) => keep(e.message || 'error', where(e.filename, e.lineno)));
  addEventListener('unhandledrejection', (e) => keep((e.reason && e.reason.message) || e.reason || 'rejected', where('', 0, e.reason && e.reason.stack)));
}

// The part of a debug report only the phone knows: which app version it loaded, how it is shown, and its recent errors.
// the version these files were loaded as: their address carries it (/v0.46.0/js/...)
const loaded = (import.meta.url.match(/\/v([^/]+)\/js\//) || [])[1] || 'unknown';

export function phoneSection() {
  const slow = slowestRequests();
  const errs = recentErrors();
  const standalone = window.matchMedia('(display-mode: standalone)').matches || navigator.standalone === true;
  return ['', 'THIS PHONE (added by the page)', `    app loaded as v${loaded}, ${standalone ? 'installed' : 'in the browser'}, screen ${innerWidth}x${innerHeight}, ${navigator.onLine ? 'online' : 'offline'}`,
    errs.length ? '    recent app errors (newest last):' : '    no app errors kept',
    ...errs.map((e) => `      ${new Date(e.t * 1000).toISOString().slice(0, 16).replace('T', ' ')} UTC | ${e.page} | ${e.msg}`),
    slow.length ? '    slowest requests on this phone since the app opened (includes the connection):' : '    no requests timed yet', ...slow].join('\n');
}
