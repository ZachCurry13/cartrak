import { currentCar } from './car.js';
import { saveAnswer, loadAnswer, forgetOffline } from './offlinestore.js';
import { windowKey } from './offlinekey.js';
import { showOffline, hideOffline } from './offlinebar.js';

export class AuthError extends Error {}

// Screens that read secrets, tokens, people or downloads are never kept on the device.
const NEVER_KEPT = /^\/api\/(share|shares|reports|debug|export|backup|restore|security|ford|login|logout|setup|users|people|push|devices|admin|connector|ha|remote|tiles|cars\/[^/]+\/(access|backup))(\/|\?|$)/;

export async function api(path, opts = {}) {
  const init = { credentials: 'same-origin', headers: {}, method: opts.method || 'GET' };
  if (init.method !== 'GET') init.headers['X-Requested-With'] = 'cartrak';
  let units = 'imperial';
  try { if (localStorage.getItem('units') === 'metric') units = 'metric'; } catch (e) { /* storage blocked */ }
  init.headers['X-Units'] = units; // the server writes messages in the units this device uses
  const car = opts.car != null ? String(opts.car) : currentCar(); // opts.car asks about another car (All cars)
  if (car) init.headers['X-Car'] = car;
  if (opts.body !== undefined) {
    init.headers['Content-Type'] = 'application/json';
    init.body = JSON.stringify(opts.body);
  }
  const keep = init.method === 'GET' && !opts.noOffline && !NEVER_KEPT.test(path);
  const prefix = `${car || ''}|${units}|`, key = prefix + path, wkey = keep ? windowKey(prefix, path) : null;
  let r;
  try {
    r = await fetch(path, init);
  } catch (e) {
    if (!(e instanceof TypeError)) throw e; // fetch() rejects with a TypeError when there is no connection
    if (!keep) throw new Error('You are offline. This needs a connection.');
    const hit = (await loadAnswer(key)) || (wkey && (await loadAnswer(wkey)));
    if (!hit) throw e;
    showOffline(hit.t);
    return hit.v;
  }
  if (r.status === 401) { forgetOffline(); if (!opts.noAuthRedirect) throw new AuthError('login required'); }
  const text = await r.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch (e) { data = text; }
  if (!r.ok) throw new Error((data && data.error) || `HTTP ${r.status}`);
  hideOffline();
  if (keep && data && typeof data === 'object') { saveAnswer(key, data); if (wkey) saveAnswer(wkey, data); }
  return data;
}
