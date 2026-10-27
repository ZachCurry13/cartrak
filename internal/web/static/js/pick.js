export function pick(extra, ...names) {
  for (const nm of names) {
    const l = nm.toLowerCase().replace(/_/g, '');
    for (const [k, v] of Object.entries(extra || {})) {
      const lk = k.toLowerCase().replace(/_/g, '');
      if (lk === l || lk.endsWith('.' + l) || lk === l + '.value' || lk.endsWith('.' + l + '.value')) return v;
    }
  }
  return null;
}
