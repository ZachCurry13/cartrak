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

Any message can also carry a position: `"lat":44.98,"lon":-93.26`, plus optional `"acc"` (metres) and `"speed"` (metres per second). Numbers in quotes are fine too, because automation apps often add them.

### Android: Tasker
1. New **Profile -> State -> Net -> BT Connected**, pick your car's Bluetooth name.
2. In the **entry task** add **Net -> HTTP Request**: Method `POST`, URL as above, Headers (one per line) `Authorization: Bearer YOUR_TOKEN` and `Content-Type: application/json`, Body `{"type":"drive_start"}`.
3. In the **exit task** add the same action with Body `{"type":"drive_end"}`.
4. Exclude Tasker from battery optimisation so it runs when the screen is off.

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

Without an automation app, open **Live** on the phone and press **Record GPS from this phone**. That records while the page is open with the screen on, and keeps points in the browser until they are sent, so a dropped signal loses nothing.

A phone that reports travelling faster than walking pace also counts as "trip started" for the faster checks.

## Privacy and housekeeping
- Tokens are stored only as a hash. Removing a phone makes its token stop working at once; points it already sent are kept.
- Location points are stored in your CarTrak database only. To delete old points automatically set **Delete phone GPS points after (days)** in Admin settings (or `CARTRAK_TRACK_KEEP_DAYS`). It is empty by default, which keeps everything. Trips are never deleted by it.
- Five wrong tokens from one address lock it out for a minute, and each phone is limited to 300 messages a minute.

## Troubleshooting
- **Nothing shows under Recent messages:** the automation did not run or could not reach CarTrak. Open the CarTrak address in the phone's browser to check it loads away from home.
- **Error 401:** the token is missing, mistyped, has "Bearer" left out, or the phone was removed.
- **Error 429:** too many requests, or too many wrong tokens recently. Wait a minute.
- **Error 400:** the body is not valid JSON, or `type` is not one of `drive_start`, `drive_end`, `ping`, or the time is more than 14 days away from now.
