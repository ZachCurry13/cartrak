// A bottom sheet follows a finger (or a mouse) pulled down on its handle or its title, and closes when pulled far or flicked.
// A short pull springs back. Taps on the sheet's buttons still work: dragging starts only after the pointer has moved.
const START = 8; // px of movement before it counts as a drag, not a tap
const FAR = 90; // px pulled down that closes it
const FLICK = 0.6; // px per ms that closes it however short the pull
const SCRIM = 0.5; // the backdrop's opacity at rest, as set in look.css

export function dragToClose(sheet, bg, close) {
  let y0 = 0, t0 = 0, last = 0, lastT = 0, id = null, dragging = false;
  const move = (dy) => {
    sheet.style.transform = `translateY(${Math.max(0, dy)}px)`;
    bg.style.background = `rgba(0,0,0,${SCRIM * Math.max(0, 1 - dy / sheet.offsetHeight)})`;
  };
  const settle = (to, then) => {
    sheet.style.transition = 'transform .2s cubic-bezier(.2,0,0,1)';
    bg.style.transition = 'background .2s';
    sheet.style.transform = to;
    bg.style.background = then ? 'rgba(0,0,0,0)' : `rgba(0,0,0,${SCRIM})`;
    if (then) setTimeout(then, 190);
  };
  sheet.addEventListener('pointerdown', (e) => {
    if (e.button > 0 || !e.target.closest('.mhandle,.mhead')) return;
    id = e.pointerId; y0 = last = e.clientY; t0 = lastT = e.timeStamp; dragging = false;
  });
  sheet.addEventListener('pointermove', (e) => {
    if (e.pointerId !== id) return;
    const dy = e.clientY - y0;
    if (!dragging) {
      if (Math.abs(dy) < START) return;
      dragging = true;
      sheet.setPointerCapture(id);
      sheet.style.animation = 'none';
      sheet.style.transition = bg.style.transition = 'none';
    }
    last = e.clientY; lastT = e.timeStamp;
    move(dy);
  });
  const end = (e) => {
    if (e.pointerId !== id) return;
    id = null;
    if (!dragging) return;
    dragging = false;
    const dy = last - y0, speed = (last - y0) / Math.max(1, lastT - t0);
    if (e.type !== 'pointercancel' && (dy > FAR || speed > FLICK)) settle(`translateY(${sheet.offsetHeight}px)`, close);
    else settle('none');
  };
  sheet.addEventListener('pointerup', end);
  sheet.addEventListener('pointercancel', end);
  // A drag ends in a click on whatever is under the finger; swallow that one.
  sheet.addEventListener('click', (e) => { if (Date.now() - lastT < 60 && Math.abs(last - y0) > START) { e.preventDefault(); e.stopPropagation(); } }, true);
}
