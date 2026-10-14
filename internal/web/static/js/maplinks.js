import { h } from './ui.js';

// Links that open a spot in the phone's own map app: Google Maps everywhere, Apple Maps on Apple devices.
// Nothing is sent anywhere until the link is tapped.
export const googleMapsUrl = (lat, lon) => `https://www.google.com/maps?q=${lat},${lon}`;
export const directionsUrl = (lat, lon) => `https://www.google.com/maps/dir/?api=1&destination=${lat},${lon}&travelmode=driving`;
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
