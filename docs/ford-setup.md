# Ford developer app setup

CarTrak reads your car through Ford's **FordConnect** API. That needs an app
registered with Ford. Ford controls this program and has changed who can get
access over time, so check Ford's developer site for the current steps and terms.

## What you need from Ford
- **Client ID** and **Client secret** for your app
- **Application-Id** (sent with every API call)
- A **redirect URI** registered for your app: `https://<your hostname>/ford/callback`

Put them in `config.yaml` (or the matching environment variables); see
[Configuration](configuration.md). Treat the secret like a password. Do not
commit it, post it, or paste it into chat.


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

## Linking your car
1. Open CarTrak as admin -> **More** -> **Connect Ford** (use your separate CarTrak account, see above).
2. Ford's own page opens. Sign in with the FordPass account you set up for CarTrak and approve the vehicle.
3. You are sent back to CarTrak. It stores a refresh token and starts reading.

CarTrak never sees your Ford password. If Ford expires the link (for example after
a long period), More shows "not linked": press **Reconnect Ford**.

## What data is available
CarTrak stores every field the API returns and shows the common ones on the
dashboard. Ford's FordConnect data dictionary groups data as: charging, driving,
EV/battery, trip, vehicle data/health/location/security. What your car and your
app are actually allowed to return can be narrower than the dictionary; the
**Data** page shows exactly what came back in the latest reading.

## Be considerate
CarTrak polls every 30 s while driving, 2 min while charging and 10 min when
parked. Lowering those numbers may breach Ford's rate limits.

## Not affiliated
CarTrak is an independent project and is not affiliated with or endorsed by Ford Motor Company.
