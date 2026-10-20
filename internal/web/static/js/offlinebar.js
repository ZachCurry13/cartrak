// A thin bar saying the screen is showing what this device saved, because there is no signal. It goes away on the next answer from the server.
let bar = null;

const clock = (t) => new Date(t).toLocaleString([], { weekday: 'short', hour: 'numeric', minute: '2-digit' });

export function showOffline(t) {
  if (!bar) {
    bar = document.createElement('div');
    bar.className = 'offbar';
    bar.setAttribute('role', 'status');
    document.body.append(bar);
  }
  bar.textContent = `Offline. Showing what this device saved (${clock(t)}). Changes need a connection.`;
  bar.hidden = false;
}

export function hideOffline() { if (bar) bar.hidden = true; }
