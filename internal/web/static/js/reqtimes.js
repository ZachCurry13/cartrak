// How long each kind of request took to come back on this phone (the network included), kept in memory only and shown nowhere in
// the app. A debug report adds the slowest, beside the server's own times, so a slow page can be told apart: server or connection.
const stats = new Map();

export function noteRequest(path, ms) {
  const key = path.split('?')[0].replace(/\/\d+/g, '/{id}');
  const s = stats.get(key) || { n: 0, total: 0, max: 0 };
  s.n += 1; s.total += ms; s.max = Math.max(s.max, ms);
  stats.set(key, s);
}

export function slowestRequests() {
  return [...stats.entries()].map(([k, s]) => ({ k, n: s.n, avg: Math.round(s.total / s.n), max: Math.round(s.max) }))
    .sort((a, b) => b.avg - a.avg).slice(0, 12)
    .map((r) => `      ${r.k}: average ${r.avg} ms, slowest ${r.max} ms, ${r.n} time${r.n === 1 ? '' : 's'}`);
}
