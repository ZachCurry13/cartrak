// "Saved": one short message at the bottom of the screen, the same for every change, instead of each page finding its own words.
let el = null, timer = null;

export function toast(text) {
  if (!el) {
    el = document.createElement('div');
    el.id = 'toast';
    el.setAttribute('role', 'status');
    el.setAttribute('aria-live', 'polite');
    document.body.append(el);
  }
  el.textContent = text;
  el.classList.add('on');
  clearTimeout(timer);
  timer = setTimeout(() => el.classList.remove('on'), 2200);
}
