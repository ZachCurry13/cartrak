// What a bug report records about the page: the screen it was on (size, phone or computer, touch or mouse, theme, browser) and the
// words visible on it. Layout facts are in layoutfacts.js.

// "phone 411 x 890 (screen 411 x 914, 2.6x), touch, installed app, dark, portrait; Chrome on Android".
export function deviceInfo() {
  const touch = matchMedia('(pointer: coarse)').matches;
  const installed = matchMedia('(display-mode: standalone)').matches || navigator.standalone === true;
  const w = innerWidth;
  const kind = touch ? (w < 600 ? 'phone' : 'tablet') : 'computer';
  const theme = document.documentElement.dataset.theme || (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark (system)' : 'light (system)');
  const orient = innerHeight >= innerWidth ? 'portrait' : 'landscape';
  const ua = navigator.userAgent.replace(/\s+/g, ' ');
  return `${kind} ${w} x ${innerHeight} (screen ${screen.width} x ${screen.height}, ${devicePixelRatio}x), ${touch ? 'touch' : 'mouse'}, ${installed ? 'installed app' : 'browser tab'}, ${theme}, ${orient}; ${ua}`;
}

// The words on the page, in reading order, trimmed.
export function pageText() {
  const view = document.getElementById('view');
  return ((view && view.innerText) || '').replace(/\n{3,}/g, '\n\n').trim().slice(0, 40000);
}

