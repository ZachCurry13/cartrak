// The floating action button: the one thing to do on a page, in reach of a thumb. It takes you to the field that starts it.
import { h } from './ui.js';
import { icon } from './navlayout.js';

const PLUS = 'M12 5v14M5 12h14';

// target: the element to scroll to and focus, or a function that finds it (a page may redraw its form).
export function fab(label, target, path = PLUS) {
  return h('button', {
    class: 'fab', type: 'button',
    onclick: () => {
      const el = typeof target === 'function' ? target() : target;
      if (!el) return;
      el.scrollIntoView({ behavior: matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth', block: 'center' });
      setTimeout(() => el.focus && el.focus({ preventScroll: true }), 350);
    },
  }, icon(path), h('span', {}, label));
}
