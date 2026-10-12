import { h, ago, applyTheme } from './ui.js';
applyTheme();
import { api, AuthError } from './api.js';
import { loginPage } from './pages/login.js';

const routes = {
  '': () => import('./pages/dashboard.js'),
  trips: () => import('./pages/trips.js'),
  trip: () => import('./pages/tripdetail.js'),
  charging: () => import('./pages/charging.js'),
  data: () => import('./pages/data.js'),
  settings: () => import('./pages/settings.js'),
  rules: () => import('./pages/rules.js'),
};

const view = document.getElementById('view');
const tabs = document.getElementById('tabs');
export const session = { me: null };
let stopTimer = null;

async function render() {
  clearInterval(stopTimer);
  const [name, ...args] = (location.hash.replace(/^#\/?/, '') || '').split('/');
  const tab = ['', 'trips', 'charging', 'data'].includes(name) ? name : name === 'trip' ? 'trips' : 'settings';
  tabs.querySelectorAll('a').forEach((a) => a.classList.toggle('on', a.getAttribute('href') === `#/${tab}` || (tab === '' && a.getAttribute('href') === '#/')));
  try {
    session.me = session.me || await api('/api/me');
    tabs.hidden = false;
    const mod = await (routes[name] || routes[''])();
    view.replaceChildren();
    const refresh = await mod.default(view, args, session);
    if (typeof refresh === 'function') stopTimer = setInterval(refresh, 30000);
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
  } catch (e) { /* offline */ }
}

addEventListener('hashchange', render);
setInterval(syncLabel, 30000);
render().then(syncLabel);

if ('serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js').catch(() => {});
