# Troubleshooting

**The app will not start / "permission denied" in the logs.**
The `data` and `config` folders must be writable by UID 568 (the `apps` user). Fix the dataset permissions, then Start again.

**"Not linked" on the dashboard.**
Open More -> Connect Ford. If Ford shows an error about the redirect, the redirect URI registered with Ford must match `FORD_REDIRECT_URL` exactly, including `https://` and the path `/ford/callback`.

**Linking works but no data appears.**
Check More -> Ford connection for the last error. Ford's API may return fewer fields than its dictionary lists, or the vehicle may not be authorized for your app. The Data page shows what actually arrived.

**Trips are labelled wrong.**
Check `CARTRAK_TZ`: rules use that time zone. After changing rules press "Re-apply to past trips". Trips you labelled by hand are never changed.

**No update banner.**
Update checks can be off (More -> About), the container may have no internet, or you are already on the newest version. Pre-release/`main` builds never show update notices.

**I forgot the admin password.**
Set `CARTRAK_ADMIN_USER` (your admin name), `CARTRAK_ADMIN_PASS` (a new password) and `CARTRAK_RESET_ADMIN=1`, restart the app once, sign in, then remove those three settings and restart again. Everyone signed in as that user is logged out.

**Phone shows an old version after updating.**
Close the installed app fully and reopen it. If it persists, clear the site data once.

**Cloudflare error 502.**
The tunnel's service URL must point at CarTrak: `http://localhost:<host port>` when the tunnel app uses Host Network, or `http://<TrueNAS address>:<host port>`. (`http://cartrak:8080` only works with the one-file YAML install, where both containers are in the same app.)

**A setting is greyed out ("locked") in Admin settings.**
It is set by an environment variable or `config.yaml`, which win over the Admin page. Change it there, or remove it there to edit it in the app.

Still stuck? Open an issue at https://github.com/ZachCurry13/cartrak/issues. Do not include your Ford secret or tokens.
