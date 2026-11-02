import { h } from './ui.js';

// Links that open a spot in the phone's own map app: Google Maps everywhere, Apple Maps on Apple devices.
// Nothing is sent anywhere until the link is tapped.
export const googleMapsUrl = (lat, lon) => `https://www.google.com/maps?q=${lat},${lon}`;
export const directionsUrl = (lat, lon) => `https://www.google.com/maps/dir/?api=1&destination=${lat},${lon}&travelmode=driving`;
// A whole route in Google Maps: the first point is the start, the last the destination and any between are stops, in order.
// Google takes up to nine stops between a start and a destination, so a longer route (ten customers and the way home) becomes
// several links that carry on from where the one before ended. Points are [lat, lon].
export function routeUrls(points) {
  const ll = (p) => `${p[0].toFixed(6)},${p[1].toFixed(6)}`;
  const urls = [];
  for (let from = 0; from < points.length - 1; from += 10) {
    const part = points.slice(from, from + 11);
    const q = new URLSearchParams({ api: '1', origin: ll(part[0]), destination: ll(part[part.length - 1]), travelmode: 'driving' });
    if (part.length > 2) q.set('waypoints', part.slice(1, -1).map(ll).join('|'));
    urls.push(`https://www.google.com/maps/dir/?${q}`);
  }
  return urls;
}
export const appleMapsUrl = (lat, lon) => `https://maps.apple.com/?daddr=${lat},${lon}&dirflg=d`;
const apple = () => /iPhone|iPad|iPod|Macintosh/.test(navigator.userAgent);

// Buttons for one place. `what` is how to refer to it: "the car", "where it ended".
export function mapButtons(lat, lon, what = 'the car') {
  const link = (href, text) => h('a', { class: 'btn sec', href, target: '_blank', rel: 'noopener' }, text);
  return h('p', { class: 'seg', style: 'margin:8px 0 0' },
    link(googleMapsUrl(lat, lon), 'Open in Google Maps'),
    link(directionsUrl(lat, lon), `Directions to ${what}`),
    apple() ? link(appleMapsUrl(lat, lon), 'Open in Apple Maps') : null);
}
