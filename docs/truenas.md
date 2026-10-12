# Running CarTrak on TrueNAS with a Cloudflare tunnel

## 1. Storage
Create a dataset (for example `apps/cartrak`). It holds the SQLite database
and all cached Ford data. Back it up like any other dataset. The container
runs as UID 568 (the TrueNAS `apps` user); give that user write access.

## 2. Cloudflare tunnel
1. In Cloudflare Zero Trust, go to Networks -> Tunnels -> Create a tunnel (Cloudflared).
2. Copy the tunnel token into `TUNNEL_TOKEN` in the YAML.
3. Add a public hostname, for example `cartrak.yourdomain.com`, with service
   `HTTP` and URL `cartrak:8080` (the compose service name).
4. Optional but recommended: add a Cloudflare Access policy for that hostname
   (one-time PIN to your email). CarTrak has its own login too, so you get two layers.

## 3. Ford developer app
In the Ford developer portal, register
`https://cartrak.yourdomain.com/ford/callback` as the redirect URI. Put the
client ID, client secret and Application-Id in the YAML. Treat the secret
like a password; never commit it.

## 4. Install
Apps -> Discover Apps -> Custom App -> Install via YAML, paste
`deploy/truenas-compose.yaml` with your values filled in.

The first GitHub package is private by default. In GitHub, open the
package's settings and set it to **public**, or TrueNAS cannot pull it.

## 5. First run
Open your hostname, create the admin account, then More -> Connect Ford.
Ford's own page asks you to sign in and authorize the vehicle; CarTrak never
sees your Ford password. Add family as read-only viewers under More -> People.

## 6. Install on your phone
Open the site in Chrome (Android) or Safari (iPhone) and choose
"Install app" / "Add to Home Screen".

## Updating
Releases publish `:latest`. Press **Update** on the app in TrueNAS to pull it.
