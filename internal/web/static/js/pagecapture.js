// What a bug report records about the page: the screen it was on, its visible text, and a picture of it. The picture is drawn from the
// page itself (no library): the page's own styles and markup go into an SVG, the browser draws that into a canvas, and the canvas
// becomes a PNG. Anything that cannot be drawn this way (a map's tiles are copied in, a sheet or a toast is left out) never stops the
// report: the text and the screen details are always there.

const MAX_HEIGHT = 8000; // px; a very long page is cut here

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

const svgNS = 'http://www.w3.org/2000/svg';

// All of the page's own style rules, as text.
function pageCss() {
  let css = '';
  for (const sheet of document.styleSheets) {
    try { css += [...sheet.cssRules].map((r) => r.cssText).join('\n') + '\n'; } catch (e) { /* a sheet from another address: skipped */ }
  }
  return css;
}

// An image on the page as a data address, so it still draws inside the SVG (which may not fetch anything).
function inlineImages(live, copy) {
  for (let i = 0; i < live.length; i++) {
    const src = live[i];
    try {
      if (src.tagName === 'IMG' && src.complete && src.naturalWidth) {
        const c = document.createElement('canvas');
        c.width = src.naturalWidth; c.height = src.naturalHeight;
        c.getContext('2d').drawImage(src, 0, 0);
        copy[i].setAttribute('src', c.toDataURL('image/png'));
        copy[i].removeAttribute('srcset');
      } else if (src.tagName === 'CANVAS') {
        const img = document.createElement('img');
        img.setAttribute('src', src.toDataURL('image/png'));
        img.setAttribute('style', `width:${src.clientWidth}px;height:${src.clientHeight}px;display:block`);
        copy[i].replaceWith(img);
      }
    } catch (e) { /* tainted or empty: left out */ }
  }
}

// A PNG of the page as a data address, or null when the browser cannot draw it.
export async function screenshot() {
  try {
    const width = document.documentElement.clientWidth;
    const height = Math.min(document.documentElement.scrollHeight, MAX_HEIGHT);
    const body = document.body.cloneNode(true);
    const live = [...document.body.querySelectorAll('img,canvas')], copy = [...body.querySelectorAll('img,canvas')];
    inlineImages(live, copy);
    for (const el of body.querySelectorAll('script,.sheetbg,#toast,.updbar,iframe,video,audio')) el.remove();
    for (const el of body.querySelectorAll('input[type=password]')) el.value = '';
    const xml = new XMLSerializer().serializeToString(body);
    const root = document.documentElement;
    const attrs = ['data-theme', 'data-season', 'data-page'].map((a) => (root.hasAttribute(a) ? ` ${a}="${root.getAttribute(a).replace(/"/g, '')}"` : '')).join('');
    const svg = `<svg xmlns="${svgNS}" width="${width}" height="${height}"${attrs}><foreignObject x="0" y="0" width="${width}" height="${height}"><style><![CDATA[${pageCss().replace(/]]>/g, '] ]>')}]]></style>${xml}</foreignObject></svg>`;
    const img = new Image();
    img.decoding = 'sync';
    await new Promise((ok, no) => { img.onload = ok; img.onerror = no; img.src = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`; });
    const canvas = document.createElement('canvas');
    canvas.width = width; canvas.height = height;
    const ctx = canvas.getContext('2d');
    ctx.fillStyle = getComputedStyle(document.body).backgroundColor || '#000';
    ctx.fillRect(0, 0, width, height);
    ctx.drawImage(img, 0, 0);
    return canvas.toDataURL('image/png');
  } catch (e) {
    return null;
  }
}

