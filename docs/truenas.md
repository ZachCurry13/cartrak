# Install on TrueNAS SCALE (Custom App)

This guide installs CarTrak as a TrueNAS Custom App from YAML, with a Cloudflare
tunnel for access from your phone. Menu names can differ slightly between
TrueNAS versions.

You need: TrueNAS SCALE with Apps enabled, a Cloudflare account with a domain,
and a Ford developer app (see [Ford setup](ford-setup.md), including the recommendation to use a separate FordPass account for CarTrak). To try CarTrak first
without Ford, use `CARTRAK_PROVIDER: demo`.

## 1. Create the two folders
CarTrak keeps everything in two folders that you map from your pool:

| Folder in the container | What it holds | Back it up? |
|---|---|---|
| `/data` | the database: every reading, trip, charge, rule and user | **Yes** |
| `/config` | `config.yaml` with your settings and Ford credentials | Yes, keep it private |

1. Storage -> Datasets: create a dataset such as `apps/cartrak`, then two folders or child datasets inside it: `data` and `config`.
2. Permissions for both: owner user **apps** (UID 568), owner group **apps**, Read/Write/Execute for the owner. The container runs as 568, so without this it cannot write.

## 2. Cloudflare tunnel
Follow [Cloudflare tunnel](cloudflare.md) and keep the **tunnel token** and your
hostname (for example `cartrak.example.com`) for the next step.

## 3. Install
1. Apps -> Discover Apps -> **Custom App** -> **Install via YAML**.
2. Name it `cartrak` and paste [`deploy/truenas-compose.yaml`](../deploy/truenas-compose.yaml).
3. Replace each `CHANGE-ME`: the two host paths under `volumes`, your time zone, and the tunnel token.
4. Save. The first start downloads the image.

When it starts, CarTrak writes `config.example.yaml` into your `config` folder.
Copy it to `config.yaml`, fill in your Ford details and restart the app. See
[Configuration](configuration.md). (Environment variables in the YAML work too and win over the file.)

## 4. First run
1. Open `https://cartrak.example.com` on your phone or computer.
2. Create the admin account (10+ character password).
3. Before connecting Ford, set up a separate FordPass account as described in [Ford setup](ford-setup.md).
4. The **Getting started** page walks you through connecting Ford, installing on your phone, adding a work rule, and adding family.

## 5. Install on your phone
Android (Chrome): menu -> **Install app**. iPhone (Safari): Share -> **Add to Home Screen**.

## Updating
See [Updating](updating.md). In short: CarTrak shows an "Update available"
banner; in TrueNAS open the app, **Stop**, then **Start**. The YAML uses
`pull_policy: always`, so Start downloads the newest release. Your folders are untouched.

## Restoring or moving
Stop the app, copy the `data` and `config` folders, point the new install at them, start. That is the whole migration.

## Trouble?
See [Troubleshooting](troubleshooting.md).
