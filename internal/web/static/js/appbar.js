// The top bar of a page: it lifts onto a tonal surface once the page has scrolled, leaves the way when you scroll down to read and
// comes back as soon as you scroll up, as Material's top app bar does.
export function startAppBar() {
  const bar = document.getElementById('top');
  if (!bar) return;
  let last = scrollY, queued = false;
  const update = () => {
    queued = false;
    const y = scrollY;
    bar.classList.toggle('raised', y > 4);
    if (Math.abs(y - last) > 8) { bar.classList.toggle('away', y > last && y > 90); last = y; }
  };
  addEventListener('scroll', () => { if (!queued) { queued = true; requestAnimationFrame(update); } }, { passive: true });
  addEventListener('hashchange', () => { bar.classList.remove('away'); last = scrollY; });
}
