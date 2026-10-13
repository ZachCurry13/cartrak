# Backup, restore and security

## Backup
**More, Backup** (admins) downloads a copy of everything CarTrak stores.
- **Small backup:** everything except the raw copy of every Ford reading. Trips, charging, places, settings, accounts, the charts and the GPS points are kept. The Car data history for old readings will be empty.
- **Full backup:** everything, including the raw readings. This is large on an install that has run for a long time (roughly 10 GB a year at the default polling, and CarTrak never deletes readings).

Making a backup needs free space in the data folder about the size of the copy for a few moments, and the app is slow while it runs. Sign-in sessions are never included, so a backup cannot be used to get into a running CarTrak. It does contain your Ford sign-in, account password hashes and notification keys, so keep it private, like a password.

## Restore
1. Stop the CarTrak app (on TrueNAS, Stop).
2. In the data folder, delete `cartrak.db-wal` and `cartrak.db-shm` if they exist, and replace `cartrak.db` with your backup (renamed to `cartrak.db`).
3. Start CarTrak. Everyone has to sign in again.

## Security
**More, Security** (admins) has two lists:
- **Signed in now:** every phone and browser that is signed in, with where it signed in from, and a button to sign it out. Use it for a lost phone or a device you do not recognise.
- **Sign-in log:** the last 100 sign-in attempts, newest first, with the name that was tried, whether it worked, and the device and address. Many failures in a row, or a name you do not know, is worth a look. Passwords are never stored.

The address comes from the connection (or from Cloudflare when CarTrak is reached through a tunnel).
