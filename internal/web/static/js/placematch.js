import { dist } from './ui.js';

// What to say under a search result's name: what it is, its short address, and how far away it is.
export const matchDetail = (m) => [m.kind ? m.kind[0].toUpperCase() + m.kind.slice(1) : null, m.address, m.distKm != null ? `${dist(m.distKm, 0)} away` : null].filter(Boolean).join(' · ');
