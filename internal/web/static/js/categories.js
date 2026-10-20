import { api } from './api.js';
import { setCategories } from './ui.js';

// The ways a trip can be labelled on this car: Personal, Work, Medical and the ones the owner added.
let list = [{ key: 'personal', label: 'Personal' }, { key: 'work', label: 'Work' }, { key: 'medical', label: 'Medical' }];

export const categoryList = () => list;
// The colour a category is drawn in: its own, or the app's for Work and Personal.
export const catColor = (key) => {
  const c = list.find((x) => x.key === key);
  if (c && c.color) return c.color;
  return key === 'work' ? 'var(--work)' : key === 'personal' ? 'var(--personal)' : 'var(--muted)';
};
export const catLabel = (key) => (list.find((c) => c.key === key) || {}).label || key;

export async function loadCategories() {
  try { list = await api('/api/categories'); } catch (e) { /* keep the built-in three */ }
  setCategories(list);
  return list;
}
