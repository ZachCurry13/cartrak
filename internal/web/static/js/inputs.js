import { h } from './ui.js';

// A number the person types, drawn the same everywhere: a finger-sized box with the currency before it and what it is "per" or
// "total" after it, the whole number selected when it is tapped, and the number keypad on a phone. Money, a rate and a price all
// come through here, so they look and behave the same on every screen. Returns the element to place and the input to read.
export function numberInput({ value = '', currency = '', unit = '', step = '1', min = 0, max, placeholder = '', label, disabled = false, id }) {
  const input = h('input', { id, type: 'number', step, min, max, inputmode: 'decimal', value: value == null ? '' : value, placeholder, 'aria-label': label, disabled });
  input.addEventListener('focus', () => input.select());
  const el = h('span', { class: 'fieldbox' }, currency ? h('span', { class: 'pre', 'aria-hidden': 'true' }, currency) : null, input, unit ? h('span', { class: 'suf', 'aria-hidden': 'true' }, unit) : null);
  return { el, input };
}

// Money: two decimals.
export const moneyInput = (opts) => numberInput({ step: '0.01', ...opts });
