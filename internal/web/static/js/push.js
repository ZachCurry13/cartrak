import { api } from './api.js';

// Helpers for Web Push on this device. The service worker (sw.js) shows the messages.

const b64 = (s) => Uint8Array.from(atob(s.replace(/-/g, '+').replace(/_/g, '/') + '='.repeat((4 - (s.length % 4)) % 4)), (c) => c.charCodeAt(0));
const enc = (buf) => btoa(String.fromCharCode(...new Uint8Array(buf))).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');

// Why this device cannot get alerts, or '' when it can.
export function unsupported() {
  const ios = /iphone|ipad|ipod/i.test(navigator.userAgent);
  const installed = matchMedia('(display-mode: standalone)').matches || navigator.standalone;
  if (ios && !installed) return 'On iPhone and iPad, add CarTrak to the Home Screen first (Share, then Add to Home Screen), then open it from there.';
  if (!('serviceWorker' in navigator) || !('PushManager' in window) || !('Notification' in window)) return 'This browser cannot receive alerts.';
  if (!window.isSecureContext) return 'Alerts need the secure (https) address, such as your Cloudflare address, not http on your home network.';
  return '';
}

export async function registration() {
  return navigator.serviceWorker.ready;
}

// The push address of this device, or null when it is not subscribed.
export async function current() {
  const sub = await (await registration()).pushManager.getSubscription();
  return sub ? sub.endpoint : null;
}

// Asks permission, subscribes this device and tells the server which alerts it wants.
export async function enable(publicKey, label, events, quiet) {
  if (await Notification.requestPermission() !== 'granted') throw new Error('Notifications are blocked for this site. Allow them in the browser settings, then try again.');
  const reg = await registration();
  let sub = await reg.pushManager.getSubscription();
  if (!sub) sub = await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: b64(publicKey) });
  return api('/api/push/subscribe', { method: 'POST', body: { endpoint: sub.endpoint, keys: { p256dh: enc(sub.getKey('p256dh')), auth: enc(sub.getKey('auth')) }, label, events, quiet } });
}

export async function disable() {
  const sub = await (await registration()).pushManager.getSubscription();
  if (!sub) return null;
  const out = await api('/api/push/unsubscribe', { method: 'POST', body: { endpoint: sub.endpoint } });
  await sub.unsubscribe();
  return out;
}
