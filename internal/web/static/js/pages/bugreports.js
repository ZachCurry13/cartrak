import { h, ago } from '../ui.js';
import { api } from '../api.js';
import { copyButton } from '../phonehelp.js';

const when = (ts) => new Date(ts * 1000).toLocaleString([], { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });
const KEEP = [[0, 'Until I clear them'], [30, '30 days'], [60, '60 days'], [90, '90 days']];

// The bug reports kept in the app: each is text (the page's words, the screen, layout facts, the debug report). A link, made on
// demand, lets someone read them without signing in; it expires and can be stopped at any time.
export default async function bugReports(root, _a, session) {
  if (session.me.role !== 'admin') return root.append(h('div', { class: 'banner' }, 'Admins only.'));
  const box = h('div');
  root.append(h('p', {}, h('a', { href: '#/settings' }, '← More')), h('div', { class: 'page-title' }, h('h2', {}, 'Bug reports')),
    h('p', { class: 'muted' }, 'Tap Send a bug report at the foot of any page. The report is kept here as text: what the page said, your screen, how it was laid out and the debug report.'), box);
  const msg = h('p', { class: 'muted', role: 'status' });

  async function load() {
    const d = await api('/api/bug-reports');
    const open = new Set([...box.querySelectorAll('details[open]')].map((x) => x.dataset.id));
    const days = h('select', { id: 'br-days', 'aria-label': 'How long the link works' }, ...[[1, '1 day'], [7, '7 days'], [30, '30 days']].map(([v, t]) => h('option', { value: v, selected: v === 7 }, t)));
    const keep = h('select', { id: 'br-keep', 'aria-label': 'How long reports are kept', onchange: async (e) => { await api('/api/bug-reports/keep', { method: 'PUT', body: { days: Number(e.target.value) } }); load(); } },
      ...KEEP.map(([v, t]) => h('option', { value: v, selected: v === d.keepDays }, t)));
    const link = d.link;
    const fixedKeep = h('select', { id: 'br-fixed', 'aria-label': 'When a report is marked fixed', onchange: async (e) => { await api('/api/bug-reports/keep', { method: 'PUT', body: { fixedDays: Number(e.target.value) } }); load(); } },
      ...[[0, 'Clear it at once'], [1, 'After a day'], [7, 'After a week']].map(([v, t]) => h('option', { value: v, selected: v === d.fixedDays }, t)));
    box.replaceChildren(
      h('section', { class: 'card' }, h('h2', {}, 'A link to read them'),
        link ? h('div', {}, h('p', { class: 'muted' }, `A link is open until ${when(link.expires)}. ${link.opens ? `Opened ${link.opens} time${link.opens === 1 ? '' : 's'}, last ${ago(link.lastUsed)}.` : 'Not opened yet.'}`),
          d.linkUrl ? h('div', {}, h('p', { style: 'overflow-wrap:anywhere' }, d.linkUrl),
            h('p', { class: 'seg' }, copyButton(d.linkUrl, 'Copy the link'), copyButton(`Please read my CarTrak bug reports: ${d.linkUrl}\nEach report has a "page as it looked" copy at ./<number>/page: open it at the window size in the report.`, 'Copy a message for Claude'))) : null,
          d.addressKnown ? null : h('p', { class: 'banner' }, 'CarTrak does not know its public address yet (More, Remote access), so the link cannot be opened from outside.'))
          : h('p', { class: 'muted' }, 'No link is open. Make one, give it to whoever helps, and stop it when they are done.'),
        h('label', { for: 'br-days' }, 'Works for'), days,
        h('p', { class: 'seg' }, h('button', { type: 'button', onclick: async () => { await api('/api/bug-reports/link', { method: 'POST', body: { days: Number(days.value) } }); load(); } }, link ? 'Make a new link' : 'Make a link'),
          link ? h('button', { class: 'sec', type: 'button', onclick: async () => { await api('/api/bug-reports/link', { method: 'DELETE' }); load(); } }, 'Stop the link') : null),
        h('p', { class: 'muted' }, 'Anyone with the link can read every report, including the debug report, until it expires or you stop it. A new link replaces the old one.')),
      h('section', { class: 'card' }, h('h2', {}, 'Keeping and clearing'), h('label', { for: 'br-keep' }, 'Keep reports'), keep, h('label', { for: 'br-fixed' }, 'When a report is marked fixed'), fixedKeep,
        h('p', { class: 'seg' }, h('button', { class: 'sec', type: 'button', onclick: () => clear(30) }, 'Clear older than 30 days'), h('button', { class: 'sec', type: 'button', onclick: clearFixed }, 'Clear the fixed ones'), h('button', { class: 'sec', type: 'button', onclick: () => clear(0) }, 'Clear all')),
        h('p', { class: 'muted' }, 'Whoever has the link can mark a report fixed when it is dealt with. With "Clear it at once" it is removed straight away; otherwise it is hidden and cleared after the time you chose.'), msg),
      h('section', { class: 'card' }, h('h2', {}, `Reports (${d.reports.length})`),
        ...(d.reports.length ? d.reports.map((r) => h('details', { 'data-id': String(r.id), open: open.has(String(r.id)), ontoggle: (e) => { if (e.target.open) show(e.target, r.id); } },
          h('summary', {}, `${r.fixed ? '✓ Fixed · ' : ''}#${r.id} · ${when(r.ts)} · ${r.page}`, r.note ? ` · ${r.note.split('\n')[0].slice(0, 60)}` : ''))) : [h('p', { class: 'muted' }, 'Nothing here yet.')])));
    for (const el of box.querySelectorAll('details[open]')) show(el, Number(el.dataset.id));
  }

  async function show(el, id) {
    if (el.querySelector('pre')) return;
    const r = await api(`/api/bug-reports/${id}`);
    el.append(h('pre', { style: 'white-space:pre-wrap;overflow-wrap:anywhere;margin:var(--sp-2) 0' }, `${r.note}\n\n${r.device}\n\n${r.layout}`),
      h('p', { class: 'seg' }, r.hasPage ? h('a', { class: 'btn sec', href: `/api/bug-reports/${id}/page`, target: '_blank', rel: 'noopener' }, 'See the page as it looked') : null,
        h('button', { class: 'sec', type: 'button', onclick: async () => { await api(`/api/bug-reports/${id}/fixed`, { method: 'PUT', body: { fixed: !r.fixed } }); load(); } }, r.fixed ? 'Open it again' : 'Mark fixed'),
        h('button', { class: 'text sm', type: 'button', onclick: async () => { await api(`/api/bug-reports/${id}`, { method: 'DELETE' }); load(); } }, 'Delete this report')));
  }

  async function clearFixed() {
    const r = await api('/api/bug-reports/clear-fixed', { method: 'POST' });
    msg.textContent = `Cleared ${r.removed} fixed.`;
    load();
  }

  async function clear(days) {
    if (!confirm(days ? `Clear reports older than ${days} days?` : 'Clear every report?')) return;
    const r = await api('/api/bug-reports/clear', { method: 'POST', body: { olderThanDays: days } });
    msg.textContent = `Cleared ${r.removed}.`;
    load();
  }
  await load();
}
