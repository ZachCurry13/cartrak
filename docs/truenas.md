# Install on TrueNAS SCALE (Custom App form)

This walks through TrueNAS's **Install Custom App** screen field by field, in the
order the form shows its sections. Menu and field names can differ a little
between TrueNAS versions; the values that matter are in **bold**. Prefer YAML?
See [the YAML option](#option-install-from-yaml-instead) at the end.

You need TrueNAS SCALE with Apps enabled. For access from your phone away from
home you also need a Cloudflare account with a domain ([Cloudflare tunnel](cloudflare.md)).
To try CarTrak first without Ford, you can add the variable `CARTRAK_PROVIDER` = `demo`.

## 1. Create the two folders
CarTrak keeps everything in two folders you map from your pool:

| Mount path in the container | What it holds | Back it up? |
|---|---|---|
| `/data` | the database: every reading, trip, charge, rule, user and your Ford settings | **Yes** |
| `/config` | an optional `config.yaml` (settings as a file) | Yes, keep it private |

1. Storage -> Datasets: create a dataset such as `apps/cartrak`, with two folders (or child datasets) inside it: `data` and `config`.
2. Permissions for both: owner user **apps** (UID 568), owner group **apps**, Read/Write/Execute for the owner. The container runs as 568; without this it cannot write.

## 2. Apps -> Discover Apps -> Custom App
Fill the form top to bottom. Anything not mentioned can stay at its default.

### Application name
- **Application Name:** `cartrak`
- Version: leave as is.

### Image Configuration
- **Repository:** `ghcr.io/zachcurry13/cartrak`
- **Tag:** `latest`
- **Pull Policy:** choose the option that **always pulls** the image (the default, "only if not present", will never fetch updates). This is what makes Stop then Start update CarTrak.

### Container Configuration
- Timezone: set yours if the form has this field.
- **Environment Variables:** none are required. Optionally add `CARTRAK_TZ` = your time zone name (for example `America/Chicago`). Ford credentials and the time zone can instead be entered later inside CarTrak (see step 4), so you do not need to add them here.

### Security Context Configuration
- Leave **Privileged** off.
- **Run As User:** `568` and **Run As Group:** `568`.

### Network Configuration
- Leave **Host Network** off.
- **Add a Port:**
  - **Host Port:** any free port, for example `31000` (pick one that no other app on your TrueNAS uses)
  - **Container Port:** `8080`
  - **Protocol:** `TCP`

### Portal Configuration (optional but handy)
Adds a **Web UI** button to the app in TrueNAS: enable it, protocol `HTTP`, use the node IP, **Port** = the host port you chose, path `/`.

### Storage Configuration
Add two entries, both of type **Host Path**:
1. **Host Path:** your `data` folder, **Mount Path:** `/data`
2. **Host Path:** your `config` folder, **Mount Path:** `/config`

(Leave "Enable ACL" off unless you have set ACLs yourself.)

### Labels, Resources
Leave defaults. If you want limits, 1 CPU and 512 MB of memory is plenty.

Press **Install**. The first start downloads the image.

## 3. First run
1. Open the app (the **Web UI** button, or `http://<your TrueNAS address>:31000`).
2. Create the admin account (10+ character password).
3. The **Getting started** page appears. Set up a [separate FordPass account](ford-setup.md#recommended-use-a-separate-ford-account-for-cartrak) first, as recommended.

## 4. Remote access (Cloudflare tunnel)
Easiest: in CarTrak open **More -> Remote access (Cloudflare tunnel)**, paste your tunnel token and public hostname, and turn it on. CarTrak runs the connector itself, so no second app is needed. Details: [Cloudflare tunnel](cloudflare.md). The separate-app steps below are the alternative.

Install the tunnel as its **own second Custom App** (same form):
- **Application Name:** `cloudflared`
- **Repository:** `cloudflare/cloudflared`, **Tag:** `latest`, Pull Policy: always
- **Command** (add three entries, in order): `tunnel`, `--no-autoupdate`, `run`
- **Environment Variable:** `TUNNEL_TOKEN` = the token from Cloudflare
- **Network:** turn **Host Network** on, so the tunnel can reach CarTrak at `http://localhost:31000`
  (otherwise use `http://<your TrueNAS address>:31000` as the service URL)

Full steps, including the Cloudflare side and a second login lock: [Cloudflare tunnel](cloudflare.md).
Set the redirect URL in step 4 to `https://<your hostname>/ford/callback` (the same address you register with Ford).

## 5. Enter your Ford settings and connect
In CarTrak: **More -> Admin settings**. Fill in the Ford client ID, client secret,
Application-Id, the redirect URL and your time zone, then **Save**. They apply
immediately and are stored in the database (`/data`). The secret is never shown
again or sent to the browser. Then press **Connect Ford**.

Prefer files? Put the same values in `/config/config.yaml` (CarTrak writes a
commented `config.example.yaml` there on first start) or in environment
variables. Those win over the admin page and show as "locked" there. See
[Configuration](configuration.md).

## 6. Install on your phone
Android (Chrome): menu -> **Install app**. iPhone (Safari): Share -> **Add to Home Screen**.

## Updating
CarTrak shows an "Update available" banner. In TrueNAS open the app, **Stop**, then **Start**.
With Pull Policy set to always pull, Start downloads the newest release. Your folders are never changed.
See [Updating](updating.md).

## Moving or restoring
Stop the app, copy the `data` (and `config`) folder, point the new install at them, start.

## Option: install from YAML instead
Apps -> Discover Apps -> Custom App -> **Install via YAML**, then paste
[`deploy/truenas-compose.yaml`](../deploy/truenas-compose.yaml). It contains CarTrak and the tunnel in one app.

## Trouble?
[Troubleshooting](troubleshooting.md).
