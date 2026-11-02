// Behavior every page shares, so no page has to remember it: a number field selects its text when tapped and brings up the right
// keypad, and a long help paragraph is folded to two lines with a More button, so the answer is not buried under the explanation.
// It watches the page (pages draw in pieces as their answers arrive) and tidies what is new.
const LONG = 170;

function tidy(root) {
  for (const i of root.querySelectorAll('input[type=number]:not([inputmode])')) {
    i.setAttribute('inputmode', /\./.test(i.step || '') || i.step === 'any' ? 'decimal' : 'numeric');
  }
  for (const p of root.querySelectorAll('p.muted')) {
    if (p.dataset.tidy || p.closest('.banner, .topbar, .sheet, label, .seg') || p.textContent.length <= LONG || p.querySelector('a, button, input, select')) continue;
    p.dataset.tidy = '1';
    p.classList.add('clamp');
    const more = document.createElement('button');
    more.type = 'button';
    more.className = 'morebtn';
    more.textContent = 'More';
    more.setAttribute('aria-expanded', 'false');
    more.addEventListener('click', () => {
      const open = p.classList.toggle('open');
      more.textContent = open ? 'Less' : 'More';
      more.setAttribute('aria-expanded', String(open));
    });
    p.after(more);
  }
}

export function startConsistency(view) {
  document.addEventListener('focusin', (e) => { const t = e.target; if (t && t.matches && t.matches('input[type=number]') && t.select) t.select(); });
  let timer = null;
  new MutationObserver(() => { clearTimeout(timer); timer = setTimeout(() => tidy(view), 200); }).observe(view, { childList: true, subtree: true });
}
