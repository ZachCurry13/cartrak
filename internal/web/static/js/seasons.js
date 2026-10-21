// Seasonal themes: the accent color, a header ornament, the Wrapped colors and a short effect on Home follow the holidays
// around the date. Each device chooses: Automatic (the default), one theme, or Off. Only the accent changes; green still means
// charging and amber driving, and every theme has a deeper accent for light mode so text on it stays readable.

// key: name, emoji, accent (dark, light), text on the accent (dark, light), two Wrapped colors, the Home effect
export const THEMES = {
  newyear: ['New Year', '🎉', '#f5c542', '#8a6300', '#2a1f00', '#fff', '#f5c542', '#7c3aed', 'confetti'],
  winter: ['Winter', '❄️', '#7cc7ff', '#1d5f9a', '#04203a', '#fff', '#38bdf8', '#6366f1', 'snow'],
  valentine: ["Valentine's", '💕', '#ff6b9a', '#b4245a', '#3a0618', '#fff', '#f43f5e', '#ec4899', 'hearts'],
  stpatrick: ["St. Patrick's", '☘️', '#3ddc84', '#11753f', '#03240f', '#fff', '#16a34a', '#facc15', 'shamrocks'],
  easter: ['Easter', '🐣', '#c4a7ff', '#6d48c2', '#1d0d40', '#fff', '#a78bfa', '#f9a8d4', 'eggs'],
  spring: ['Spring', '🌷', '#ff8fb8', '#a8305f', '#3a0a1e', '#fff', '#f472b6', '#4ade80', 'petals'],
  summer: ['Summer', '☀️', '#ffb020', '#965400', '#2a1800', '#fff', '#f59e0b', '#06b6d4', 'sun'],
  july4: ['Fourth of July', '🎆', '#ff5a5f', '#b3261e', '#330606', '#fff', '#ef4444', '#3b82f6', 'fireworks'],
  autumn: ['Autumn', '🍂', '#ff9a3c', '#9c4a00', '#2e1400', '#fff', '#ea580c', '#a16207', 'leaves'],
  halloween: ['Halloween', '🎃', '#ff8a1f', '#a84a00', '#2a1300', '#fff', '#f97316', '#7c3aed', 'bats'],
  thanksgiving: ['Thanksgiving', '🦃', '#e8a33d', '#8a5200', '#2a1700', '#fff', '#d97706', '#b45309', 'leaves'],
  christmas: ['Christmas', '🎄', '#ff4d5e', '#b3132b', '#33040b', '#fff', '#dc2626', '#16a34a', 'lights'],
};

const md = (m, d) => m * 100 + d;

// Easter Sunday (Gregorian), as month*100 + day.
function easter(y) {
  const a = y % 19, b = Math.floor(y / 100), c = y % 100, d = Math.floor(b / 4), e = b % 4, f = Math.floor((b + 8) / 25), g = Math.floor((b - f + 1) / 3);
  const h = (19 * a + b - d - g + 15) % 30, i = Math.floor(c / 4), k = c % 4, l = (32 + 2 * e + 2 * i - h - k) % 7, m = Math.floor((a + 11 * h + 22 * l) / 451);
  return md(Math.floor((h + l - 7 * m + 114) / 31), ((h + l - 7 * m + 114) % 31) + 1);
}

// Thanksgiving: the fourth Thursday of November.
function thanksgiving(y) {
  const first = new Date(y, 10, 1).getDay();
  return md(11, 1 + ((4 - first + 7) % 7) + 21);
}

// The theme for a date, and whether it is the holiday itself (a bigger burst that day).
export function themeFor(date = new Date()) {
  const y = date.getFullYear(), now = md(date.getMonth() + 1, date.getDate());
  const e = easter(y), eDay = new Date(y, Math.floor(e / 100) - 1, e % 100);
  const twoWeeksBefore = new Date(eDay); twoWeeksBefore.setDate(eDay.getDate() - 14);
  const eFrom = md(twoWeeksBefore.getMonth() + 1, twoWeeksBefore.getDate());
  const tg = thanksgiving(y);
  const pick = (key, day) => ({ key, day: day === now });
  if (now >= 1231 || now <= 101) return pick('newyear', now <= 101 ? 101 : 1231);
  if (now >= 201 && now <= 214) return pick('valentine', 214);
  if (now >= 301 && now <= 317) return pick('stpatrick', 317);
  if (now >= eFrom && now <= e) return pick('easter', e);
  if (now >= 318 && now <= 531) return pick('spring');
  if (now >= 701 && now <= 704) return pick('july4', 704);
  if (now >= 601 && now <= 831) return pick('summer');
  if (now >= 901 && now <= 930) return pick('autumn');
  if (now >= 1001 && now <= 1031) return pick('halloween', 1031);
  if (now >= 1101 && now <= tg) return pick('thanksgiving', tg);
  if (now > tg && now <= 1226) return pick('christmas', 1225);
  return pick('winter');
}

const KEY = 'season';
export const getSeasonPref = () => { try { return localStorage.getItem(KEY) || 'auto'; } catch (e) { return 'auto'; } };
export const setSeasonPref = (v) => { try { localStorage.setItem(KEY, v); } catch (e) { /* storage blocked */ } };

// The theme in use now: the chosen one, the date's, or none.
export function activeTheme() {
  const pref = getSeasonPref();
  if (pref === 'off') return null;
  if (THEMES[pref]) return { key: pref, day: false };
  return themeFor();
}

// One stylesheet with every theme's colors, for dark and light, so the page switches with an attribute.
function styles() {
  let css = '';
  for (const [k, t] of Object.entries(THEMES)) {
    const dark = `--accent:${t[2]};--accent-text:${t[4]};--season1:${t[6]};--season2:${t[7]}`, light = `--accent:${t[3]};--accent-text:${t[5]};--season1:${t[6]};--season2:${t[7]}`;
    css += `:root[data-season=${k}]{${dark}}@media (prefers-color-scheme:light){:root[data-season=${k}]:not([data-theme=dark]){${light}}}:root[data-season=${k}][data-theme=light]{${light}}`;
  }
  return css;
}

// Puts the theme on the page: the colors, and the ornament beside the name in the header.
export function applySeason() {
  if (!document.getElementById('seasoncss')) {
    const st = document.createElement('style');
    st.id = 'seasoncss';
    st.textContent = styles();
    document.head.append(st);
  }
  const t = activeTheme();
  const root = document.documentElement;
  if (t) root.dataset.season = t.key; else delete root.dataset.season;
  const h1 = document.querySelector('#top h1');
  let orn = document.getElementById('ornament');
  if (h1 && t) {
    if (!orn) { orn = document.createElement('span'); orn.id = 'ornament'; orn.setAttribute('aria-hidden', 'true'); h1.append(orn); } // inside the title, so the header row stays one line
    orn.textContent = THEMES[t.key][1];
    orn.title = THEMES[t.key][0];
  } else if (orn) orn.remove();
  return t;
}
