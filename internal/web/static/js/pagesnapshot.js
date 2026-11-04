// A copy of the page as it looks right now, as text: its markup with every style rule of the app written in, and nothing that can run.
// Opened later at the same window size it shows the same layout, so a layout problem can be seen without a screenshot. Pictures,
// maps and charts drawn on a canvas do not come along; what was typed into fields is left out.
const MAX = 1500000; // characters; a bigger page is saved without its copy

const styles = () => {
  const out = [];
  for (const sheet of document.styleSheets) {
    try { for (const rule of sheet.cssRules) out.push(rule.cssText); } catch (e) { /* a sheet from another site cannot be read */ }
  }
  return out.join('\n');
};

export function pageSnapshot() {
  try {
    const root = document.documentElement.cloneNode(true);
    root.querySelectorAll('script,link,noscript,iframe,object,embed,video,audio,canvas,meta,title,base').forEach((e) => e.remove());
    root.querySelectorAll('input,textarea,select').forEach((e) => { e.removeAttribute('value'); if (e.tagName === 'TEXTAREA') e.textContent = ''; });
    root.querySelectorAll('[src]').forEach((e) => { if (!String(e.getAttribute('src')).startsWith('data:')) e.removeAttribute('src'); });
    root.querySelectorAll('*').forEach((e) => { for (const a of [...e.attributes]) if (/^on/i.test(a.name)) e.removeAttribute(a.name); });
    const head = document.createElement('head');
    head.innerHTML = '<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">';
    const style = document.createElement('style');
    style.textContent = styles();
    head.append(style);
    root.querySelector('head')?.remove();
    root.prepend(head);
    const html = `<!doctype html>\n${root.outerHTML}`;
    return html.length > MAX ? '' : html;
  } catch (e) { return ''; }
}
