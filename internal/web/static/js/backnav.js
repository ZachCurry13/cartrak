// Back that goes where you came from. A page's own "← More" link used to be a normal link: it named a page chosen by whoever wrote
// the page (Stats always said "← Charging", even when you opened it from Trips) and following it added a step to the history, so
// the phone's Back button then went somewhere that looked random. Now the app keeps a note of the pages it showed, in order, and the
// "←" link at the top of a page names the page before it and goes there the same way the Back button does. Opened straight from a
// bookmark or a notification there is no page before, and the link keeps the name and the place its page gave it.
const KEY = 'cartrak.nav';
const TABS = { '#/': 'Home', '#/trips': 'Trips', '#/live': 'Live', '#/charging': 'Charging', '#/health': 'Health', '#/data': 'Data', '#/settings': 'More' };
let idx = 0, stack = [], titles = {}, replacing = false;

const load = () => { try { const v = JSON.parse(sessionStorage.getItem(KEY)); if (v) { stack = v.stack || []; titles = v.titles || {}; } } catch (e) { /* no memory: start fresh */ } };
const save = () => { try { sessionStorage.setItem(KEY, JSON.stringify({ stack, titles })); } catch (e) { /* ignore */ } };
const here = () => location.hash || '#/';
const name = (hash) => TABS[hash] || titles[hash] || '';

// Called once as the app starts, and again by every change of page.
function arrive() {
  const s = history.state;
  if (s && typeof s.idx === 'number') idx = s.idx; // Back or Forward: the entry knows its own place
  else if (replacing) replacing = false; // a replaced entry keeps its place
  else idx = stack.length ? idx + 1 : 0; // a new page on top of the history
  if (!(s && typeof s.idx === 'number' && stack[idx] === here())) stack = stack.slice(0, idx);
  stack[idx] = here();
  try { history.replaceState({ ...(s || {}), idx }, ''); } catch (e) { /* ignore */ }
  save();
}

export function startBackNav() {
  load();
  const s = history.state;
  idx = s && typeof s.idx === 'number' ? s.idx : 0;
  if (!stack.length || stack[idx] !== here()) { stack = []; idx = 0; }
  arrive();
  addEventListener('hashchange', (e) => { if (e.oldURL && e.newURL && e.oldURL !== e.newURL) arrive(); }); // the app's own refresh events carry no addresses
  document.addEventListener('click', (e) => {
    if (e.defaultPrevented || e.button || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    const a = e.target.closest && e.target.closest('a[href^="#/"]');
    if (!a || !/^←/.test(a.textContent.trim()) || idx < 1) return;
    e.preventDefault();
    history.back();
  });
}

// After a page has drawn: remember its name, and make its "←" link say where it really goes.
export function noteRendered(title, root) {
  if (title) { titles[here()] = title; save(); }
  const before = idx > 0 ? name(stack[idx - 1]) : '';
  if (!before) return;
  for (const a of root.querySelectorAll('a[href^="#/"]')) if (/^←/.test(a.textContent.trim())) a.textContent = `← ${before}`;
}

// A change of page that should not be a step in the history: a redirect, or the same page showing another day.
export const goReplace = (hash) => {
  const to = hash.startsWith('#') ? hash : `#/${hash}`;
  if (to === here()) return;
  replacing = true;
  location.replace(to);
};
