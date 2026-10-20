// The server fills in __V__ with the running version, so every version has its own cache and its own file addresses
// (/v0.26.0/js/app.js). Nothing needs bumping by hand.
const V = '__V__';
const CACHE = 'cartrak-' + V;
const SHELL = ['/', '/css/app.css', '/css/menu.css', '/css/responsive.css', '/css/look.css', '/css/history.css', '/manifest.webmanifest', '/icons/icon.svg',
  '/js/app.js', '/js/api.js', '/js/ui.js',
  '/js/pages/login.js', '/js/pages/start.js', '/js/pages/admin.js', '/js/pages/connect.js', '/js/offline.js', '/js/pages/remote.js', '/js/pages/ford.js', '/js/pages/display.js', '/js/pages/people.js', '/js/pages/about.js', '/js/pages/status.js', '/js/healthstrip.js', '/js/cards.js', '/js/pages/live.js', '/js/datarows.js', '/js/timeline.js',
  '/js/rangeband.js', '/js/ring.js', '/js/connectorcard.js', '/js/mileagecheck.js', '/js/energy.js', '/js/chargeforecast.js', '/js/charge.js', '/js/chargecard.js', '/js/pages/notifications.js', '/js/push.js', '/js/pages/mileage.js', '/js/pages/security.js', '/js/pages/plan.js', '/js/pages/maintenance.js', '/js/pages/places.js', '/js/pages/insights.js', '/js/pages/timeuse.js', '/js/pages/parkedwatch.js', '/js/pages/guide.js', '/js/maplinks.js', '/js/restorecard.js', '/js/storagecard.js', '/js/pages/review.js', '/js/placecard.js', '/js/pages/journey.js', '/js/tripselect.js', '/js/sharecard.js', '/js/tripplan.js', '/js/pages/backup.js', '/js/map.js', '/js/pages/vehiclehealth.js', '/js/healthrows.js', '/js/spark.js', '/js/pages/phone.js', '/js/phonehelp.js', '/js/gpslog.js', '/js/gpscard.js', '/js/mode.js', '/js/diag.js', '/js/pages/changelog.js', '/js/notice.js', '/js/aboutcard.js', '/js/pages/dashboard.js', '/js/pages/trips.js', '/js/pages/tripdetail.js', '/js/pages/compare.js', '/js/pages/datacheck.js', '/js/pages/year.js', '/js/pages/debug.js', '/js/livepower.js', '/js/chargenow.js', '/js/chargestate.js', '/js/vacation.js', '/js/searchindex.js', '/js/curve.js', '/js/bestspeed.js', '/js/bestspeedcard.js', '/js/reasonscard.js', '/js/chart.js', '/js/speedcard.js', '/js/tirecard.js', '/js/tripwhy.js', '/js/pages/repair.js', '/js/tripfix.js', '/js/batterycard.js', '/js/pages/history.js', '/js/pages/stats.js', '/js/placedonut.js', '/js/speedcompare.js', '/js/remotestart.js', '/js/pages/day.js', '/js/daydetail.js', '/js/car.js', '/js/routes.js', '/js/searchsheet.js', '/js/moresheet.js', '/js/inboxstate.js', '/js/homecards.js', '/js/pages/inbox.js', '/js/pages/reports.js', '/js/tripcard.js', '/js/tripdistance.js', '/js/reportbuilder.js', '/js/reporteditor.js', '/js/categories.js', '/js/pages/categories.js', '/js/pages/chargingsettings.js', '/js/insightcard.js', '/js/pages/help.js', '/js/pages/cars.js', '/js/routeestimatebar.js', '/js/pages/fields.js',
  '/js/pages/charging.js', '/js/pages/data.js', '/js/pages/settings.js', '/js/pages/rules.js'].map((p) => (p.startsWith('/js/') || p.startsWith('/css/') ? `/v${V}${p}` : p));

self.addEventListener('install', (e) => {
  // cache: 'reload' skips any copy kept by the browser or a CDN in front of the server, so a new version really downloads.
  e.waitUntil(caches.open(CACHE).then((c) => c.addAll(SHELL.map((u) => new Request(u, { cache: 'reload' })))).then(() => self.skipWaiting()));
});
self.addEventListener('activate', (e) => {
  e.waitUntil(caches.keys().then((ks) => Promise.all(ks.filter((k) => k !== CACHE).map((k) => caches.delete(k)))).then(() => self.clients.claim()));
});
// App files: cache first. API and Ford link routes always go to the network
// (never cached: they carry private vehicle data).
self.addEventListener('fetch', (e) => {
  const u = new URL(e.request.url);
  if (e.request.method !== 'GET' || u.origin !== location.origin || u.pathname.startsWith('/api/') || u.pathname.startsWith('/ford/') || u.pathname.startsWith('/s/')) return;
  e.respondWith(caches.match(e.request).then((hit) => hit || fetch(e.request)));
});

// Alerts: the server sends a short encrypted message. Show it, and open the right page when tapped.
self.addEventListener('push', (e) => {
  let d = {};
  try { d = e.data ? e.data.json() : {}; } catch (err) { d = { body: e.data ? e.data.text() : '' }; }
  e.waitUntil(self.registration.showNotification(d.title || 'CarTrak', {
    body: d.body || '', tag: d.tag || undefined, icon: '/icons/icon-192.png', badge: '/icons/icon-192.png', data: { url: d.url || '#/' },
  }));
});
self.addEventListener('notificationclick', (e) => {
  e.notification.close();
  const target = '/' + ((e.notification.data && e.notification.data.url) || '#/');
  e.waitUntil(self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then(async (cs) => {
    for (const c of cs) {
      if ('navigate' in c) await c.navigate(target).catch(() => {});
      return c.focus();
    }
    return self.clients.openWindow(target);
  }));
});
