import { api } from './api.js';

// A small red dot on the More tab when a service needs a look (Ford not set up, or a test that failed). Admins only.
let count = 0;

function paint() {
  for (const a of document.querySelectorAll('#tabs a[href="#/settings"]')) {
    let dot = a.querySelector('.attn');
    if (!dot) { dot = document.createElement('i'); dot.className = 'attn'; a.append(dot); }
    dot.hidden = count <= 0;
    dot.setAttribute('role', 'img');
    dot.setAttribute('aria-label', 'Something needs attention in Integrations');
  }
}

export async function refreshAttention() {
  try { count = (await api('/api/integrations/attention')).attention || 0; } catch (e) { count = 0; }
  paint();
  const tabs = document.getElementById('tabs');
  if (tabs && !tabs.dataset.attn) { tabs.dataset.attn = '1'; new MutationObserver(paint).observe(tabs, { childList: true }); } // the bar is rebuilt when its layout changes
}
