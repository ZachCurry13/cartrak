# Several cars and several people

CarTrak can follow more than one car, and each person sees only the cars they were given.

## How it is kept apart

- Every car has **its own database**: its readings, trips, charging, places, settings and alerts. One car's data cannot
  appear in another's. The first car keeps the database it always had (`cartrak.db` in the data folder). Cars you add
  live in `cars/<number>/` inside the data folder.
- The **people**, their sign-ins, their notification devices and the list of cars are in a small shared database,
  `core.db`. Signing in once works for every car.
- Each car has its own Ford link. By default every car uses the Ford app (client ID and secret) set on the first car;
  a car can have its own under **More, Settings** instead.

## The first start after updating

On the first start of this version CarTrak makes `core.db`, copies the people, sign-ins and notification devices into
it, and clears the old copy of the accounts from the first car's database. Before it does, it saves a copy of the
database as `backups/before-several-cars-<time>.db` in the data folder. Nothing else changes: you are still signed in,
and everyone keeps seeing the car they saw.

## What people can be

- **Administrator of the installation** (the existing *Admin* role): manages people, remote access and backups of the
  whole installation, owns the first car, and can open any car.
- **Owner of a car**: the person who brought it. They manage it as an administrator would (settings, Fix a day,
  backups, who can see it), and can share it with others as read-only. They cannot manage people or remote access,
  and cannot see other cars.
- **Viewer of a car**: can look but not change.

Under **More, People** an administrator decides, for each person, whether they **see the first car** and whether they
**may add a car of their own**. A person who may add a car sees **More, Cars, Add a car**, gives it a name, and signs in
to their own Ford account to connect it.

## Opening another person's car

An administrator can open any car, for example to help work out a problem. The car's owner sees a note on **More, Cars**
saying that an administrator looked, and when. An administrator is not sent that car's alerts unless they were given
access to it.

## Switching cars

With more than one car, a car selector appears at the top. Each browser tab remembers its own choice, so two tabs can
show two cars. **More, Cars** lists the cars you can open, lets owners rename a car, choose who can see it, or stop
following it (its database stays on disk). Alerts from a car name the car when you follow more than one.

CarTrak follows up to **6 cars**, because each one is polled from Ford on its own.

## Backups

- **Backup** on a car downloads that car's database only, with no accounts in it. Restoring uploads into that car's own
  restore folder and replaces only that car.
- Administrators of the installation also get **Download everything (all cars)**: a zip with `core.db` and each car's
  database. To put it all back by hand, stop CarTrak, replace `core.db` and the car database files, and start it again.
  Applying a restore stops CarTrak for everyone, so only an administrator of the installation can confirm one.

## Phones and links

A phone sends its position with a device token that belongs to one car, so it needs no change. Public share links
likewise belong to the car they were made on.

## Two vehicles on one Ford account
Add a car for each vehicle (**More, Cars, Add a car**) and link the same Ford account to each. On Ford's page, choose the vehicle
for that car. If the account shares more than one vehicle with a car anyway, that car's **Ford connection** page shows **Which
vehicle is this?**: tap **Follow this one** on the right vehicle. A vehicle another car already follows is named, so the same one
is not followed twice. Until you choose, a car follows the first vehicle Ford lists, and never mixes two.

## All cars
With two or more cars, the car picker at the top has **All cars**. **Overview** shows every car side by side (battery or fuel,
range, whether it is driving, charging or parked, and its last trip) and this month together: the distance and trips of all of
them, how much of the driving was electric, the charging cost and each car's share. **This year** is the same for the year, and
**Trips** lists every car's recent trips in one list, each marked with its car. Tap a car or a trip to open it. A person sees only
the cars they were given.
