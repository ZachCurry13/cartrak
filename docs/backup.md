# Backup, restore and security

## Backup
**More, Backup** (admins) downloads a copy of everything CarTrak stores.
- **Small backup:** everything except the raw copy of every Ford reading. Trips, charging, places, settings, accounts, the charts and the GPS points are kept. The Car data history for old readings will be empty.
- **Full backup:** everything, including the raw readings. This is large on an install that has run for a long time (roughly 10 GB a year at the default polling, and CarTrak never deletes readings).

Making a backup needs free space in the data folder about the size of the copy for a few moments, and the app is slow while it runs. Sign-in sessions are never included, so a backup cannot be used to get into a running CarTrak. It does contain your Ford sign-in, account password hashes and notification keys, so keep it private, like a password.

## Restore
**More, Backup, Restore a backup** (admins) does it from the app:
1. Choose the backup file you downloaded and press **Upload and check**. CarTrak checks that it is a healthy CarTrak database and shows what is in it: how many trips, charging sessions and readings, the dates, and the accounts. Nothing has changed yet.
2. Type **RESTORE** and press **Restore and restart**. CarTrak restarts, puts the backup in place, and the page reloads when it is back (usually under a minute). If it does not come back, press Start for the app in TrueNAS.

Your current data is not deleted. It is kept in the data folder as `cartrak.db.before-restore-<time>`, so you can undo a mistake by swapping it back, and delete it when you are happy. Everyone is signed out, and signs in with the accounts and passwords from the backup.

**Large backups:** uploads through a Cloudflare tunnel are limited to 100 MB on the free plan. For a larger backup, restore from your home network, or copy the file into the `restore` folder inside CarTrak's data folder; it then appears on the page with a **Check** button.

**By hand, if the app will not start:** stop CarTrak, delete `cartrak.db-wal` and `cartrak.db-shm` if they exist, replace `cartrak.db` in the data folder with the backup (renamed to `cartrak.db`), and start CarTrak.

## Security
**More, Security** (admins) has two lists:
- **Signed in now:** every phone and browser that is signed in, with where it signed in from, and a button to sign it out. Use it for a lost phone or a device you do not recognise.
- **Sign-in log:** the last 100 sign-in attempts, newest first, with the name that was tried, whether it worked, and the device and address. Many failures in a row, or a name you do not know, is worth a look. Passwords are never stored.

The address comes from the connection (or from Cloudflare when CarTrak is reached through a tunnel).
