import { h } from './ui.js';
import { api } from './api.js';

// Making or changing a custom report: its columns in order, the heading each one shows, columns with a fixed value (a company name, a
// cost center), and whether the summary sits above the table.
export function reportEditor(opts, tpl, done) {
  const cfg = tpl ? JSON.parse(JSON.stringify(tpl.config)) : { columns: opts.columns.filter((c) => c.default).map((c) => ({ key: c.key, label: '' })), fixed: [], summary: true, title: '' };
  cfg.fixed = cfg.fixed || [];
  const name = h('input', { id: 'rn', maxlength: 40, value: tpl ? tpl.name : '', placeholder: 'For example FedEx report' });
  const title = h('input', { id: 'rtt', maxlength: 80, value: cfg.title || '', placeholder: 'First line of the summary (optional)' });
  const summary = h('input', { type: 'checkbox', id: 'rs', checked: cfg.summary });
  const msg = h('p', { class: 'muted', role: 'alert' });
  const list = h('div');
  const fixedList = h('div');
  const label = (key) => (opts.columns.find((c) => c.key === key) || {}).label || key;

  function drawCols() {
    list.replaceChildren(...cfg.columns.map((c, i) => {
      const sel = h('select', { 'aria-label': 'Column', onchange: (e) => { c.key = e.target.value; drawCols(); } }, ...opts.columns.map((o) => h('option', { value: o.key, selected: o.key === c.key }, o.label)));
      const head = h('input', { value: c.label || '', maxlength: 60, placeholder: `Heading (${label(c.key)})`, 'aria-label': 'Heading', oninput: (e) => { c.label = e.target.value; } });
      const move = (d) => () => { const j = i + d; if (j < 0 || j >= cfg.columns.length) return; [cfg.columns[i], cfg.columns[j]] = [cfg.columns[j], cfg.columns[i]]; drawCols(); };
      return h('div', { class: 'card', style: 'margin:0 0 8px' }, sel, head,
        h('p', { class: 'seg', style: 'margin:6px 0 0' }, h('button', { class: 'sec', type: 'button', 'aria-label': 'Move up', onclick: move(-1) }, '↑'), h('button', { class: 'sec', type: 'button', 'aria-label': 'Move down', onclick: move(1) }, '↓'),
          h('button', { class: 'danger', type: 'button', onclick: () => { cfg.columns.splice(i, 1); drawCols(); } }, 'Remove')));
    }));
  }
  function drawFixed() {
    fixedList.replaceChildren(...cfg.fixed.map((f, i) => h('div', { class: 'card', style: 'margin:0 0 8px' },
      h('input', { value: f.label, maxlength: 60, placeholder: 'Heading', 'aria-label': 'Fixed column heading', oninput: (e) => { f.label = e.target.value; } }),
      h('input', { value: f.value, maxlength: 200, placeholder: 'Value on every row', 'aria-label': 'Fixed column value', oninput: (e) => { f.value = e.target.value; } }),
      h('p', { style: 'margin:6px 0 0' }, h('button', { class: 'danger', type: 'button', onclick: () => { cfg.fixed.splice(i, 1); drawFixed(); } }, 'Remove')))));
  }
  drawCols();
  drawFixed();

  async function save() {
    try {
      cfg.summary = summary.checked;
      cfg.title = title.value.trim();
      await api('/api/reports/templates', { method: 'POST', body: { id: tpl ? tpl.id : 0, name: name.value, config: cfg } });
      done(true);
    } catch (e) { msg.textContent = e.message; }
  }
  return h('section', { class: 'card' }, h('h2', {}, tpl ? 'Change this report' : 'New custom report'),
    h('label', { for: 'rn' }, 'Name'), name,
    h('label', { for: 'rtt' }, 'Summary title'), title,
    h('label', { for: 'rs', style: 'display:flex;gap:8px;align-items:center;margin:8px 0' }, summary, 'Put a summary above the table, and a totals sheet'),
    h('h3', { style: 'margin:14px 0 6px' }, 'Columns, in order'), list,
    h('p', {}, h('button', { class: 'sec', type: 'button', onclick: () => { cfg.columns.push({ key: opts.columns[0].key, label: '' }); drawCols(); } }, 'Add a column')),
    h('h3', { style: 'margin:14px 0 6px' }, 'Fixed columns'), h('p', { class: 'muted' }, 'A column with the same value on every row, such as a company name or a cost center.'), fixedList,
    h('p', {}, h('button', { class: 'sec', type: 'button', onclick: () => { cfg.fixed.push({ label: '', value: '' }); drawFixed(); } }, 'Add a fixed column')),
    h('p', { class: 'seg' }, h('button', { onclick: save }, 'Save report'), h('button', { class: 'sec', onclick: () => done(false) }, 'Cancel')), msg);
}
