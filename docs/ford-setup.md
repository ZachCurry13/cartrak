# Ford developer app setup

CarTrak reads your car through Ford's **FordConnect 2.0** API. That needs an app
registered with Ford. Ford controls this program and has changed who can get
access over time, so check Ford's developer site for the current steps and terms.


## You need your own Ford developer app
CarTrak does not include, share or proxy any Ford credentials. **Every person who
installs CarTrak registers their own app on Ford's developer site** and enters
that app's client ID and secret in their own CarTrak. Nobody uses anyone else's
credentials, including the author's.

- A Ford developer registration is made for a stated purpose. If you registered
  as **personal use**, use it only for your own vehicle(s). Read Ford's developer
  terms before sharing access or running CarTrak for other people.
- Your redirect URI is specific to your own address (`https://<your hostname>/ford/callback`),
  so each installation registers its own.
- Ford decides who may get access to its APIs and can change this at any time.
  CarTrak cannot grant access.

## What you need from Ford
- **Client ID** and **Client secret** for your app
- A **redirect URI** registered for your app: `https://<your hostname>/ford/callback`

That is all CarTrak needs for FordConnect 2.0. Enter them in CarTrak under
**More -> Admin settings** (easiest), or put them in `config.yaml` or environment
variables; see [Configuration](configuration.md). Treat the secret like a password.
Do not commit it, post it, or paste it into chat.

(The older FordConnect 1 API also needed an Application-Id. If your app was
registered for that older API, set the provider to `fordconnect1`; see Configuration.)

## Recommended: use a separate Ford account for CarTrak
Owners report that Ford has sometimes disabled a FordPass account after
third-party tools were connected to it. This is anecdotal, not published Ford
policy, but the safeguard is cheap, so we recommend it:

1. Create a **second FordPass account** with its own email address, used only for CarTrak.
2. From your **main** account, give that second account access to the vehicle
   (FordPass has a way to share or add another user to a vehicle; the wording
   and steps vary by app version, so check Ford's help).
3. Sign in with the **second** account when you press **Connect Ford** in CarTrak.

That way, if Ford ever restricts the account, your main FordPass login, your
vehicle ownership and your phone key are not affected. If the second account
cannot see the vehicle, finish step 2 first and then link again.


## Redirect address: automatic or paste
Ford sends your browser back to a **redirect URI that you registered on your Ford
app**, and CarTrak must ask for exactly that address. There are two setups:

| You registered | Set the redirect URL in CarTrak to | How linking finishes |
|---|---|---|
| `http://localhost` (common for personal apps) | exactly `http://localhost` | After you sign in, the page that opens (on your own device) shows an error or nothing: that is expected. Copy the whole address from the browser's address bar (it contains `code=…`) and paste it into **More -> Connect Ford -> step 2**. |
| `https://<your hostname>/ford/callback` (needs a [Cloudflare tunnel](cloudflare.md) or another public address) | exactly that address | Automatic: you land back in CarTrak and you are connected. |

Admin settings has buttons for both. If Ford shows `redirect_uri_mismatch` (or
`AADB2C90006`, "not registered"), the address CarTrak sent is not one registered
on your Ford app: fix the redirect URL in Admin settings so it matches exactly,
including `http` or `https` and any path.

## Linking your car
1. Open CarTrak as admin -> **More** -> **Connect Ford** and press **Open Ford sign-in** (use your separate CarTrak account, see above).
2. Ford's own page opens. Sign in with the FordPass account you set up for CarTrak and approve the vehicle.
3. Finish as described under "Redirect address" above (automatic, or paste the address). CarTrak stores a refresh token and starts reading.

CarTrak never sees your Ford password. If Ford expires the link (for example after
a long period), More shows "not linked": press **Reconnect Ford**.

**If the sign-in page does not open correctly:** Ford's FordConnect 2.0 sign-in
address is not listed in the developer collection, so CarTrak uses the standard
pattern for Ford's login service. If Ford's documentation gives a different
sign-in (authorize) address, paste it into **Admin settings -> Ford sign-in
address (advanced)**.

## Check the connection (Ford diagnostics)
**More -> Admin settings -> Ford diagnostics -> Test Ford connection** calls each
Ford endpoint once and shows whether it answered (HTTP 200 is good). **Copy
responses** copies the answers with **location and VIN values removed**, which is
safe to paste into a bug report. Tokens are never included.

## Which Ford data CarTrak reads
CarTrak asks FordConnect 2.0 for:

| Endpoint | How often | What it is |
|---|---|---|
| `telemetry` | every reading (30 s driving, 2 min charging, 10 min parked; faster in Road trip mode) | the live vehicle data |
| `garage` | every 30 min | your vehicles |
| `vehicle-health/alerts` | every 30 min | health alerts |
| `wallbox` | every 30 min | home charger |
| `electric/departure-times` | every 30 min | scheduled departures |
| `electric/charge-schedules` | every 30 min | charge schedules |

Everything that comes back is stored (raw and parsed) and shown on the **Data** page.
The `fccs` (charging station activity) endpoint needs a station id and a date range,
so it is not read automatically yet. What your app is allowed to see can be narrower
than Ford's data dictionary; an endpoint that refuses shows as an error in the
diagnostics and is recorded, but never stops the other data.

## Be considerate
Lowering the polling intervals may breach Ford's rate limits. If Ford answers
"too many requests", CarTrak waits as long as Ford asks.

## Not affiliated
CarTrak is an independent project and is not affiliated with or endorsed by Ford Motor Company.
