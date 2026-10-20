// Which car this tab is looking at. Each tab keeps its own choice (sessionStorage), and the last choice is remembered for
// the next visit. The server also gets it in a cookie, for links that open in the browser instead of through the app.
const KEY = 'car';

export function currentCar() {
  try { return sessionStorage.getItem(KEY) || localStorage.getItem(KEY) || ''; } catch (e) { return ''; }
}

export function setCar(id) {
  const v = String(id);
  try { sessionStorage.setItem(KEY, v); localStorage.setItem(KEY, v); } catch (e) { /* storage blocked */ }
  document.cookie = `cartrak_car=${encodeURIComponent(v)}; path=/; SameSite=Lax`;
}

// Adds the car to an address that is opened by the browser itself (a download, the Ford sign-in).
export function withCar(url) {
  const c = currentCar();
  return c ? `${url}${url.includes('?') ? '&' : '?'}car=${encodeURIComponent(c)}` : url;
}
