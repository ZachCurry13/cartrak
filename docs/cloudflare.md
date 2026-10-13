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

## Updating the Cloudflare connector
Cloudflare publishes a new connector (cloudflared) every month or so. The Docker image ships one version, and once a day CarTrak asks GitHub whether a newer one exists (the same switch as the CarTrak update check, under More, About). If there is, **More, Remote access** says so, and the **Cloudflare connector** card on that page has an **Update** button.

What the button does, step by step:
1. Downloads the Linux file for your machine from github.com over https (no other address is allowed, including redirects).
2. Checks its SHA-256 against the checksum GitHub publishes for that file. If GitHub published none, or it does not match, nothing is installed.
3. Runs it once to confirm it works and reports the expected version. If the data folder does not allow running programs, the update stops here.
4. Switches the tunnel to it. Remote access drops for a few seconds. If the tunnel is not connected within 90 seconds, the previous connector is put back, and that version is not offered again.
5. Keeps the two newest connectors in the data folder (the `bin` folder) and uses the new one at every start.

If you are using CarTrak through the tunnel while you update, the page may reload when it reconnects. CarTrak itself cannot be updated from the app on TrueNAS: the whole container is replaced, so update it with Stop, then Start. When CarTrak itself updates to a newer image, it uses the newer of the two connectors.

The **Components** list under More, About shows the versions of CarTrak, the connector, SQLite and Go. The repository also asks Dependabot for a weekly pull request when a library, GitHub Action or base image has a newer version; nothing is released without review.
