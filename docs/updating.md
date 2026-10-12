# Updating CarTrak

## How you find out
- The version is shown at the top right of every page (a gold dot means an update is available) and in **More -> About**.
- About once a day CarTrak asks GitHub's public releases list for the newest version. It sends no vehicle or personal data. Turn it off in More -> About.
- After an update, the app shows a one-time "CarTrak was updated" note linking to **What's new**, the full changelog built into the app.

## TrueNAS
1. Apps -> **cartrak** -> **Stop**.
2. **Start**. The YAML has `pull_policy: always`, so it downloads the newest release before starting.
3. Reload CarTrak on your phone. If the old screen is still showing, close and reopen the app once.

Your `data` and `config` folders are never changed by an update. CarTrak upgrades
its own database on start. Before a big update you can copy the `data` folder as a backup.

## Pinning a version
Replace `:latest` in the YAML with a specific version such as `:0.2.0` to stay on it.
Releases are listed at https://github.com/ZachCurry13/cartrak/releases.

## Docker without TrueNAS
```bash
docker compose pull && docker compose up -d
```
