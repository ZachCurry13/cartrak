// Facts about how the page is laid out, as text, so a layout problem can be understood without a picture: what is too wide, what
// is too small to tap, what is cut off and what is hard to read.
const view = () => document.getElementById('view') || document.body;
const label = (el) => {
  const t = (el.getAttribute('aria-label') || el.innerText || el.value || el.placeholder || '').trim().replace(/\s+/g, ' ').slice(0, 40);
  const cls = el.className && typeof el.className === 'string' ? `.${el.className.trim().split(/\s+/)[0]}` : '';
  return `${el.tagName.toLowerCase()}${cls}${t ? ` "${t}"` : ''}`;
};
const visible = (el) => { const r = el.getBoundingClientRect(); return r.width > 0 && r.height > 0; };
const list = (title, items, max = 10) => (items.length ? [`${title} (${items.length}):`, ...items.slice(0, max).map((x) => `  - ${x}`)] : []);

const cx = Object.assign(document.createElement('canvas'), { width: 1, height: 1 }).getContext('2d', { willReadFrequently: true });
const rgba = (c) => { cx.clearRect(0, 0, 1, 1); cx.fillStyle = '#000'; cx.fillStyle = c; cx.fillRect(0, 0, 1, 1); const d = cx.getImageData(0, 0, 1, 1).data; return [d[0], d[1], d[2], d[3] / 255]; };
const lum = ([r, g, b]) => { const f = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; }; return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b); };
const over = (t, b) => [0, 1, 2].map((i) => t[i] * t[3] + b[i] * (1 - t[3])).concat([1]);
const bgOf = (el) => {
  const st = [];
  for (let e = el; e; e = e.parentElement) { const c = rgba(getComputedStyle(e).backgroundColor); if (c[3] > 0) st.push(c); if (c[3] >= 1) break; }
  let b = [0, 0, 0, 1];
  for (const c of st.reverse()) b = over(c, b);
  return b;
};

export function layoutFacts() {
  const sw = document.documentElement.scrollWidth;
  const out = [`Window ${innerWidth} x ${innerHeight}, page ${sw} wide${sw > innerWidth ? `: SCROLLS SIDEWAYS by ${sw - innerWidth} px` : ''}.`];
  try {
    const all = [...view().querySelectorAll('*')].filter(visible);
    out.push(...list('Wider than the window', all.filter((e) => e.getBoundingClientRect().right > innerWidth + 1).map((e) => `${label(e)} ends at ${Math.round(e.getBoundingClientRect().right)} px`)));
    out.push(...list('Cut off by their box', all.filter((e) => /hidden|clip/.test(getComputedStyle(e).overflowX) && e.scrollWidth > e.clientWidth + 2 && e.clientWidth > 0 && !e.classList.contains('vh')).map(label))); // screen-reader-only text is meant to be clipped
    const controls = [...view().querySelectorAll('button,.btn,select,input:not([type=checkbox]):not([type=radio]):not([type=hidden])')].filter(visible);
    out.push(...list('Controls under 38 px high', controls.filter((e) => e.getBoundingClientRect().height < 38).map((e) => `${label(e)} is ${Math.round(e.getBoundingClientRect().height)} px`)));
    const weak = [];
    for (const el of all) {
      if (![...el.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim())) continue;
      const cs = getComputedStyle(el), bg = bgOf(el), fg = over(rgba(cs.color), bg);
      const ratio = (Math.max(lum(fg), lum(bg)) + 0.05) / (Math.min(lum(fg), lum(bg)) + 0.05);
      const fs = parseFloat(cs.fontSize);
      if (ratio < (fs >= 24 || (fs >= 18.66 && parseInt(cs.fontWeight, 10) >= 700) ? 3 : 4.5)) weak.push(`${label(el)} has contrast ${ratio.toFixed(2)}`);
    }
    out.push(...list('Hard to read', weak));
  } catch (e) { out.push(`(some checks failed: ${e.message})`); }
  if (out.length === 1) out.push('Nothing wider than the window, cut off, too small or low in contrast.');
  return out.join('\n');
}
