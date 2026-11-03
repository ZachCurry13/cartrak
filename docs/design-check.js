// Paste into the browser console on a running CarTrak (any page). Visits every page in dark and light and reports:
//   - the page scrolling sideways,
//   - buttons, links-as-buttons and fields shorter than 38 px,
//   - text whose contrast is under 4.5:1 (3:1 for large text).
// Set the window to the phone size you want first (360 or 411 wide). It takes about a minute; the result is in window.designCheck.
(async () => {
  const routes = ['', 'trips', 'charging', 'analytics', 'settings', 'about', 'admin', 'backup', 'cars', 'categories', 'changelog', 'chargingsettings', 'compare', 'data', 'datacheck', 'debug', 'developer', 'display', 'fieldcheck', 'fields', 'fleet', 'ford', 'fuel', 'guide', 'help', 'history', 'inbox', 'insights', 'live', 'maintenance', 'mileage', 'notifications', 'parkedwatch', 'people', 'phone', 'places', 'plan', 'planfuel', 'remote', 'repair', 'reports', 'review', 'rules', 'security', 'service', 'status', 'stops', 'timeuse', 'trophies', 'vehiclehealth', 'year', 'stats'];
  const cx = Object.assign(document.createElement('canvas'), { width: 1, height: 1 }).getContext('2d', { willReadFrequently: true });
  const rgba = (c) => { cx.clearRect(0, 0, 1, 1); cx.fillStyle = '#000'; cx.fillStyle = c; cx.fillRect(0, 0, 1, 1); const d = cx.getImageData(0, 0, 1, 1).data; return [d[0], d[1], d[2], d[3] / 255]; };
  const lum = ([r, g, b]) => { const f = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; }; return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b); };
  const over = (t, b) => [0, 1, 2].map((i) => t[i] * t[3] + b[i] * (1 - t[3])).concat([1]);
  const bgOf = (el) => { const st = []; for (let e = el; e; e = e.parentElement) { const c = rgba(getComputedStyle(e).backgroundColor); if (c[3] > 0) st.push(c); if (c[3] >= 1) break; } let b = [0, 0, 0, 1]; for (const c of st.reverse()) b = over(c, b); return b; };
  const out = [];
  for (const theme of ['dark', 'light']) {
    document.documentElement.dataset.theme = theme;
    for (const p of routes) {
      location.hash = '#/' + p;
      await new Promise((r) => setTimeout(r, 900)); // let the page and any colour transition settle
      const v = document.getElementById('view');
      if (document.documentElement.scrollWidth > innerWidth) out.push(`${theme} ${p || 'home'}: scrolls sideways`);
      const small = [...v.querySelectorAll('button,.btn,select,input:not([type=checkbox]):not([type=radio]):not([type=hidden]):not([type=range])')].filter((e) => { const r = e.getBoundingClientRect(); return r.width && r.height && r.height < 38; }).length;
      if (small) out.push(`${theme} ${p || 'home'}: ${small} controls under 38 px`);
      for (const el of v.querySelectorAll('*')) {
        if (![...el.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim())) continue;
        const r = el.getBoundingClientRect();
        if (!r.width || !r.height) continue;
        const cs = getComputedStyle(el);
        const bg = bgOf(el), fg = over(rgba(cs.color), bg);
        const ratio = (Math.max(lum(fg), lum(bg)) + 0.05) / (Math.min(lum(fg), lum(bg)) + 0.05);
        const fs = parseFloat(cs.fontSize), need = fs >= 24 || (fs >= 18.66 && parseInt(cs.fontWeight) >= 700) ? 3 : 4.5;
        if (ratio < need) out.push(`${theme} ${p || 'home'}: contrast ${ratio.toFixed(2)} on "${el.textContent.trim().slice(0, 30)}"`);
      }
    }
  }
  window.designCheck = [...new Set(out)];
  console.log(window.designCheck.length ? window.designCheck.join('\n') : 'design check: all clean');
})();
