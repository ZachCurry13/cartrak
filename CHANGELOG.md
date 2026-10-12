# Changelog

## Unreleased

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
