// What a tap feels like: a Material ripple that spreads from the finger, and, where the phone allows it (Android), a short
// vibration when a switch is flipped or a button pressed. iPhone does not let web pages vibrate; there the ripple is all there is.
const KEY = 'cartrak.haptics';

export const getHaptics = () => { try { return localStorage.getItem(KEY) !== 'off'; } catch (e) { return true; } };
export const setHaptics = (on) => { try { localStorage.setItem(KEY, on ? 'on' : 'off'); } catch (e) { /* no storage: it stays on */ } };

const calm = () => matchMedia('(prefers-reduced-motion: reduce)').matches;
export const buzz = (pattern) => { if (typeof navigator.vibrate === 'function' && getHaptics() && !calm()) navigator.vibrate(pattern); };

const PRESS = 'button,.btn,nav#tabs a,.mtile,.mrow,.item,.ibrow,a.hcard,a.tcard,.linkrow a,.atabs a,.chipbtn,details>summary,.msearch,.cardlink';
const FIRM = 'button.danger,.btn.danger';

export function startTouchFeedback() {
  let lit = null;
  const unlight = () => { if (lit) lit.classList.remove('rip'); lit = null; };
  document.addEventListener('pointerdown', (e) => {
    unlight();
    const el = e.target instanceof Element && e.target.closest(PRESS);
    if (!el || el.disabled) return;
    const r = el.getBoundingClientRect();
    el.style.setProperty('--rx', `${e.clientX - r.left}px`);
    el.style.setProperty('--ry', `${e.clientY - r.top}px`);
    void el.offsetWidth; // restart the animation if it is still running
    el.classList.add('rip');
    lit = el;
    setTimeout(() => { if (lit === el) unlight(); }, 480);
  }, { passive: true });
  // a finger that starts scrolling is not a press
  document.addEventListener('pointercancel', unlight, { passive: true });
  document.addEventListener('click', (e) => {
    const t = e.target instanceof Element ? e.target : null;
    if (!t || (t instanceof HTMLInputElement && t.type === 'checkbox')) return; // switches buzz on change, below
    if (t.closest(FIRM)) buzz(24);
    else if (t.closest('button,.btn,nav#tabs a')) buzz(8);
  }, true);
  document.addEventListener('change', (e) => {
    const t = e.target;
    if (t instanceof HTMLInputElement && (t.type === 'checkbox' || t.type === 'radio')) buzz(t.checked ? 14 : 8);
    else if (t instanceof HTMLSelectElement) buzz(8);
  }, true);
}
