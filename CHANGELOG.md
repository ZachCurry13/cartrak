# Changelog

## 0.6.0

### ☁️ Remote access inside CarTrak
- A new Remote access page (More): paste your Cloudflare tunnel token and public address, switch it on, and CarTrak runs the connector for you, with a status and a connector log. The Docker image now includes cloudflared.

### 🔗 Connecting Ford
- Ford sending the code after a # in the address is now handled: the callback is a page that reads the whole address, and the paste box understands it too.
- Getting started now puts remote access before linking Ford, and the Connect Ford page warns when the redirect is a plain http address that Ford will refuse.

## 0.5.1

### 🔗 Connecting Ford
- A new **Connect Ford** page with two steps. If your Ford app was registered with the redirect `http://localhost`, you can now finish linking by pasting the address Ford sends you to. Ford apps registered with CarTrak's own address still connect automatically.
- Clear messages when Ford refuses: the Azure error is explained in plain words, and the "redirect not registered" mistake tells you exactly what to fix.
- The Admin page has buttons to fill in the redirect URL.

## 0.5.0

### 🔌 FordConnect 2.0
- CarTrak now uses Ford's current FordConnect 2.0 API: telemetry (the live vehicle data), your garage, health alerts, home charger, departure times and charge schedules. Version 0.4 and earlier used the older API, which is why connecting could fail.
- You only need your Ford client ID and secret now. No Application-Id.
- A new **Ford diagnostics** tool on the Admin page tests every Ford endpoint and can copy the answers with location and VIN removed, so problems are easy to report.
- An optional "Ford sign-in address" setting, in case Ford documents a different one.
- The older API is still available as the provider `fordconnect1`.
- The docs now spell out that everyone installing CarTrak registers their own Ford developer app; no Ford credentials are included or shared.

## 0.4.0

### 📡 Live and on the road
- A new **Live** tab: a full-screen dashboard of big tiles (charge, range, speed, charging power, this trip) that updates the moment a new reading arrives. It keeps the screen awake and has a Full screen button.
- **Road trip mode** asks Ford for readings more often while you drive, and turns itself off after 12 hours. If Ford asks CarTrak to slow down, it waits as long as Ford says and tells you.

### 📱 Phone, tablet and desktop
- Wider screens get a multi-column home page, multi-column lists, and a sidebar. Phones keep the single column.
- A per-device **Data use** setting (More): Auto, Normal, Live or **Data saver**. Data saver shows the essentials, refreshes every 5 minutes, and downloads the trip route and the full field list only when you tap. It only affects the device you set it on.
- Pages are compressed and refreshes with nothing new cost almost nothing.

### 📚 Docs
- A new guide for the live view, road trip mode and data saver.

## 0.3.1

### 🐛 Fixes
- Connect Ford now opens Ford's real sign-in page. Version 0.3.0 sent you to the wrong Ford address and showed "Not Found".

## 0.3.0

### ⚙️ Settings in the app
- A new Admin settings page (More -> Admin settings) for your Ford client ID, secret, Application-Id, redirect URL and time zone. Changes apply right away and the secret is never shown again. Settings in config.yaml or environment variables still work and win, and show as locked.
- Getting started and the home page send you to Admin settings until the Ford details are filled in.

### 🔒 Safer sign-in settings
- New settings for whether to trust proxy headers (on by default behind Cloudflare) and how many days a sign-in lasts.

### 📚 Docs
- The GitHub front page and a new field-by-field guide follow TrueNAS's Install Custom App form, with a table of every setting. The one-file YAML install is still available.

## 0.2.0

### 👋 Easier to start
- A Getting started guide opens the first time you sign in: connect Ford, install on your phone, add a work rule, add family. It stays available in More.
- The home page reminds you until setup is finished.

### 🔔 Versions and updates
- The version is always shown at the top of the screen and in More -> About.
- CarTrak tells you when a new version is out (you can turn this off) and how to install it. After an update, a note points you to what changed.
- A "What's new" page shows every release's changes inside the app.

### 🗂 Settings and storage
- Settings can live in a `config.yaml` file in a mapped `config` folder, next to the `data` folder, so both survive updates.
- TrueNAS: the YAML now maps both folders and downloads the newest release when you Stop then Start the app.
- A password recovery switch for the admin account.

### 📚 Docs
- Step-by-step guides on GitHub for TrueNAS, Cloudflare tunnels, Ford setup, settings, updating and troubleshooting.
- We now recommend a separate FordPass account for CarTrak, in the guides and in the Getting started page, so your main account is never at risk.

## 0.1.0

### 🚗 Your car's data
- Connects to Ford's FordConnect API: sign in once on Ford's own page, then CarTrak keeps itself updated.
- Dashboard cards for battery, charging, vehicle status, location and tires. A searchable "All data" page shows every field Ford returns.
- A built-in demo car lets you try everything without a Ford account.

### 🧭 Trips and charging
- Trips and charging sessions are found automatically, with distance, battery used and top speed.
- Label trips personal or work yourself, or let rules do it (for example weekdays 8am-5pm = work). Your own choice always wins.
- Work and personal mileage totals, and CSV export.

### 📱 On your phone
- Install it like an app from your phone's browser. Light and dark themes, in a Ford-inspired blue.
- Login with an admin account and read-only family accounts.

### 🛠 Running it
- Docker image, TrueNAS YAML and Cloudflare tunnel instructions.
