import { api } from './api.js';
import { setCategories } from './ui.js';

// The ways a trip can be labelled on this car: Personal, Work, Medical and the ones the owner added.
let list = [{ key: 'personal', label: 'Personal' }, { key: 'work', label: 'Work' }, { key: 'medical', label: 'Medical' }];

export const categoryList = () => list;
export const catLabel = (key) => (list.find((c) => c.key === key) || {}).label || key;

export async function loadCategories() {
  try { list = await api('/api/categories'); } catch (e) { /* keep the built-in three */ }
  setCategories(list);
  return list;
}
