// The last few things that went wrong in the app on this phone, kept on the phone only, so a debug report can include them.
const KEY = 'cartrak.errors';

export function recentErrors() {
  try { return JSON.parse(localStorage.getItem(KEY) || '[]'); } catch (e) { return []; }
}

function keep(msg) {
  try {
    const list = recentErrors();
    list.push({ t: Math.floor(Date.now() / 1000), page: (location.hash.split('/')[1] || 'home').slice(0, 20), msg: String(msg).slice(0, 200) });
    localStorage.setItem(KEY, JSON.stringify(list.slice(-10)));
  } catch (e) { /* storage blocked: nothing to keep */ }
}

export function watchErrors() {
  addEventListener('error', (e) => keep(e.message || 'error'));
  addEventListener('unhandledrejection', (e) => keep((e.reason && e.reason.message) || e.reason || 'rejected'));
}

// The part of a debug report only the phone knows: which app version it loaded, how it is shown, and its recent errors.
// the version these files were loaded as: their address carries it (/v0.46.0/js/...)
const loaded = (import.meta.url.match(/\/v([^/]+)\/js\//) || [])[1] || 'unknown';

export function phoneSection() {
  const errs = recentErrors();
  const standalone = window.matchMedia('(display-mode: standalone)').matches || navigator.standalone === true;
  return ['', 'THIS PHONE (added by the page)', `    app loaded as v${loaded}, ${standalone ? 'installed' : 'in the browser'}, screen ${innerWidth}x${innerHeight}, ${navigator.onLine ? 'online' : 'offline'}`,
    errs.length ? '    recent app errors (newest last):' : '    no app errors kept',
    ...errs.map((e) => `      ${new Date(e.t * 1000).toISOString().slice(0, 16).replace('T', ' ')} UTC | ${e.page} | ${e.msg}`)].join('\n');
}
