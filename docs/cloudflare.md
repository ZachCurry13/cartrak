# Remote access with a Cloudflare tunnel

A tunnel lets your phone reach CarTrak from anywhere without opening a port
on your router. The `cloudflared` container in the TrueNAS YAML makes an
outbound connection to Cloudflare; nothing connects in.

## Create the tunnel
1. Cloudflare dashboard -> **Zero Trust** -> Networks -> **Tunnels** -> Create a tunnel -> **Cloudflared**.
2. Name it (for example `cartrak`). On the install screen, copy the **token** (the long string after `--token`). Put it in `TUNNEL_TOKEN` in the YAML.
3. Add a **public hostname**:
   - Subdomain/domain: for example `cartrak` on your domain.
   - Service type **HTTP**. URL: `http://localhost:31000` if the tunnel app uses Host Network, otherwise `http://<your TrueNAS address>:31000` (use your own host port). With the one-file YAML install the URL is `http://cartrak:8080`.
4. Save. The hostname is now `https://cartrak.yourdomain.com`.

## Use that hostname for Ford
The Ford redirect address is `https://cartrak.yourdomain.com/ford/callback`.
It must be exactly what you registered in your Ford developer app, and in
`FORD_REDIRECT_URL`.

## Add a second lock (recommended)
CarTrak has its own login, but the data includes your car's location, so add
Cloudflare Access too:
1. Zero Trust -> Access -> Applications -> Add -> **Self-hosted**, your hostname.
2. Policy: **Allow**, include **Emails** (yours and family members').
3. Login method: One-time PIN.

One thing to know: the Ford redirect is a normal browser visit that you make
while signed in, so Access does not get in the way of linking.

## Using it
CarTrak trusts Cloudflare's `CF-Connecting-IP` header for its login lockout, and
marks its cookie Secure when it sees HTTPS from the tunnel. Do not publish the
container's port 8080 to the internet directly.
