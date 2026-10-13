# Remote access with a Cloudflare tunnel

A tunnel gives CarTrak a public **https** address without opening a port on your
router: CarTrak makes an outbound connection to Cloudflare, and nothing connects in.
You need it for two things:
- using CarTrak from your phone away from home, and
- linking Ford **automatically**: Ford's sign-in service only accepts a plain `http://`
  redirect for `localhost`. Any other address (such as `http://<your NAS>:<port>`)
  must be `https://`, so it has to go through a tunnel. (Without a tunnel you can
  still link with the paste step; see [Ford setup](ford-setup.md).)

## Option 1 (recommended): inside CarTrak
CarTrak can run the Cloudflare connector for you. The official Docker image includes it.

1. In Cloudflare: **Zero Trust -> Networks -> Tunnels -> Create a tunnel -> Cloudflared**. Name it (for example `cartrak`).
2. Copy the **token** Cloudflare shows (the long string after `--token`; pasting the whole command also works).
3. Add a **public hostname**, for example `cartrak.yourdomain.com`. Service type **HTTP**, URL `localhost:<port>` where `<port>` is the port CarTrak listens on **inside** the container (`8080` unless you changed `CARTRAK_ADDR`). Do not use the TrueNAS host port here: the connector runs inside the same container.
4. In CarTrak open **More -> Remote access (Cloudflare tunnel)**. Paste the token, enter the public hostname, tick **Turn on remote access**, and press **Save & connect**.
5. The status changes to **Connected**. Open `https://cartrak.yourdomain.com` to check.
6. For Ford: press **Use as Ford redirect URL** on that page, and register the same address
   (`https://cartrak.yourdomain.com/ford/callback`) as a redirect URI on your Ford app, if Ford lets you add one.
   Then **More -> Connect Ford** finishes by itself.

The token is stored on the server only and is never shown again. The connector log
(on the same page) hides it. You can also set `CARTRAK_TUNNEL_TOKEN`,
`CARTRAK_PUBLIC_ADDRESS` and `CARTRAK_REMOTE_ENABLED=true` in `config.yaml` or the environment;
those show as locked on the page. If the connector is missing from your build the page says so.

## Option 2: a separate tunnel app
Run `cloudflare/cloudflared` as its own app (command `tunnel --no-autoupdate run`, environment
variable `TUNNEL_TOKEN`). The public hostname's service URL is then
`http://<your TrueNAS address>:<host port>` (or `http://localhost:<host port>` if that app uses Host Network).
With the one-file YAML install in the same compose project it is `http://cartrak:8080`.

## Add a second lock (recommended)
CarTrak has its own login, but the data includes your car's location, so add Cloudflare Access too:
1. Zero Trust -> Access -> Applications -> Add -> **Self-hosted**, your hostname.
2. Policy: **Allow**, include **Emails** (yours and family members').
3. Login method: One-time PIN.

Ford's redirect happens in your own signed-in browser, so Access does not get in the way of linking.

## Notes
- CarTrak trusts Cloudflare's `CF-Connecting-IP` header for its login lockout and marks its cookie Secure over HTTPS (`CARTRAK_TRUST_PROXY`, default on).
- Do not publish CarTrak's own port to the internet directly.
- The connector needs outbound access to Cloudflare (port 7844, TCP and UDP).
