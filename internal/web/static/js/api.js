import { currentCar } from './car.js';

export class AuthError extends Error {}

export async function api(path, opts = {}) {
  const init = { credentials: 'same-origin', headers: {}, method: opts.method || 'GET' };
  if (init.method !== 'GET') init.headers['X-Requested-With'] = 'cartrak';
  const car = currentCar();
  if (car) init.headers['X-Car'] = car;
  if (opts.body !== undefined) {
    init.headers['Content-Type'] = 'application/json';
    init.body = JSON.stringify(opts.body);
  }
  const r = await fetch(path, init);
  if (r.status === 401 && !opts.noAuthRedirect) throw new AuthError('login required');
  const text = await r.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch (e) { data = text; }
  if (!r.ok) throw new Error((data && data.error) || `HTTP ${r.status}`);
  return data;
}
