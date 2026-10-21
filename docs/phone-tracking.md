# Phone tracking (optional)

A phone can tell CarTrak the moment a trip starts or ends, and send its GPS so the trip page draws a finer route.
Ford still decides what counts as a trip, so the odometer and battery numbers stay exact. If you set up nothing,
nothing changes.

What you get:
- **Faster start:** when the phone says "trip started", CarTrak asks Ford right away and checks more often until Ford reports the trip, instead of waiting for the parked interval.
- **Faster finish:** "trip ended" makes CarTrak check straight away so the trip closes promptly.
- **A finer route:** GPS points from the phone are drawn on the trip page instead of Ford's readings (which can be 30 seconds or more apart).

Browsers and phones cannot watch for your car in the background by themselves. An automation app on the phone does that and sends a message to CarTrak.

## 1. Pair the phone
1. Open CarTrak, **More -> Phone tracking**, type a name, press **Create token**.
2. Copy the token now. It is shown once and CarTrak keeps only a scrambled copy. Lost it? Remove the phone and add it again.
3. Press **Send a test check-in**. The phone shows "Heard from just now" in the list and the message appears under Recent messages.

The phone must be able to reach CarTrak when you are away from home, so set up [remote access](cloudflare.md) first and use that address (the page shows it in the examples).

## 2. Send messages from the phone
Both messages are a `POST` with two headers:

```
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json
```

| When | Address | Body |
|---|---|---|
| Trip starts | `https://YOUR-ADDRESS/api/ingest/event` | `{"type":"drive_start"}` |
| Trip ends | `https://YOUR-ADDRESS/api/ingest/event` | `{"type":"drive_end"}` |
| Still alive (optional) | `https://YOUR-ADDRESS/api/ingest/event` | `{"type":"ping"}` |

Any message can also carry a position: `"lat":44.98,"lon":-93.26`, plus optional `"acc"` (meters) and `"speed"` (meters per second). Numbers in quotes are fine too, because automation apps often add them.

### Android: Tasker
1. New **Profile -> State -> Net -> BT Connected**, pick your car's Bluetooth name.
2. In the **entry task** add **Net -> HTTP Request**: Method `POST`, URL as above, Headers (one per line) `Authorization: Bearer YOUR_TOKEN` and `Content-Type: application/json`, Body `{"type":"drive_start"}`.
3. In the **exit task** add the same action with Body `{"type":"drive_end"}`.
4. Exclude Tasker from battery optimization so it runs when the screen is off.

### Android: MacroDroid
1. New macro, trigger **Connectivity -> Bluetooth Event -> Device Connected** (your car). Add trigger **Disconnected** to a second macro.
2. Action **Connectivity -> HTTP Request**: `POST`, the URL above, content type `application/json`, body `{"type":"drive_start"}`, and a header `Authorization` with value `Bearer YOUR_TOKEN`.
3. Second macro: same, body `{"type":"drive_end"}`.

### iPhone: Shortcuts
1. **Shortcuts -> Automation -> New Automation -> Bluetooth** (or **CarPlay**), choose your car, **Is Connected**.
2. Add **Get Contents of URL**: URL as above, Method `POST`, Headers `Authorization` = `Bearer YOUR_TOKEN` and `Content-Type` = `application/json`, Request Body `JSON` with a text field `type` = `drive_start`.
3. Turn on **Run Immediately** (not "Ask Before Running") so it fires by itself.
4. Make a second automation for **Is Disconnected** with `drive_end`.

## 3. GPS points (optional)
For a route you can send `POST /api/ingest/points` with `{"points":[{"ts":1700000000,"lat":44.98,"lon":-93.26,"speed":12,"acc":8}]}` (up to 1000 points per call; `ts` is unix seconds or milliseconds). Automation apps can do this on a timer while driving.

Without an app, open **Live** on the phone and press **Record GPS from this phone**. That records only while the page is open with the screen on (see 3b for background tracking), and keeps points in the browser until they are sent, so a dropped signal loses nothing.

A phone that reports traveling faster than walking pace also counts as "trip started" for the faster checks.

## 3b. Tracking in the background (so the page does not need to be open)
A web page cannot track you once you leave it: phones put it to sleep. A free tracking app can run in the background and send each position to CarTrak. Two work well; pick the one for your phone. Both use the token from step 1, so create a device first.

### Android: GPSLogger (open source)
1. Install **GPSLogger** from F-Droid or Google Play.
2. Open Settings, **Logging details**, then **Log to custom URL** and switch it on.
3. Set the **URL** to `https://YOUR-ADDRESS/api/ingest/points` and the **HTTP method** to `POST`.
4. Under **HTTP headers** add two lines: `Authorization: Bearer YOUR_TOKEN` and `Content-Type: application/json`.
5. Under **HTTP body** enter: `{"points":[{"ts":%TIMESTAMP,"lat":%LAT,"lon":%LON,"speed":%SPD,"acc":%ACC}]}`
6. Turn on **Allow auto sending** and leave **Discard offline locations** off, so positions recorded without a signal are sent later.
7. Set the **logging interval** to about 5 to 10 seconds, switch off battery optimization for the app, and turn on **Start on boot**.

Check it worked: More, Phone tracking should show "Heard from just now" and a growing point count for that device. The placeholder names above (`%LAT`, `%LON`, `%SPD`, `%ACC`, `%TIMESTAMP`) come from published setup guides; if the app lists different names in its own help, use those. If a language setting makes it write decimals with commas, switch the phone to a region that uses dots. GPSLogger can also be started and stopped by Tasker or MacroDroid, for example when the car's Bluetooth connects.

### iPhone and Android: OwnTracks
1. Install **OwnTracks** from the App Store or Google Play and allow location **Always**.
2. Open Preferences, **Connection**, and set **Mode** to **HTTP**.
3. Set the **URL** to `https://YOUR-ADDRESS/api/ingest/owntracks`.
4. Under **Identification** set any **Username** (for example `phone`) and use the device **token as the Password**.
5. Choose a monitoring mode: **Significant** is gentle on the battery, **Move** sends positions often while you are moving.
6. Press **Report location** to test; the device shows as heard from.

OwnTracks only sends when the phone decides to, and iPhones sometimes pause background apps; if it goes quiet, opening the app resumes it. One Home Assistant document says the Android version of OwnTracks is no longer supported, so on Android GPSLogger is the safer choice.

A phone that reports traveling faster than walking pace also makes CarTrak check the car more often, exactly like the "trip started" message.

## Privacy and housekeeping
- Tokens are stored only as a hash. Removing a phone makes its token stop working at once; points it already sent are kept.
- Location points are stored in your CarTrak database only. To delete old points automatically set **Delete phone GPS points after (days)** in Admin settings (or `CARTRAK_TRACK_KEEP_DAYS`). It is empty by default, which keeps everything. Trips are never deleted by it.
- Five wrong tokens from one address lock it out for a minute, and each phone is limited to 300 messages a minute.

## Troubleshooting
- **Nothing shows under Recent messages:** the automation did not run or could not reach CarTrak. Open the CarTrak address in the phone's browser to check it loads away from home.
- **Error 401:** the token is missing, mistyped, has "Bearer" left out, or the phone was removed.
- **Error 429:** too many requests, or too many wrong tokens recently. Wait a minute.
- **Error 400:** the body is not valid JSON, or `type` is not one of `drive_start`, `drive_end`, `ping`, or the time is more than 14 days away from now.
