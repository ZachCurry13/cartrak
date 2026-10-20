# Backup, restore and security

## Backup
**More, Backup** (admins) downloads a copy of everything CarTrak stores.
- **Small backup:** everything except the raw copy of every Ford reading. Trips, charging, places, settings, accounts, the charts and the GPS points are kept. The Car data history for old readings will be empty.
- **Full backup:** everything, including the raw readings. This is large on an install that has run for a long time (roughly 10 GB a year at the default polling, and CarTrak never deletes readings).

Making a backup needs free space in the data folder about the size of the copy for a few moments, and the app is slow while it runs. Sign-in sessions are never included, so a backup cannot be used to get into a running CarTrak. It does contain your Ford sign-in, account password hashes and notification keys, so keep it private, like a password.

## Disk space
**Status** shows how much room is left on the disk CarTrak stores its data on (on the Docker image; it cannot tell on other systems). It turns amber under 5 GB or 8 percent free and red under 1 GB, because a full disk stops CarTrak saving anything. Turn on the **Low disk space** alert under More, Notifications to be told. If you are low, use **Free up space** below, or give the data dataset more room in TrueNAS.

## Free up space (optional)
CarTrak keeps a full raw copy of every reading from Ford, about 85 KB each, roughly 10 GB a year at the default polling. **More, Backup, Free up space** can remove that detail for older readings. You choose how many recent days to keep (30, 90, 180 or 365) and type **TRIM** to confirm.

What is kept: trips, charging, places, journeys, maintenance, the charts and trends, GPS points, and each reading's main numbers (battery, range, odometer, position and so on). What goes: the full raw detail, so the **Car data** page shows a note instead of the long list for those older readings.

It runs in the background in small steps, so CarTrak stays usable, and you can leave the page. It cannot be undone except by restoring a backup, so make a full backup first if you might want the old detail. The database file does not get smaller by itself; the freed space is reused for new readings, so it stops growing for a long while. To make the file itself compact, make a **full backup** (which is compact) and restore it.

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

## Export to spreadsheets
**More, Backup, Export to spreadsheets** downloads one zip with every trip, every charging session, your named places and your service reminders as CSV files, with a short README. Open them in Excel or Sheets. It is for reading your data elsewhere, not a backup, and it cannot be restored. It shows where the car has been, so keep it private. Admins only.

## With several cars

A backup made on a car holds that car only, and no accounts (the people are kept in `core.db`, not in a car's database).
Restoring replaces only that car. Administrators of the installation can also download a zip of the people and every car
under **Backup, Download everything (all cars)**. See [several cars and people](several-cars.md).
