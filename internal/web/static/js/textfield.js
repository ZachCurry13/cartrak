// Material text fields: a label that sits in the field and floats up to its top edge when the field is focused or has something in it.
// Pages keep writing a <label> followed by its field; this moves the label into a wrapper beside the field (the field itself is not
// replaced, so a page's own references to it still work). A field with a visible placeholder, a picker, or a date keeps its label up.
const FIELD = 'input[type=text],input[type=number],input[type=search],input[type=email],input[type=password],input[type=url],input[type=tel],input:not([type]),select,textarea,input[type=date],input[type=time],input[type=datetime-local],input[type=month]';
const SKIP = '.fieldbox,.filters,.seg,.days,.mfield,.finder,.sheet,.dock,.hmonth,.wrapped,.mfree';
let n = 0;

const alwaysUp = (f) => f.tagName === 'SELECT' || /^(date|time|datetime-local|month)$/.test(f.type);

export function upgradeFields(root) {
  for (const label of root.querySelectorAll('label:not([data-mf])')) {
    const field = label.nextElementSibling;
    if (!field || !field.matches(FIELD) || field.hidden || field.hasAttribute('data-unit') || field.closest('.fieldbox') || label.closest(SKIP) || label.querySelector('input,select,textarea')) continue; // a field with a unit goes in its own box (consistency.js), with the label above
    if (label.htmlFor && field.id && label.htmlFor !== field.id) continue;
    label.dataset.mf = '1';
    if (!field.id) field.id = `mf${++n}`;
    label.htmlFor = field.id;
    const wrap = document.createElement('div');
    wrap.className = 'mfield';
    if (alwaysUp(field) || field.placeholder.trim()) wrap.classList.add('always');
    else if (field.tagName !== 'SELECT') field.placeholder = ' '; // lets CSS tell an empty field from a filled one
    label.before(wrap);
    wrap.append(field, label);
  }
}
