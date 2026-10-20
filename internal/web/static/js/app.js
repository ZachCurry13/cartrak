import { h, ago, applyTheme } from './ui.js';
applyTheme();
import { api, AuthError } from './api.js';
import { loginPage } from './pages/login.js';
import { startRefresh, effective } from './mode.js';
import { cached } from './offline.js';
import { startTour } from './tourui.js';
import { currentCar, setCar } from './car.js';
import { routes } from './routes.js';
import { loadCategories } from './categories.js';
import { refreshBell } from './inboxstate.js';
import { openSearch } from './searchsheet.js';
import { openMore } from './moresheet.js';


const view = document.getElementById('view');
document.querySelector('.skip').addEventListener('click', (e) => { e.preventDefault(); view.focus(); });
const tabs = document.getElementById('tabs');
export const session = { me: null, about: null, vehicle: null, installPrompt: null, cars: [] };
let stopRefresh = null;

// Chrome/Android offers a one-tap install; keep the event for the Getting started page.
addEventListener('beforeinstallprompt', (e) => { e.preventDefault(); session.installPrompt = e; });

const store = {
  get: (k) => { try { return localStorage.getItem(k); } catch (e) { return null; } },
  set: (k, v) => { try { localStorage.setItem(k, v); } catch (e) { /* ignore */ } },
};

let loadedVersion = ''; // the version this page started with: a phone app can stay open for days
async function loadAbout() {
  session.about = await cached('about', () => api('/api/about'));
  if (!loadedVersion) loadedVersion = session.about.version;
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

// The cars this person can open. The one being looked at is remembered; if it is gone, the first is used. With more than one car
// a switcher appears in the header, and an administrator looking at someone else's car is told so.
async function loadCars() {
  const d = await api('/api/cars');
  session.cars = d.cars;
  const ids = d.cars.map((c) => String(c.id));
  if (ids.length && !ids.includes(currentCar())) setCar(ids[0]);
  if (ids.length) await loadCategories();
  const sel = document.getElementById('carsel');
  sel.hidden = d.cars.length < 2;
  sel.replaceChildren(...d.cars.map((c) => h('option', { value: String(c.id), selected: String(c.id) === currentCar() }, c.name)));
  sel.onchange = () => { setCar(sel.value); location.hash = '#/'; location.reload(); };
  const here = d.cars.find((c) => String(c.id) === currentCar());
  // The role that counts is the one for the car on screen: an owner manages their car, a viewer reads it. The role for the
  // whole installation (people, remote access) stays as siteRole.
  session.me.siteRole = session.me.siteRole || session.me.role;
  session.me.role = session.me.siteRole === 'admin' || (here && here.level === 'owner') ? 'admin' : 'viewer';
  for (const id of ['srch', 'bell', 'help']) document.getElementById(id).hidden = !d.cars.length; // a person with no car yet still sees Help
  document.getElementById('srch').onclick = () => openSearch(session.me.role === 'admin');
  document.getElementById('help').onclick = (e) => { e.preventDefault(); startTour(session); }; // the ? opens the guided tour; the Help page is in More
  refreshBell();
  const note = document.getElementById('carnote');
  note.hidden = !(here && here.theirs);
  note.textContent = here && here.theirs ? `You are looking at ${here.name} as an administrator. It is not your car, and its owner can see that you looked.` : '';
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
  const tab = ['', 'trips', 'live', 'charging', 'health', 'data'].includes(name) ? name : name === 'stats' ? 'charging' : name === 'fields' ? 'data' : name === 'trip' || name === 'history' || name === 'day' || name === 'journey' || name === 'review' ? 'trips' : 'settings';
  tabs.querySelectorAll('a').forEach((a) => a.classList.toggle('on', a.getAttribute('href') === `#/${tab}` || (tab === '' && a.getAttribute('href') === '#/')));
  try {
    if (!session.me) {
      session.me = await cached('me', () => api('/api/me'));
      await loadCars();
      if (!session.cars.length) { // a person who may bring a car but has none yet
        tabs.hidden = true;
        if (name !== 'cars') { location.hash = '#/cars'; return; }
      } else {
        await loadAbout();
        await loadVehicle();
        if (await firstRunTour()) return; // hashchange re-renders onto the tour
      }
    }
    tabs.hidden = !session.cars.length;
    for (const a of tabs.querySelectorAll('a[href="#/health"], a[href="#/data"]')) a.hidden = session.me.role !== 'admin'; // viewers get the simple set of tabs
    const mod = await (routes[name] || routes[''])();
    view.replaceChildren();
    const refresh = await mod.default(view, args, session);
    const head = view.querySelector('h2');
    document.title = head && head.textContent ? `${head.textContent} · CarTrak` : 'CarTrak';
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
    // The time that matters is when the car last reported. When it is much older than CarTrak's last check with
    // Ford, the car is asleep or out of signal, which is normal and not a fault.
    const el = document.getElementById('sync');
    const car = st.carLastReport || st.poller.lastOk;
    const stale = Date.now() / 1000 - st.poller.lastOk > 900;
    el.textContent = st.poller.linked ? `car reported ${ago(car)}${stale ? ' · Ford not answering' : ''}` : 'not linked';
    el.title = st.poller.linked ? `CarTrak last checked Ford ${ago(st.poller.lastOk)}. The car sends data when it is awake, so an older time here usually means it is parked and asleep.` : '';
    loadVehicle();
    refreshBell();
  } catch (e) { /* offline */ }
}

// The More tab slides a sheet up over the page, as the app on a phone usually does; the whole More page is one row down in it.
tabs.querySelector('a[href="#/settings"]').addEventListener('click', (e) => { if (session.me) { e.preventDefault(); openMore(session); } });
addEventListener('hashchange', render);
setInterval(() => { if (!document.hidden) syncLabel(); }, effective() === 'saver' ? 300000 : 30000);
// Ford may send the browser back to the site root (the redirect address registered on the Ford app may have
// no /ford/callback path). If the address carries a sign-in answer, hand it to the callback page to finish.
const fordReturn = /[?&](code|error)=/.test(location.search) || /[#&](code|error)=/.test(location.hash);
if (fordReturn) location.replace('/ford/callback' + location.search + location.hash);
else render().then(syncLabel);

function showUpdateBar() {
  if (document.getElementById('upd')) return;
  document.body.append(h('div', { id: 'upd', class: 'updbar', role: 'status' }, 'A new version of CarTrak is ready. ',
    h('button', { onclick: () => location.reload() }, 'Reload')));
}

if ('serviceWorker' in navigator) {
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
      if (loadedVersion && a.version && a.version !== loadedVersion) showUpdateBar();
    } catch (e) { /* offline or signed out: nothing to compare */ }
  });
  navigator.serviceWorker.register('/sw.js').catch(() => {});
}
