// Bump CACHE whenever any file in SHELL changes.
const CACHE = 'cartrak-v0.9.0-7';
const SHELL = ['/', '/css/app.css', '/css/theme.css', '/css/menu.css', '/css/responsive.css', '/manifest.webmanifest', '/icons/icon.svg',
  '/js/app.js', '/js/api.js', '/js/ui.js',
  '/js/pages/login.js', '/js/pages/start.js', '/js/pages/admin.js', '/js/pages/connect.js', '/js/offline.js', '/js/pages/remote.js', '/js/pages/ford.js', '/js/pages/display.js', '/js/pages/people.js', '/js/pages/about.js', '/js/pages/status.js', '/js/healthstrip.js', '/js/cards.js', '/js/pages/live.js', '/js/pages/vehiclehealth.js', '/js/healthrows.js', '/js/spark.js', '/js/pages/phone.js', '/js/phonehelp.js', '/js/gpslog.js', '/js/gpscard.js', '/js/mode.js', '/js/diag.js', '/js/pages/changelog.js', '/js/notice.js', '/js/aboutcard.js', '/js/pages/dashboard.js', '/js/pages/trips.js', '/js/pages/tripdetail.js',
  '/js/pages/charging.js', '/js/pages/data.js', '/js/pages/settings.js', '/js/pages/rules.js'];

self.addEventListener('install', (e) => {
  e.waitUntil(caches.open(CACHE).then((c) => c.addAll(SHELL)).then(() => self.skipWaiting()));
});
self.addEventListener('activate', (e) => {
  e.waitUntil(caches.keys().then((ks) => Promise.all(ks.filter((k) => k !== CACHE).map((k) => caches.delete(k)))).then(() => self.clients.claim()));
});
// App files: cache first. API and Ford link routes always go to the network
// (never cached: they carry private vehicle data).
self.addEventListener('fetch', (e) => {
  const u = new URL(e.request.url);
  if (e.request.method !== 'GET' || u.origin !== location.origin || u.pathname.startsWith('/api/') || u.pathname.startsWith('/ford/')) return;
  e.respondWith(caches.match(e.request).then((hit) => hit || fetch(e.request)));
});
