import { h, ago, applyTheme } from './ui.js';
applyTheme();
import { api, AuthError } from './api.js';
import { loginPage } from './pages/login.js';
import { startRefresh, effective } from './mode.js';
import { cached } from './offline.js';

const routes = {
  '': () => import('./pages/dashboard.js'),
  trips: () => import('./pages/trips.js'),
  trip: () => import('./pages/tripdetail.js'),
  journey: () => import('./pages/journey.js'),
  charging: () => import('./pages/charging.js'),
  health: () => import('./pages/vehiclehealth.js'),
  data: () => import('./pages/data.js'),
  settings: () => import('./pages/settings.js'),
  rules: () => import('./pages/rules.js'),
  start: () => import('./pages/start.js'),
  changelog: () => import('./pages/changelog.js'),
  admin: () => import('./pages/admin.js'),
  connect: () => import('./pages/connect.js'),
  remote: () => import('./pages/remote.js'),
  ford: () => import('./pages/ford.js'),
  display: () => import('./pages/display.js'),
  people: () => import('./pages/people.js'),
  mileage: () => import('./pages/mileage.js'),
  plan: () => import('./pages/plan.js'),
  maintenance: () => import('./pages/maintenance.js'),
  places: () => import('./pages/places.js'),
  insights: () => import('./pages/insights.js'),
  review: () => import('./pages/review.js'),
  security: () => import('./pages/security.js'),
  backup: () => import('./pages/backup.js'),
  notifications: () => import('./pages/notifications.js'),
  phone: () => import('./pages/phone.js'),
  about: () => import('./pages/about.js'),
  status: () => import('./pages/status.js'),
  live: () => import('./pages/live.js'),
};

const view = document.getElementById('view');
const tabs = document.getElementById('tabs');
export const session = { me: null, about: null, vehicle: null, installPrompt: null };
let stopRefresh = null;

// Chrome/Android offers a one-tap install; keep the event for the Getting started page.
addEventListener('beforeinstallprompt', (e) => { e.preventDefault(); session.installPrompt = e; });

const store = {
  get: (k) => { try { return localStorage.getItem(k); } catch (e) { return null; } },
  set: (k, v) => { try { localStorage.setItem(k, v); } catch (e) { /* ignore */ } },
};

async function loadAbout() {
  session.about = await cached('about', () => api('/api/about'));
  const ver = document.getElementById('ver');
  ver.textContent = `v${session.about.version}`;
  ver.classList.toggle('update', !!session.about.update.available);
  ver.title = session.about.update.available ? `Update available: v${session.about.update.latest}` : 'Version';
  // Remember which version this device last saw, to announce "What's new" after an update.
  session.about.justUpdated = !!store.get('seenVersion') && store.get('seenVersion') !== session.about.version;
  if (!store.get('seenVersion')) store.set('seenVersion', session.about.version);
}

// The kind of car (electric, hybrid, gas) decides which tabs make sense: a gas car has no Charging tab.
async function loadVehicle() {
  try { session.vehicle = await api('/api/vehicle'); } catch (e) { /* keep what we had */ }
  const charging = tabs.querySelector('a[href="#/charging"]');
  if (charging) charging.hidden = !!session.vehicle && !session.vehicle.plug;
}

async function firstRunTour() {
  if (session.me.role !== 'admin' || store.get('welcomed')) return false;
  store.set('welcomed', '1');
  const ob = await api('/api/onboarding');
  if (ob.dismissed) return false;
  location.hash = '#/start';
  return true;
}

async function render() {
  if (stopRefresh) { stopRefresh(); stopRefresh = null; }
  const [name, ...args] = (location.hash.replace(/^#\/?/, '') || '').split('/');
  const tab = ['', 'trips', 'live', 'charging', 'health', 'data'].includes(name) ? name : name === 'trip' || name === 'journey' || name === 'review' ? 'trips' : 'settings';
  tabs.querySelectorAll('a').forEach((a) => a.classList.toggle('on', a.getAttribute('href') === `#/${tab}` || (tab === '' && a.getAttribute('href') === '#/')));
  try {
    if (!session.me) {
      session.me = await cached('me', () => api('/api/me'));
      await loadAbout();
      await loadVehicle();
      if (await firstRunTour()) return; // hashchange re-renders onto the tour
    }
    tabs.hidden = false;
    const mod = await (routes[name] || routes[''])();
    view.replaceChildren();
    const refresh = await mod.default(view, args, session);
    if (typeof refresh === 'function') stopRefresh = startRefresh(refresh); // schedule depends on this device's data mode
    else if (refresh && refresh.cleanup) stopRefresh = refresh.cleanup;
  } catch (e) {
    if (e instanceof AuthError) return showLogin();
    view.replaceChildren(h('div', { class: 'banner' }, `Something went wrong: ${e.message}`));
  }
}

async function showLogin() {
  session.me = null;
  tabs.hidden = true;
  const s = await api('/api/setup', { noAuthRedirect: true }).catch(() => ({}));
  view.replaceChildren(loginPage(s.needsSetup, () => { location.hash = '#/'; render(); }));
}

async function syncLabel() {
  if (!session.me) return;
  try {
    const st = await api('/api/status');
    document.getElementById('sync').textContent = st.poller.linked ? `updated ${ago(st.poller.lastOk)}` : 'not linked';
    loadVehicle();
  } catch (e) { /* offline */ }
}

addEventListener('hashchange', render);
setInterval(() => { if (!document.hidden) syncLabel(); }, effective() === 'saver' ? 300000 : 30000);
// Ford may send the browser back to the site root (the redirect address registered on the Ford app may have
// no /ford/callback path). If the address carries a sign-in answer, hand it to the callback page to finish.
const fordReturn = /[?&](code|error)=/.test(location.search) || /[#&](code|error)=/.test(location.hash);
if (fordReturn) location.replace('/ford/callback' + location.search + location.hash);
else render().then(syncLabel);

if ('serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js').catch(() => {});
