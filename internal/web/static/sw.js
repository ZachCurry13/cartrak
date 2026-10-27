// The server fills in __V__ with the running version, so every version has its own cache and its own file addresses
// (/v0.26.0/js/app.js). Nothing needs bumping by hand.
const V = '__V__';
const CACHE = 'cartrak-' + V;
const TILES = 'cartrak-tiles'; // map pictures seen on this device: kept across versions, wiped at sign-out
const MAX_TILES = 500;
const SHELL = ['/', '/css/app.css', '/css/menu.css', '/css/responsive.css', '/css/look.css', '/css/history.css', '/css/print.css', '/manifest.webmanifest', '/icons/icon.svg',
  '/js/app.js', '/js/api.js', '/js/ui.js',
  '/js/pages/login.js', '/js/pages/start.js', '/js/pages/admin.js', '/js/pages/connect.js', '/js/offline.js', '/js/pages/remote.js', '/js/pages/ford.js', '/js/pages/display.js', '/js/pages/people.js', '/js/pages/about.js', '/js/pages/status.js', '/js/healthstrip.js', '/js/cards.js', '/js/pages/live.js', '/js/datarows.js', '/js/timeline.js',
  '/js/rangeband.js', '/js/ring.js', '/js/connectorcard.js', '/js/mileagecheck.js', '/js/energy.js', '/js/chargeforecast.js', '/js/charge.js', '/js/stopcard.js', '/js/daystops.js', '/js/pages/stops.js', '/js/chargecard.js', '/js/pages/notifications.js', '/js/push.js', '/js/pages/mileage.js', '/js/pages/security.js', '/js/pages/plan.js', '/js/pages/planfuel.js', '/js/pages/maintenance.js', '/js/pages/places.js', '/js/pages/insights.js', '/js/pages/timeuse.js', '/js/pages/parkedwatch.js', '/js/pages/guide.js', '/js/maplinks.js', '/js/restorecard.js', '/js/storagecard.js', '/js/pages/review.js', '/js/placecard.js', '/js/pages/journey.js', '/js/tripselect.js', '/js/sharecard.js', '/js/tripplan.js', '/js/pages/backup.js', '/js/map.js', '/js/pages/vehiclehealth.js', '/js/healthrows.js', '/js/spark.js', '/js/pages/phone.js', '/js/phonehelp.js', '/js/gpslog.js', '/js/gpscard.js', '/js/mode.js', '/js/diag.js', '/js/pages/changelog.js', '/js/notice.js', '/js/aboutcard.js', '/js/pages/dashboard.js', '/js/pages/trips.js', '/js/pages/tripdetail.js', '/js/pages/compare.js', '/js/pages/datacheck.js', '/js/pages/year.js', '/js/pages/debug.js', '/js/livepower.js', '/js/chargenow.js', '/js/chargestate.js', '/js/vacation.js', '/js/searchindex.js', '/js/curve.js', '/js/bestspeed.js', '/js/bestspeedcard.js', '/js/reasonscard.js', '/js/chart.js', '/js/speedcard.js', '/js/tirecard.js', '/js/tripwhy.js', '/js/pages/repair.js', '/js/tripfix.js', '/js/batterycard.js', '/js/pages/history.js', '/js/pages/stats.js', '/js/placedonut.js', '/js/speedcompare.js', '/js/remotestart.js', '/js/pages/day.js', '/js/daydetail.js', '/js/car.js', '/js/routes.js', '/js/searchsheet.js', '/js/moresheet.js', '/js/inboxstate.js', '/js/homecards.js', '/js/fieldchecks.js', '/js/pages/fieldcheck.js', '/js/outingcard.js', '/js/workdaycard.js', '/js/homeparts.js', '/js/pick.js', '/js/pages/inbox.js', '/js/pages/reports.js', '/js/tripcard.js', '/js/offlinestore.js', '/js/offlinebar.js', '/js/offlinekey.js', '/js/tour.js', '/js/savingscard.js', '/js/plancard.js', '/js/eventrow.js', '/js/roadday.js', '/js/tourui.js', '/js/tripdistance.js', '/js/reportbuilder.js', '/js/reporteditor.js', '/js/categories.js', '/js/pages/categories.js', '/js/pages/chargingsettings.js', '/js/insightcard.js', '/js/pages/help.js', '/js/pages/cars.js', '/js/routeestimatebar.js', '/js/pages/fields.js', '/js/setup.js', '/js/pages/setup.js', '/js/errlog.js', '/js/chargesession.js', '/js/charts2.js', '/js/funwords.js', '/js/funcharts.js', '/js/infographic.js', '/js/insightsmore.js', '/js/insightsmore2.js', '/js/pages/analytics.js', '/js/pages/wrapped.js', '/js/replay.js', '/js/coachcard.js', '/js/navlayout.js', '/js/navedit.js', '/js/seasons.js', '/js/seasonfx.js', '/js/whatsnew.js', '/js/serviceform.js', '/js/servicecard.js', '/js/pages/service.js', '/js/vehiclepick.js', '/js/myprefs.js', '/js/fuelunits.js', '/js/carwords.js', '/js/tripplaces.js', '/js/insightsfuel.js', '/js/pages/fuel.js', '/js/pages/trophies.js', '/js/pages/fleet.js',
  '/js/pages/charging.js', '/js/pages/data.js', '/js/pages/settings.js', '/js/pages/rules.js'].map((p) => (p.startsWith('/js/') || p.startsWith('/css/') ? `/v${V}${p}` : p));

self.addEventListener('install', (e) => {
  // cache: 'reload' skips any copy kept by the browser or a CDN in front of the server, so a new version really downloads.
  e.waitUntil(caches.open(CACHE).then((c) => c.addAll(SHELL.map((u) => new Request(u, { cache: 'reload' })))).then(() => self.skipWaiting()));
});
self.addEventListener('activate', (e) => {
  e.waitUntil(caches.keys().then((ks) => Promise.all(ks.filter((k) => k !== CACHE && k !== TILES).map((k) => caches.delete(k)))).then(() => self.clients.claim()));
});
// Map tiles: the copy on the device is used at once and refreshed in the background, so a map seen before still draws with no
// signal. Only real pictures are kept, and the oldest go first.
async function tile(req) {
  const c = await caches.open(TILES);
  const hit = await c.match(req);
  const fresh = fetch(req).then(async (r) => {
    if (r.ok && (r.headers.get('content-type') || '').startsWith('image/')) {
      await c.put(req, r.clone());
      const ks = await c.keys();
      for (const k of ks.slice(0, Math.max(0, ks.length - MAX_TILES))) await c.delete(k);
    }
    return r;
  });
  if (hit) { fresh.catch(() => {}); return hit; }
  return fresh;
}

// App files: cache first. API and Ford link routes always go to the network
// (never cached: they carry private vehicle data).
self.addEventListener('fetch', (e) => {
  const u = new URL(e.request.url);
  if (e.request.method === 'GET' && u.origin === location.origin && u.pathname.startsWith('/api/tiles/')) { e.respondWith(tile(e.request)); return; }
  if (e.request.method !== 'GET' || u.origin !== location.origin || u.pathname.startsWith('/api/') || u.pathname.startsWith('/ford/') || u.pathname.startsWith('/s/')) return;
  e.respondWith(caches.match(e.request).then((hit) => hit || fetch(e.request)));
});

// Alerts: the server sends a short encrypted message. Show it, and open the right page when tapped.
self.addEventListener('push', (e) => {
  let d = {};
  try { d = e.data ? e.data.json() : {}; } catch (err) { d = { body: e.data ? e.data.text() : '' }; }
  e.waitUntil(self.registration.showNotification(d.title || 'CarTrak', {
    body: d.body || '', tag: d.tag || undefined, silent: d.quiet === '1', icon: '/icons/icon-192.png', badge: '/icons/badge-96.png', data: { url: d.url || '#/' },
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
