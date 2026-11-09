import { h } from './ui.js';

// "Copy link" and (where the phone offers it) "Share" for a link, so a route can be sent to someone instead of only opened here.
const copyText = async (text) => {
  try { await navigator.clipboard.writeText(text); return true; } catch (e) { /* older app views refuse: select and copy below */ }
  const t = h('textarea', { value: text, readonly: true, style: 'position:fixed;left:-9999px' });
  document.body.append(t); t.select();
  let ok = false;
  try { ok = document.execCommand('copy'); } catch (e) { ok = false; }
  t.remove();
  return ok;
};

export function linkActions(href, what = 'link') {
  const copy = h('button', { class: 'sec', type: 'button' }, `Copy ${what}`);
  copy.onclick = async () => {
    copy.textContent = (await copyText(href)) ? 'Copied' : 'Could not copy';
    setTimeout(() => { copy.textContent = `Copy ${what}`; }, 2000);
  };
  const out = [copy];
  if (navigator.share) out.push(h('button', { class: 'sec', type: 'button', onclick: () => navigator.share({ title: 'Route', url: href }).catch(() => {}) }, 'Share'));
  return out;
}
