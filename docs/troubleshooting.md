# Troubleshooting

**Start here: More -> Status.**
It checks the whole setup (Ford settings, Ford link, last reading, remote access, updates) in plain language, and each problem has a button that opens the right page. The same result shows as a strip on Home. It never shows keys or passwords.

**The app will not start / "permission denied" in the logs.**
The `data` and `config` folders must be writable by UID 568 (the `apps` user). Fix the dataset permissions, then Start again.

**Ford says `redirect_uri_mismatch` / AADB2C90006, or I land on a "Not Found" page.**
Open More -> Connect Ford: CarTrak now asks Ford which of your addresses are registered and switches to one Ford accepts, showing the result for each. If it says none are registered, add the address (the exact text shown) under Account Linking Info in the Ford developer portal, then reload that page.

If it still fails, the details below apply.
The redirect URL CarTrak sent is not registered on your Ford app. If you registered `http://localhost`, set the redirect URL in Admin settings to exactly `http://localhost`; after signing in, the page that opens will not load (that is expected), so copy its address from the address bar and paste it under More -> Connect Ford, step 2. If you registered a CarTrak address, enter that instead. See [Ford setup](ford-setup.md#redirect-address-automatic-or-paste).

**"Not linked" on the dashboard.**
Open More -> Connect Ford. If Ford shows an error about the redirect, the redirect URI registered with Ford must match `FORD_REDIRECT_URL` exactly, including `https://` and the path `/ford/callback`.

**Ford shows an error, "Not Found" or "login service unavailable" when I press Connect Ford.**
Run **More -> Admin settings -> Ford diagnostics** after linking to see which endpoints answer. Before linking, check that the client ID and secret are right and that the redirect URL matches the one registered with Ford exactly. If Ford documents a different sign-in address, put it in "Ford sign-in address (advanced)". 
CarTrak 0.5 and newer use FordConnect 2.0; earlier versions used the older API and may not work with a FordConnect 2.0 app, so update first. Ford's own login page is also sometimes unavailable; try again later.

**Ford says AADB2C90075 / "RESTApiCallForUserInfo".**
This came from CarTrak 0.8.0 and earlier, which started the sign-in at the wrong Ford address. Update to 0.8.1 or newer and press Connect Ford again. (Details are kept below in case it ever comes back from a different cause.)
That is a problem inside Ford's own sign-in (its user-info step). Sign in with the FordPass account that owns the car or has had it shared to it, make sure that account has completed FordPass sign-up, and try again. If it repeats, send the exact message to support. If Ford documentation gives a specific sign-in link for FordConnect 2.0, paste it into Admin settings -> Ford sign-in address, using {client_id}, {redirect_uri}, {state} and {scope} where those values belong.

**Status says "Ford's servers are having trouble" or the last update failed with HTTP 500 (or 502, 503, 504).**
Ford's own servers answered with an error; nothing is wrong with your setup, your saved data is safe, and CarTrak tries again by itself (after a minute, then 2, 4 and 8 minutes, at most every 10). The message includes what Ford said. After a server or router restart Ford can answer this way for a while until the car's data is available again. If it lasts more than a few hours, check that the FordPass app shows your car, then open More, Settings, **Ford diagnostics**, press Test Ford connection and Copy responses to see each endpoint's answer. Open More, Ford connection and press Sync now to try again right away.

**Ford diagnostics shows "Rate limited" (HTTP 429).**
Ford limits how often an app may ask (about one request per endpoint every 30 seconds). CarTrak waits and retries by itself. Wait a minute, then press Test Ford connection again; it reuses recent answers so it does not make things worse. Pressing Sync now repeatedly does not help.

**Linking works but no data appears.**
Check More -> Ford connection for the last error. Ford's API may return fewer fields than its dictionary lists, or the vehicle may not be authorized for your app. The Data page shows what actually arrived.

**Trips are labeled wrong.**
Check `CARTRAK_TZ`: rules use that time zone. After changing rules press "Re-apply to past trips". Trips you labeled by hand are never changed.

**No update banner.**
Update checks can be off (More -> About), the container may have no internet, or you are already on the newest version. Pre-release/`main` builds never show update notices.

**I forgot the admin password.**
Set `CARTRAK_ADMIN_USER` (your admin name), `CARTRAK_ADMIN_PASS` (a new password) and `CARTRAK_RESET_ADMIN=1`, restart the app once, sign in, then remove those three settings and restart again. Everyone signed in as that user is logged out.

**Phone shows an old version after updating.**
Close the installed app fully and reopen it. If it persists, clear the site data once.

**Cloudflare error 502.**
The tunnel's service URL must point at CarTrak: `http://localhost:<host port>` when the tunnel app uses Host Network, or `http://<TrueNAS address>:<host port>`. (`http://cartrak:8080` only works with the one-file YAML install, where both containers are in the same app.)

**The Live page says "Reconnecting…".**
The live feed is a long-lived connection. It works through a Cloudflare tunnel; if it keeps dropping, check that nothing in front of CarTrak buffers or times out streaming responses, and that the phone has signal. CarTrak sends a heartbeat every 25 seconds and the page reconnects by itself.

**Live view shows "Ford asked CarTrak to slow down".**
Ford limited how often the app may ask. CarTrak waits as long as Ford requests. Turn off Road trip mode or raise `CARTRAK_POLL_LIVE_SEC`.

**A setting is grayed out ("locked") in Admin settings.**
It is set by an environment variable or `config.yaml`, which win over the Admin page. Change it there, or remove it there to edit it in the app.

Still stuck? Open an issue at https://github.com/ZachCurry13/cartrak/issues. Do not include your Ford secret or tokens.

## The Debug report, for one car or every car

**More, Debug report** makes a text report of the car on screen for the last 6 hours to 14 days. If you administer the whole installation and have more than one car, pick any car from the list (yours or, for an administrator of the whole installation, anyone's), or choose **Every car, together** to get one report with each car under its own heading. It lists the readings, trips, charging, polling, alerts and the settings that shape behavior, and has no positions, people's or cars' names, keys or VIN. Read it before sending it and delete anything you would rather not share.

## Field check

**More, Field check** is a short list of things to try in the car and on your phone, each with what should happen. Tap Works, Problem or Skipped, add a note, then Copy results to paste them to whoever is helping. The top of the page also reports what the app can see for itself: the version, whether alerts are set up on this device, how quick the Savings page is, whether live gas prices are on for this car, and how many work stops were found. Your answers are kept on this device only, until you tap Start over.
