import { h } from './ui.js';
import { api } from './api.js';

// Telling the person a new version is ready. loadedVersion() is the version this page started with: a phone app can stay open for days.
function showUpdateBar() {
  if (document.getElementById('upd')) return;
  document.body.append(h('div', { id: 'upd', class: 'updbar', role: 'status' }, 'A new version of CarTrak is ready. ',
    h('button', { onclick: () => location.reload() }, 'Reload')));
}

export function watchUpdates(loadedVersion) {
  if (!('serviceWorker' in navigator)) return;
  // The app files are cached on the device. After an update the new files arrive in the background while the page
  // still runs the old ones, which is how an old screen can sit next to a new server version. Say so, with a button.
  const hadWorker = !!navigator.serviceWorker.controller;
  navigator.serviceWorker.addEventListener('controllerchange', () => { if (hadWorker) showUpdateBar(); });
  // A phone keeps the app open in the background for days. When it comes back, check for a new version at once, both
  // the cached files and the version the server reports.
  document.addEventListener('visibilitychange', async () => {
    if (document.hidden) return;
    navigator.serviceWorker.getRegistration().then((r) => r && r.update()).catch(() => {});
    try {
      const a = await api('/api/about');
      const loaded = loadedVersion();
      if (loaded && a.version && a.version !== loaded) showUpdateBar();
    } catch (e) { /* offline or signed out: nothing to compare */ }
  });
  navigator.serviceWorker.register('/sw.js').catch(() => {});
}
