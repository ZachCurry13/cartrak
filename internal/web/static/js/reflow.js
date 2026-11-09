// When the window changes size and Home's cards (or a list in columns) move to another number of columns, each one glides to its
// new place instead of jumping. While the window is simply being dragged, the columns keep their count and nothing animates.
const MS = 260;
const calm = () => matchMedia('(prefers-reduced-motion: reduce)').matches;

function snap(el) {
  const out = new Map();
  for (const c of el.children) { const r = c.getBoundingClientRect(); out.set(c, { left: r.left, top: r.top + scrollY }); } // top counted from the page, so scrolling is not mistaken for a move
  return out;
}
const columns = (m) => new Set([...m.values()].map((r) => Math.round(r.left / 8))).size;

function watch(el) {
  let prev = snap(el), cols = columns(prev);
  const check = () => { // the observer runs after layout, once per frame, so the new places are already known
    const now = snap(el), c = columns(now);
    if (c !== cols && !calm()) {
      for (const [node, to] of now) {
        const from = prev.get(node);
        if (!from || !node.animate) continue;
        const dx = from.left - to.left, dy = from.top - to.top;
        if (Math.abs(dx) < 2 && Math.abs(dy) < 2) continue;
        node.animate([{ transform: `translate(${dx}px,${dy}px)` }, { transform: 'none' }], { duration: MS, easing: 'cubic-bezier(.2,0,0,1)' });
      }
    }
    prev = now; cols = c;
  };
  new MutationObserver(() => { prev = snap(el); cols = columns(prev); }).observe(el, { childList: true });
  new ResizeObserver(check).observe(el);
}

// Called once for the page area: every column layout that appears in it is watched.
export function mountReflow(view) {
  const attach = () => { for (const el of view.querySelectorAll('.dash,.cols')) if (!el.dataset.reflow) { el.dataset.reflow = '1'; watch(el); } };
  new MutationObserver(attach).observe(view, { childList: true, subtree: true });
  attach();
}
