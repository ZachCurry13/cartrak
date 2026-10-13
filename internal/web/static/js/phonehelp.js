import { h } from './ui.js';

// Copies text to the clipboard and says so on the button. Falls back to selecting the text.
export function copyButton(text, label = 'Copy') {
  const b = h('button', { class: 'sec', type: 'button' }, label);
  b.onclick = async () => {
    try { await navigator.clipboard.writeText(text); b.textContent = 'Copied'; } catch (e) { b.textContent = 'Press and hold to copy'; }
    setTimeout(() => { b.textContent = label; }, 2000);
  };
  return b;
}

const block = (title, text) => h('div', {}, h('b', {}, title), h('pre', { class: 'code' }, text), h('p', {}, copyButton(text)));

// The two requests an automation app needs, with the real address and (when known) token filled in.
export function recipes(address, token) {
  const auth = `Authorization: Bearer ${token || 'YOUR_TOKEN'}`;
  return h('div', {},
    block('Trip started or stopped',
      `POST ${address}/api/ingest/event\n${auth}\nContent-Type: application/json\n\n{"type":"drive_start"}\n\nUse "drive_end" when the trip ends.`),
    block('Location points (optional)',
      `POST ${address}/api/ingest/points\n${auth}\nContent-Type: application/json\n\n{"points":[{"ts":1700000000,"lat":44.98,"lon":-93.26,"speed":12,"acc":8}]}\n\nts is unix seconds, speed is metres per second, acc is metres.`),
    block('Background tracking with OwnTracks (iPhone and Android)',
      `Mode: HTTP\nURL: ${address}/api/ingest/owntracks\nUsername: anything\nPassword: ${token || 'YOUR_TOKEN'}`),
    h('p', { class: 'muted' }, 'Step by step for GPSLogger, OwnTracks, Tasker, MacroDroid and iPhone Shortcuts: see docs/phone-tracking.md in the CarTrak repository on GitHub.'));
}
