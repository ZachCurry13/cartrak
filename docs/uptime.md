# Knowing when CarTrak is down

A program cannot tell you it has stopped: it is not running to say so. So there are two parts, one built in and one that needs an outside watcher.

## Built in: "CarTrak is back online"
Every minute CarTrak notes that it is running. When it starts and finds it was last running more than **five minutes** ago, it sends a notification, **CarTrak back online**, saying how long it was gone and since when. A restart to update takes about a minute or two, so that does not trigger it. It uses the same notifications as everything else (More, Notifications, where it can be switched off per phone).

## Outside: an uptime monitor
To hear that CarTrak is **down**, something outside has to notice. Under **More, Integrations, Uptime monitor**, paste an address from a monitoring service. CarTrak asks that address every five minutes. When the asking stops, the service tells you (by email, a push to your phone and so on, as you set it up), and it tells you again when the asking starts again.

A free way to do it:
1. Make an account at a heartbeat-monitoring service such as [Healthchecks.io](https://healthchecks.io) (there is a free tier; check its page for the current limits).
2. Create a check with a **period of 5 minutes** and a **grace time of 5 minutes**, and add the channel you want the alert on (email, Telegram, a push app).
3. Copy the check's ping address into CarTrak. Tap **Test connection** to see it answer.

What this catches: CarTrak stopping, the server or its power going out, or the server losing internet (the asking stops either way).

What it does not catch: CarTrak running fine while the **Cloudflare tunnel** is down, because the asking goes out directly. For that, also add a monitor from a service that opens your public address from outside (an uptime service such as UptimeRobot has a free tier) and point it at the address CarTrak is reached at (More, Remote access).

The address stays on the server and is never shown again once saved. CarTrak sends nothing else to it: only a plain request every five minutes.
