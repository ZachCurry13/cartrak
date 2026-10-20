import { h } from './ui.js';
import { tourSteps } from './tour.js';

let live = null;

// Runs the guided tour: a dimmed screen with the part being explained left bright, and a sheet with Back, Next and Skip.
export function startTour(session) {
  if (live) return;
  const steps = tourSteps(session);
  let i = 0, token = 0, target = null;
  const spot = h('div', { class: 'tourspot', hidden: true });
  const dim = h('div', { class: 'tourdim' });
  const count = h('span', { class: 'muted' }), body = h('div', { class: 'tourbody' });
  const dots = h('div', { class: 'tourdots', 'aria-hidden': 'true' }, ...steps.map(() => h('i', {})));
  const back = h('button', { class: 'sec', type: 'button', onclick: () => go(i - 1) }, 'Back');
  const next = h('button', { type: 'button', onclick: () => (i === steps.length - 1 ? stop() : go(i + 1)) }, 'Next');
  const skip = h('button', { class: 'tourskip', type: 'button', onclick: () => stop() }, 'Skip guide');
  const sheet = h('div', { class: 'toursheet bottom', role: 'dialog', 'aria-label': 'Guide', 'aria-live': 'polite' },
    h('div', { class: 'tourtop' }, count, skip), body, dots, h('div', { class: 'tournav' }, back, next));
  const root = h('div', { class: 'tour' }, dim, spot, sheet);

  const place = () => {
    if (!target || !document.contains(target)) { spot.hidden = true; dim.classList.add('solid'); sheet.className = 'toursheet bottom'; return; }
    const r = target.getBoundingClientRect(), pad = 6;
    if (!r.width && !r.height) { target = null; place(); return; } // a hidden button has nothing to light up
    dim.classList.remove('solid');
    spot.hidden = false;
    Object.assign(spot.style, { left: `${r.left - pad}px`, top: `${r.top - pad}px`, width: `${r.width + pad * 2}px`, height: `${r.height + pad * 2}px` });
    sheet.className = `toursheet ${r.top > innerHeight * 0.5 ? 'top' : 'bottom'}`; // the sheet goes to the other half of the screen
  };
  const find = async (step, mine) => {
    for (let n = 0; n < 25 && mine === token; n++) {
      const e = step.target && step.target();
      if (e || !step.target) return e;
      await new Promise((r) => setTimeout(r, 120));
    }
    return null;
  };
  const go = async (n) => {
    if (n < 0 || n >= steps.length) return;
    i = n;
    const mine = ++token, step = steps[n];
    target = null;
    place();
    count.textContent = `Step ${n + 1} of ${steps.length}`;
    body.replaceChildren(h('h2', {}, `${step.icon} ${step.title}`), h('p', {}, step.text));
    [...dots.children].forEach((d, k) => d.classList.toggle('on', k === n));
    back.disabled = n === 0;
    next.textContent = n === steps.length - 1 ? 'Done' : 'Next';
    if (step.route && location.hash.replace(/^#?\/?/, '#/') !== step.route) location.hash = step.route;
    const e = await find(step, mine);
    if (mine !== token) return;
    target = e;
    if (e) e.scrollIntoView({ block: 'center', behavior: 'instant' });
    place();
    next.focus({ preventScroll: true });
  };
  const key = (e) => { if (e.key === 'Escape') stop(); else if (e.key === 'ArrowRight' && i < steps.length - 1) go(i + 1); else if (e.key === 'ArrowLeft') go(i - 1); };
  const stop = () => {
    token++;
    root.remove();
    removeEventListener('resize', place); removeEventListener('scroll', place, true); document.removeEventListener('keydown', key);
    try { localStorage.setItem('tourDone', '1'); } catch (e) { /* storage blocked */ }
    live = null;
  };
  addEventListener('resize', place);
  addEventListener('scroll', place, true);
  document.addEventListener('keydown', key);
  document.body.append(root);
  live = root;
  go(0);
}
