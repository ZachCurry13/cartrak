# Notifications (alerts)

CarTrak can send a notification to your phone or computer when something happens to the car. Open **More, Notifications** on each device you want alerts on, press **Turn on alerts here**, choose the alerts, and use **Send a test alert** to check it works.

## The alerts
| Alert | When it is sent |
|---|---|
| Charging complete | Charging stopped while the car is still plugged in (the battery reached its target or is full) |
| Plugged in / Unplugged | The charging cable was connected or removed |
| Low tire pressure | The car reports a tire as low (sent once, until the tire is fixed) |
| Charging interrupted / stopped early | A charge stopped while still plugged in, lower than your sessions usually end. Off by default |
| Bump while parked | A parked car's motion sensors read above the bump level (More, Settings). Experimental: see Parked watch. Off by default |
| Car alarm | The car reports its alarm or panic alarm is going off. Off by default |
| Left open | The parked car has a door, the tailgate, the hood or a window open for ten minutes. Off by default |
| Backup reminder | No backup has been downloaded in the last month. At most once a month, and only after CarTrak has been recording for two weeks |
| Low battery | The battery fell to the level an admin chose under More, Settings, while the car was not charging. Once, then again after it has recovered five points. Off until a level is chosen |
| Charging budget | This month's charging cost passed the budget an admin set under More, Settings. Once a month. Needs your price per kWh |
| Low disk space | The disk CarTrak stores its data on is nearly full (under 5 GB or 8 percent free). At most once every three days. Off by default |
| Weekly summary | On Sunday evening (6 pm, car time zone): distance, trips, charging and efficiency for the last seven days. Off by default |
| Service due | A reminder from More, Maintenance came due by distance or time |
| Arrived at / Left a named place | The car arrived at or left a place named under More, Places |
| Plug in tonight? | One reminder in the evening (9 pm by default) when the car is parked, not plugged in and below 60 percent. Off by default for each device: turn it on in Notifications. An admin can change the time and the level in Settings (Plug-in reminder). At most once a day, and only if the car has reported in the last two hours |
| Low 12 V battery | The 12 V battery is below 11.8 V or under 50 percent charge. At most once a day |
| Car left unlocked | The car has been parked and unlocked for more than 10 minutes (once per time it happens) |
| Trip started / Trip ended | CarTrak started or finished recording a trip; the end alert shows the distance |
| Ford connection trouble | No reading from the car for 4 hours. Open Status to see why |

A new device starts with Charging complete, Low tire pressure and Low 12 V battery. Each device has its own choices, and each can set **quiet hours** (for example 22:00 to 07:00) when nothing is sent. Quiet hours follow the car time zone from Settings.

Each alert is sent once. After CarTrak restarts it does not repeat what was already true, and it only announces things that change from then on.

## Where it works
- **Android (Chrome) and computers (Chrome, Edge, Firefox):** open the page over the secure address.
- **iPhone and iPad:** add CarTrak to the Home Screen first (Share, then Add to Home Screen) and open it from there. iOS 16.4 or newer is needed.
- **Secure address only.** Browsers allow notifications only on `https://` (for example your Cloudflare address) or `http://localhost`. Over plain `http://` on your home network the page says so.
- Allow notifications for the site when the browser asks. If you said no, change it in the browser's site settings.

## Privacy
- An alert is **encrypted** so only your device can read it, and CarTrak sends it through the push service of your browser's maker (Google for Chrome and Android, Mozilla for Firefox, Apple for iPhone). That service can see that a message was sent to your device and when, but not what it says.
- Alerts contain short text such as "Charging complete. Battery at 80%." and never a location.
- CarTrak only ever contacts the real push services (Google, Mozilla, Apple, Microsoft addresses), so nobody can aim it at another address.
- The server's identity key (VAPID) is made on first use and kept in the database. The private half never leaves the server.
- Removing a device (More, Notifications, Remove) stops its alerts at once. A device that has uninstalled the app or revoked permission is removed automatically when the push service says it is gone.

## Troubleshooting
- **Turn on alerts says blocked:** the browser's site notification permission is denied; allow it in the site settings and try again.
- **Test alert fails with "the push service answered HTTP ...":** the push service refused the message; try Turn off, then Turn on again on that device.
- **Nothing arrives on iPhone:** it must be opened from the Home Screen icon, and notifications allowed in iOS Settings for CarTrak.
- **No alerts for trips or charging:** they follow Ford's readings, which can be minutes apart and only update when the car wakes. Check Status.
