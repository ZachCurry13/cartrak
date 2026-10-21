# Changelog

## 0.54.0

### 🎅 Holiday voices for alerts (optional)
- On the Notifications page, **Holiday voices** makes the friendly alerts talk in the holiday's voice: Santa at Christmas ("Ho ho ho! Fully charged"), a friendly ghost at Halloween ("Boo-tifully charged"), a leprechaun on St. Patrick's Day, the Easter bunny, Cupid, fireworks on the Fourth of July, Thanksgiving and New Year. Only the words change, never the facts, and never a warning. It is off until you turn it on, and follows you to every device.

### 🏆 A trophy collection
- A new **Trophies** page (Insights, or More) shows every trophy your car can earn, the ones you have in color and the rest **grayed out with how far along you are**, plus **Next up** for the ones closest to done. Trophies that cannot apply are not listed: charging ones for a gas car, the tank ones for an electric car.
- **Fourteen new trophies** from your habits: Marathon day (300 miles in a day), Night rider, Low and slow, Winter warrior, Gentle month, Sunrise patrol, Weekend warrior, Regular (20 visits to one place), Triple threat, Round trip, Holiday cruiser, and 50,000, 100,000 and 150,000 miles on the clock. For charging: Quick charge, Loss detective (enter the charger's kWh on 10 sessions), Plugged-in streak, Gas station skipper. For the tank: Mile muncher (400 miles between fill-ups) and Thrifty fill. Efficiency trophies use only the odometer's own distances, so a corrected trip never counts.
- A milestone the odometer was **already past when CarTrak began** (50,000 miles on a car at 88,000) shows as earned with "Already reached when CarTrak began", not dated at your first trip. Trophies still to come that count distance or trips say **about when you will get them**, at the pace of your last 90 days of driving (once there is a week of it): the odometer milestones go by the odometer.
- **Most visited**: Insights, Highlights has a card with the places you named and the towns you went to most in the last 12 months, counted from the trips that ended there.

## 0.53.0

### ⚡ Home loads much faster
- Home used to ask for its pieces one after another (the battery, status, trips, the week, service, fuel, today, setup…), so a slow connection paid for each one in turn. It now asks for them all at once, and the page's code loads while the first answers travel. Over a slow connection it appears in about the time of two requests instead of a dozen.
- The debug report has a new **Slowest requests** section, so if a page is still slow the next report says which request is behind it.

### 🔎 The debug report shows what was changed by hand
- Each trip says when its distance was **corrected** (by hand, from the road or as the straight line) and what the odometer said. Corrections, category changes, deletions and closing a stuck trip are now logged as they happen, in the report's decisions section.

### 🚘 One car's numbers never shown under another car
- Every answer about a car now says which car it is, and the app refuses (with a message) an answer for a different car than the one on screen, instead of showing it. The remembered "last readings" are filed under the car that was on screen when the request was made, so switching cars while a refresh is on its way can no longer put one car's numbers under another's name. The debug report names the vehicle each car follows (the last six characters of its VIN) and the vehicles on the linked account, so a mix-up can be traced.

### 🧹 Consistency
- Every page now has its title in the same place and style, and the link rows on Home, Insights, journeys and the guide share one style instead of copies of the same inline settings.

### 🐛 Fixes
- A trip whose distance was corrected (by hand or from the estimated road) no longer counts toward **Most efficient drive**, the Hypermiler badge or the fuel economy record: its energy is real but the corrected distance is not the odometer's, which had doubled one drive to 6.27 mi/kWh.

## 0.52.0

### ⛽ Gas and hybrid cars, properly
- A new **Fuel** page (More, and in the bottom bar for a gas or hybrid car). When the fuel gauge jumps up, CarTrak keeps a **fill-up** and asks (an optional **Filled up** alert) for the **gallons** and the **price per gallon**, entered each time.
- CarTrak **works out the size of the tank by itself** from those fill-ups: 6 gallons that took the gauge from 30% to 70% is a 15 gallon tank. It gets better with every fill-up that starts well below full.
- **Trips** show the tank at each end and, once the tank size is known, the **fuel used, the economy (mpg or L/100 km) and the cost**. Older trips get it too, from the readings already kept. A drive with a stop at the pump in the middle is left without a number rather than a wrong one.
- **Insights, Fuel**: what you spent by month and the economy by month. **Records** adds best economy, the farthest between fill-ups, the biggest fill-up and the cheapest gas. **Wrapped** has a **Fuel** slide, and its infographic and the monthly recap speak gallons and dollars instead of kWh.
- A new **Low fuel** alert (15% or less). Home shows fill-ups still waiting for their gallons under **Needs attention**.
- Insights drops the Charging and Battery tabs, the energy flow and the charging achievements for a car that cannot plug in. Liters and a price per liter, for devices set to kilometers.
- The debug report gets a **FUEL** section.

### 📤 Export every car field
- **More, Data, All fields** has **Export every field over time**: a zip of spreadsheets with every field of the newest reading, the fields that never changed, and a table with a row each time anything changed (24 hours, 7, 30 or 90 days). Nothing is cut off, unlike printing the page. Positions and the VIN are left out unless you tick the box, so the file is safe to send. It is also under **Reports** (admins).

### 🧭 A half-circle gauge
- The battery ring on **Home** and **Live** is now a **half-circle gauge** (and the fuel tank is the same gauge for a gas or hybrid car). It takes about 40 px less height, so the chips and Needs attention rise up the screen. **Live** shows the tank, not an empty Charge gauge, for a car with no battery level.

## 0.51.0

### ▶️ Trip replay follows the road
- **Replay this trip** now follows the likely road the trip's map shows, instead of cutting straight across between the car's few reported positions. The car's own positions still set the pace, so it speeds up, slows down and waits where the car really did. Trips with a phone GPS track already played back along the real route and are unchanged.

### 🚙 Two vehicles on one Ford account, sorted out
- A car added for a Ford account that shares several vehicles now follows one that no other car already follows. Before, it followed the first vehicle on the account (so a new Escape could quietly record the F-150's readings and show up as electric). A vehicle you pick on the **Ford connection** page still wins, and the car added first keeps its vehicle.
- A car keeps following its vehicle when Ford lists the account's vehicles in a different order. Before, one reading could come from the other vehicle, with its alerts (an arrival, a low 12 V battery) sent under the wrong car.
- When a car changes which vehicle it follows, it forgets the vehicle type it learned and learns it again.
- A hybrid or gas car that reports no battery level opens Home on its **fuel tank** (level, range, oil life) instead of an empty battery ring.

### 🐛 Smaller fixes
- **Most efficient drive** (and the Hypermiler badge) now only counts drives up to 1.6 times what your car usually gets, so a drive with a stale battery reading at the end no longer shows an impossible figure.
- Percent charts, like **Battery level, day by day**, no longer run their axis up to 120%.
- **Left open** tells you about the same open door at most once every 12 hours, until the car reports it closed. A sleeping car that keeps repeating its last report no longer sends it again and again overnight.
- **Car left unlocked** says when the car last reported its locks if that was a while ago (a car in battery saver mode stops reporting), like Left open already does.
- **Slow fast charge** never judges a session whose end level is not known yet, so a charge that ran to 100% is not called slow.
- The birthday picker only offers days the chosen month has.

## 0.50.0

### 🚙 Several vehicles on one Ford account
- When the Ford account linked to a car shares more than one vehicle (an F-150 and an Escape, say), the car's **Ford connection** page asks **Which vehicle is this?** and lists them, naming any another car already follows. Each car now follows exactly one vehicle; before, two vehicles on one account could have their readings mixed.
- To add the second vehicle: **More, Cars, Add a car**, link the same Ford account, and pick it.

### 🚗🚙 All cars
- With two or more cars, the car picker at the top has **All cars**: every car side by side (battery or fuel, range, what it is doing, the last trip) and the month or the year **together**, with each car's share and how much of the driving was electric. **Trips** lists every car's trips in one list. Tap a car or a trip to open it.

### 🎂 Birthdays
- Set your birthday under **More, Display and data** (just the month and day). On that day CarTrak dresses up for you: party colors, a cake, confetti and **Happy birthday!** on Home, with a happy birthday alert on your phone in the morning. Only you see it, on every device you sign in on.

### 🎃 Holiday alerts
- Around a holiday, friendly alerts start with its emoji, such as **🎃 Charging complete** or **🎄 Trip ended**, and on the day itself they wish you a happy holiday. Home says so too. Warnings always stay plain. Turn it off with **Holiday touches** on the Notifications page.

### 🧭 Getting started, brought up to date
- New optional steps for adding another car (even one shared by the same Ford account) and your birthday. Places mentions dealerships and tire shops, and the electricity price step and search now open **Charging prices**, where those settings moved.

### 🔔 Notifications that save
- Alert choices no longer save on every tap. Tick what you want, then **Save alert choices** (a Save bar also appears at the bottom while there are changes). The page then shows what was really saved, and a phone that lost its registration is registered again with your choices.
- When a request hits a bug, CarTrak now answers with a message instead of a dropped connection (which showed as "HTTP 502"), and notes it in a new **Server errors** part of the debug report.

### 🇺🇸 US English
- The app and the guides now use US spelling throughout (color, kilometers, meters, center, canceled). Searching for "tyre" still finds tires.

### 🎃 Smaller fixes
- When CarTrak does not answer for a moment (for example while it restarts after an update), a save says so in words instead of "HTTP 502", and a page tries once more and then shows what the phone kept.
- On a narrow phone with the car picker showing, the seasonal ornament steps aside so the header stays on one line.

## 0.49.0

### 🔧 Service visits
- Mark a place as a **dealership, tire shop or repair shop** (More, Places). When the car stays there 20 minutes or more, CarTrak asks after you leave: **was it serviced?** Tap what was done (tire rotation, cabin filter, wipers and more, or your own), what it cost, and when each is due again.
- "Due again" sets your **service reminders** for you, and work that matches a reminder resets it.
- **Service history** (More, Maintenance) lists every visit with what was done, the cost and the odometer, and you can log one by hand. It downloads as a spreadsheet from Reports, Insights adds service to the year's running cost, and Wrapped gets a **Car care** slide.
- A new optional alert, **Service visit**, asks on your phone. The wait before asking can be changed in Settings, or turned off.

### 🎃 Seasonal themes
- CarTrak dresses up for the holidays: Halloween in October, Thanksgiving, Christmas (with twinkle lights), New Year's confetti, snow in winter, Valentine's, St. Patrick's, Easter and spring, the Fourth of July (fireworks), summer and autumn. Each changes the accent color and the battery ring, puts an ornament by the name, and plays a short effect on Home once a session, bigger on the holiday itself. Wrapped and its infographic take on the season's colors.
- Themes follow the dates around each holiday. Choose **Automatic**, a favorite, or **Off** in More, Display and data. The effect never plays with "reduce motion" or Data saver, and text keeps its contrast in light and dark.

### ✨ What's new, after every update
- The first time you open the app after an update, a sheet shows what changed in every version since you last opened it. **Got it** closes it until the next update.

### 🧭 Your own bottom bar and More
- **Customize** (in the More sheet) chooses the pages in the bottom bar and the tiles at the top of More, in any order. Turn on **Arrange by how often I use them** to keep both in order of the pages you open most. Saved to your account, so it follows you; **Reset** goes back to the start.
- The More tiles are now Insights, Reports, Day view, Places, Plan a trip and Settings. Things moved to where they belong: **charging prices** on the Charging page, **trip rules** on the Trips page, **alert settings** behind the bell (⚙ at the top of Notifications), and **people, cars and backups** under Settings and the Admin part of More.

### ✨ Wrapped: how you compare
- A new slide compares your driving with the average American driver (about 13,500 miles a year, from the US Department of Transportation): "1.8× the miles of the average American driver". CarTrak has no other drivers' data, so it compares with that published average rather than claiming a ranking.

### 💚 Charity
- A new **Charity** trip category, starting at the US charity mileage rate of 14 cents a mile, and a **Charity trip** mode that labels every trip of a stretch of days as Charity. Reports and the mileage log can include it.

### 🔐 Sign-ins and devices
- Admins see **everyone** at a glance: each account's signed-in devices, when it was last used, its last sign-in and failed attempts. Every device shows when it was last active.

### 🐞 Fixes
- **A day in the life** showed 100% parked for a year (or any stretch that began before CarTrak did). It now counts from the first day with data, and draws a short history clearly.
- The **cheapest kWh** record now only counts paid public charging, not home or free chargers.

## 0.48.0

### 🔔 Notifications that stay saved
- Choosing alerts sometimes did nothing: after **Refresh the app** (or reinstalling), the phone lost its notification registration, and the Notifications page then quietly ignored every tick. Refresh now keeps the registration, the page registers the phone again by itself when it finds this, and it says so when a choice cannot be saved.

### ⏻ Remote starts in the feed
- Remote starts now have rows in the **Day view** and **History**, like charges: when the car ran, for how long, the battery before and after, the energy it took and the temperature outside. The day's summary mentions them, History can show only remote starts, and Insights (Battery) adds them up.
- Remote starts from before this update are found in the stored readings, gently, in the background.

### ▶️ Trip replay
- Open a trip and press **Replay this trip**: a dot drives the route on the map while the clock, speed, battery and distance follow it. Drag the slider to jump, and play at 10×, 30×, 60× or 120×. A phone GPS track gives the smoothest replay.

### 💡 Tips from your charging
- The Charging page can show a few tips from your last two months, only when the numbers clearly back them up: charging to 100% for short daily driving, what the night rate would have saved, a fast charger that is slow for you, arriving at chargers very low, and what public charging costs on top of home.

### 🧭 Skip a suggestion
- Swipe a **Suggested setup** item to the left, or tap ×, to skip it. It leaves the Home card and shows as skipped under More, Setup checklist, with **Undo**.

### ✨ Wrapped
- The infographic lays itself out so nothing overlaps the footer, frames where most of your driving is, and draws a few trips brighter. Comparisons read properly ("2 drives across the USA, coast to coast").

## 0.47.0

### ✨ Insights
A new **Insights** section (More, Insights, and on the Reports page) makes your history fun to look at.
- **Wrapped**: your month or year as full-screen slides you tap through: the distance compared with something everyone knows, your car's **personality** (The Night Charger, The Road Tripper…), energy, cost and savings, where every kWh went, towns visited, highlights, achievements and a typical day.
- **Infographic**: Wrapped ends with one tall picture of the month or year that you can **share or save**.
- **Everywhere you've been**: every route you drove glowing on a dark map, brightest where you drive most.
- **Achievements**: 19 badges with progress rings, from a 1,000-mile month to the 150 kW club, Sub-zero and Explorer.
- **Where every kWh went** and **A day in the life**: an energy flow from the charger to city and highway driving, climate and losses, and a 24-hour clock of when the car drives and charges.
- Detailed charts in four tabs: **Charging** (energy by month, price per kWh, where it came from, when you plug in, fastest charges), **Driving** (efficiency against temperature with your own trend line, a calendar of miles, efficiency by speed, trip lengths, when you drive), **Records** (a recap for any month and your personal bests) and **Battery** (daily level, time at each level, capacity, 12 V).

### 🔔 Two new optional alerts
- **Achievement earned** when you earn a badge, and **Your month, wrapped** on the first of the month. Both also appear in the Inbox; for the first week of a month Home shows "Your September is ready".

### 🐞 Fixes
- The Low battery alert's description still said it was off until a level was chosen; it now names the Battery reserve.
- Line charts leave room for their title above the top label.

## 0.46.0

### ✨ A fresher, more consistent look
- **Charging sessions look like trips**: a battery picture filled from where the charge started to where it ended, what it added and how long, where, the battery bar, peak and average power, and the cost with where it came from (**receipt**, **place price**, **home price**, **public price**). Editing where it was, the charger's kWh and what you paid sits behind an **Edit** button.
- One color language everywhere: green is energy going in (charging), amber is energy going out (driving), gray is parked. Rows on the Day view and History now have the same colored edge as trip and charging cards.
- On a phone, trip cards give the town names the full width, with the label and share button underneath.
- The links at the top of Trips and Charging are a single row of small buttons instead of a stack.
- Fix: Home said **Charging** whenever the car was not driving. It now says Parked unless the car is really charging.

### 🧭 Suggested setup
- New people get a **Suggested setup** card on Home: install the app, take the tour, turn on alerts, and for admins name Home, set prices, compare with a gas car, price your usual chargers, phone tracking, a service reminder and a backup. Nothing is required. **Hide** takes the card away for you; the whole list stays under **More, Setup checklist**.

### 🛣️ Roads and distances
- Trips whose odometer reading was short now get their road drawn: the road is judged against the straight line from start to end, not only the odometer. Trips given up on before are tried once more.
- The Trips list finds a few missing roads in the background, and a share link finds the roads of its trips before it is made. A dashed thumbnail means the line only joins the few places the car reported.
- When a trip's distance is **drastically** short, its page says so near the top and offers the road through the recorded positions or the drive straight from start to end, or you can type the real distance. Such trips also appear under **Needs attention** on Home. Small differences are left alone, and nothing changes until you choose.

### 🧹 Clearer settings
- **One battery reserve per car** (More, Charging settings, 20% unless chosen). Home, Live, Plan ahead, the road-trip planner and the Low battery alert all use it; the separate per-phone road-trip reserve is gone.
- **City names for trips and reports** says what it really does: towns on Trips, the Day view, History and the mileage log, looked up a few at a time and remembered.
- Rarely changed service addresses fold into an **Advanced** section in Settings.
- **More** is tidier: Help has Status, Setup checklist, Help and tour, Data check and About.
- **Fix a day** is out of sight until it is needed: Home offers it once after an update that finds trips or charging differently, and Data check links to it.

### 🐞 Better debug reports
- Each charge says where its cost came from and the price per kWh.
- A **Time away** section lists modes and how each day's base was settled (chosen, or found from the nights), without positions.
- Fix a day runs are logged with what was replaced and what was left alone.
- The report adds the app version this phone loaded and its last few app errors.

## 0.45.0

### 🛣️ More for road-trip days
- The summary says how much of the day on the road went on charging ("On the road for 14 h 41 min, 3 h 33 min of it charging (24%)") and the longest stretch between charges.
- Open a public charge that is priced from your general price and you can say **what a kWh costs here**. It is saved as a place with its own price, so every charge there, past and future, is worked out with it.
- The car finishing its charge at home before you leave is no longer counted as one of the day's charges.
- **Stopped … in all** only appears when you stayed at least five minutes longer than the car charged.

## 0.44.0

### 🛣️ Road-trip days read better
- Each row of **Day view** and **History** shows the battery at its start and end (**98 → 51%**) instead of only the change: on a road trip, what you arrive with is what matters.
- Charges show **what a kWh cost**, marked **(receipt)** or **(estimate)**, so you can see which ones still need a receipt.
- A charge says how long the **whole stop** was when you stayed longer than it charged.
- When several drives come between two charges, a **Leg** line sums them up: how far, how long, and what you arrived with.
- The car finishing its charge at home just before you leave no longer has a row of its own: the first drive says **Left home at 98%**.
- Fixes: a home charge no longer reads **Home · Home**, a tiny top-up says **Top-up** instead of **about 0 kW**, slow charging shows a decimal (7.4 kW), and long durations no longer get cut off.

## 0.43.0

### 🏨 Trips away have a base
- When the car spends the nights away from home during a mode (a vacation or a work trip), CarTrak finds your **base** from where it sleeps. Time parked there is not counted as a stop, the same as home, and drives read **Base → Glen Arbor** or **Glen Arbor → Base**.
- The Day view adds **the trip so far**: how far you have driven since it began and what the driving cost per mile.
- In **Modes** you can choose one of your places as the base instead of the one CarTrak finds.
- While you are away, **Plan ahead** forecasts from your driving on the trip, up to its last day, instead of your weekdays at home.

### 💲 A fair cost per mile for a day
- The Day view used to divide the day's charging bill by the day's miles, so a charge that filled the battery for several days made one day look expensive. It now works it out from the energy the day's driving used and what a kWh has cost you (on a trip, what it cost on the trip).

## 0.42.2

### 🎨 What the colors mean
- The Trips list has a **What the colors mean** fold: the green bar is battery left after the drive and amber the part it used, a green badge is more efficient than your average and red less, the dots mark where a drive started (green) and ended (red), and the stripe on the left is the category color.
- The trip page now shows the same battery bar, with what its colors mean.
- A trip that started and ended in the same town reads **Around Traverse City** instead of just the town.

### 🪲 A clearer Debug report
- The locks-and-doors section is a few plain lines per change (when the car reported, the locks, which doors were open, hood and alarm) instead of every raw field, so the report is much shorter. The raw fields are still in the full list.
- **Alerts sent** now includes what each alert said, not only its title.

## 0.42.1

### 🔕 No false alarm when the car is just switched on
- The car reports its 12 V load as **accessory mode** when it is switched on without driving (sitting in it, or right after unplugging). CarTrak treated that as a problem and sent **The car reports a problem**. It is now counted as normal, in the alert and on the Health page; a real load problem is still reported.

## 0.42.0

### 📍 The day says where things happened
- Each row of **Day view** (and the **History** feed) now says where it was: a drive reads **Traverse City → Glen Arbor**, a stop or a charge shows its place, using a place you named or else the city. A drive that began and ended in the same town reads **Around** it.
- Drives also show their **average speed** and **efficiency**, and charges their **average power** ("about 9 kW on average"). A charge reads **Public · Escanaba Super Charger** instead of just Public.

### 🧹 A tidy-up after a review of the whole app
- **What's new** shows **bold** text as bold instead of with asterisks around it.
- Checkbox rows (Notifications, Reports) are taller, so they are easier to tap on a phone.
- With no signal, the **Status** page now opens from what the device saved too (a pattern meant for other pages was catching it).
- The Trips list and the savings figures load faster: place names are read once per list instead of once per trip, and charging sessions are worked through once.

### 🔒 Safer sign-in
- Signing in with a name that does not exist now takes as long as a wrong password, so the answer does not reveal which names exist.
- The sign-in lockout forgets addresses that have gone quiet, so a flood of wrong share links cannot make it grow without end.
- First-time setup can only create one administrator, even if two people try at the same moment.

## 0.41.0

### 💰 Cost per mile, savings and CO2
- **Home → This month** and the **Charging** page now show what charging cost **per mile (or km)**, for the month and the year so far. Each trip page shows an estimated energy cost at your recent average price.
- In **Charging settings → Compared with a gas car**, give the gas price and your gas car's economy (US gallons and mpg, or liters and L per 100 km) to see **what you saved** against it and the **CO2 avoided**. Nothing is compared until you set both.

### 🐢 Slow fast charge is fairer
- A charge all the way to 100% tapers off near full on every charger, so its average is always lower. The **Slow fast charge** alert now only judges fast charges that stopped by about 85%, and only against your earlier charges that did the same.

### 🛠️ Fix a day leaves what did not change alone
- A rebuild now compares each trip and charging session with what it would record and only replaces the ones that come out different. The ones that are already right keep their identity, public links, notes and labels, and the preview says how many were left alone. If nothing differs, it says so and there is nothing to confirm.

### 🗓️ Plan ahead
- Once there are two weeks of driving, Home shows the **next seven days**: the distance you usually drive on each weekday, the energy it takes, and when the battery would fall below your low-battery level without charging ("plug in by Sunday").

### 🔌 Charges that start between two readings
- A parked car is only asked every ten minutes, so a charge that began just after a reading was first seen part-way through, and its session started too late: too high a battery level, too little energy, too short. CarTrak now moves the start back to the last reading before it when the battery had clearly risen, and estimates the start time from the charge power. Use **Fix a day** to correct older sessions.
- **kWh from the charger** now fills itself in when a session ends, from the car's own count of what the charger delivered, when that count fits what went into the battery. A figure you typed is never replaced.

### 🧑‍🏫 A guided tour
- The **?** at the top now starts a two-minute guided tour. Each step opens a page and lights up the part it explains (the Trips tab, a trip card, the More button, the search and inbox buttons), with a sheet showing the step, Back, Next and Skip. Administrators also get steps for Reports, Notifications and Cars. It is also under **More, How to use CarTrak**. The Help page (quick settings and a checklist) is still there, under **More, Help and quick settings**, and has a button to start the tour.

### 🔋 A steadier battery capacity chart
- The car's own capacity estimate moves a few percent with the battery temperature and the charge level. The chart on Health now shows the middle reading of each day instead of the average, so a few odd readings no longer pull the line around.

### 🧭 Trip places no longer cut off
- On the Trips list, the start and end now sit on two lines that wrap, so a long township name is never cut off with "…". A trip that ended where it began shows the place once.

## 0.39.1

### 🕒 Old door reports are labeled as old
- A parked car stops reporting, so a door shown as open can be the last thing the car said before it went to sleep. The **Doors** rows on the Body tab and the **Doors and locks** line now say when the car last reported them.
- The **Left open** alert says "The car last reported this at 9:29 am, so it may have been closed since", and it no longer repeats for the very same old report (for example after every trip).

## 0.39.0

### 📴 Works with no signal
- Every read-only screen (Trips, Day, Charging, Statistics, the mileage log, Health and the rest) now remembers its last answer on the device, so with no signal it opens with a small **"Offline. Showing what this device saved"** bar instead of an error. Home already did this.
- Screens that show a "last 7 / 30 days" window find their saved answer again even though the exact time has moved on.
- Map pictures you have looked at are kept on the device (up to 500), so a map you have seen before still draws offline.
- Changes (editing a trip, settings) say "You are offline" rather than failing silently. Nothing is queued.
- Screens with secrets or people (sharing, sign-in log, Ford link, backups, reports, the Debug report) are never kept. Everything saved on the device is wiped when you sign out.

## 0.38.3

### 🚪 Honest door names
- Ford reports the two front doors as a single "front" entry, labeled with the driver's side, so a passenger door that was open showed up as "Driver front". CarTrak now calls it **Front door (driver or passenger side)** on the Body tab, the Doors summary and the Left open alert, unless Ford ever reports the passenger door on its own. The rear doors are named as before.

## 0.38.2

### 📍 Where each trip began and ended
- Each trip in the Trips list now shows **where it started and where it stopped**: the place you named if one applies (Home, Work…), otherwise the city. Cities are looked up in the background a few at a time, so a long history fills in over the next few visits to the list; nothing waits for them.

## 0.38.1

### 🪲 Pick any car for the Debug report
- The Debug report now has a list of the cars you can open, by name, so you can make the report for any one of them (an administrator of the whole installation sees every car, marked "not yours" when it belongs to someone else), as well as the car on screen or every car together. The report itself still names no car and no person.

## 0.38.0

### 🛑 Stops are stops on the road
- **A charge at home is no longer counted as a stop.** The Day view's Stops tile and the numbered stops only count charging away from home, and the written summary says how many of the charges were at home ("7 times (6 fast and 1 slow; one at home)").

### ✨ A livelier Trips list
- Each drive is now a card with a **little picture of its route** (green where it began, red where it ended), a big distance, the time and place, a **battery bar** showing the level it ended at and the stretch it used, and an **efficiency badge** that turns green or red when the drive was clearly better or worse than the rest of the list. The category sits on the right with a round **Share** button, and a stripe down the left edge takes the category's color. The Day view's tiles have icons too.

## 0.37.1

### 🔧 Fixes
- **No more stray "null".** The share box showed the word "null" under the link once the link was made, and any other page that left a spot empty could do the same. One fix covers every page.
- **Plan a trip** is just "Plan a trip" now, without "70 vs 75" (the two speeds are still picked on the planner itself).

### 📏 Fix a drive's distance
- The car's odometer sometimes updates late, so a drive can be recorded much shorter than it was, and the road estimate then refuses it ("the road found was 11.2 mi but the odometer says 5.6 mi"). Each such trip in that notice now has a **Use 11.2 mi** button, and every trip has a **Does the distance look wrong?** card: check the road distance and use it, type the real distance, or go back to the odometer's. The road is drawn on the trip, the trip is marked as corrected, and reports and the mileage log use the corrected figure. Fix a day keeps a correction when the drive stays the same drive.

### 🪲 A fuller Debug report, for one car or every car
- The report is for the car on screen. Administrators of the whole installation can now choose **Every car** and get one report with each car under its own heading (by number, never by name).
- It now also lists where this car sits among the others, how many categories there are and today's mode, how CarTrak knows where home is, places with their own charging price, trips with a corrected distance, trips whose road could not be estimated, public links in use and custom reports. Still no positions, names, keys or VIN.

### 📈 Share the efficiency details
- A share link can now include **how it went**: top and average speed, the temperature, how much more or less energy the trip used than usual, and the reasons (cold weather, highway speed, stop and go…). It is a tick box in the share box, on by default, and works for a trip, a journey or a day (a single trip also gets the temperature and reasons). It never shows the trip's category or where it went beyond what you chose.

### 🔗 Shorter share links
- New share links are 12 letters and digits (for example `/s/k4m9q2x7t3ha`) instead of 64, and the share box has a **Share…** button beside Copy on phones. Links made before keep working.

## 0.37.0

### 📊 Reports and insights
- **More, Reports.** One page for everything that adds your driving and charging up: Year in review, Statistics, History, Time, Efficiency insights, Compare two trips, Day view, the road-trip planner, Charging cost and the Mileage log, each with a line of what it shows (and the year's distance, this month's charging). The spreadsheets (trips, charging, everything) are at the bottom. A **Reports** button is also on the Trips page.
- **Did you know (Home).** One thing worth knowing about how you drive, taken from your own history and changing each day: what cold weather costs you, your best month, your most efficient trip, or what driving fast used over the last 60 days. Tap it for all the efficiency insights.

### ⚡ Charging settings, and a price for each place
- **Charging, Charging settings.** The price per kWh at home and away, the currency, the cheaper night rate and its hours, the charger's efficiency, the plug-in reminder, the low-battery level and the monthly budget now have a page of their own under Charging (and in More), instead of being mixed in with the technical settings. The owner of each car sets them for that car. They no longer appear in the general settings list.
- **Where home is.** CarTrak now treats a place you named **Home** (More, Places) as home, so charging there is priced at the home rate and counts as home in the statistics. Before, it only knew home once you marked a session as Home by hand. The page says whether CarTrak knows where home is, and how.
- **A price for any place.** Under each place in More, Places, set what charging costs there (0 if it is free), or tick that it counts as home. A cabin, a relative's house or a workplace charger can have its own rate; every other place keeps the usual home or public price. A session you set to Home or Away by hand still uses that choice.

### 🧭 A More sheet
- The **More** tab now slides a sheet up from the bottom: tiles for Reports, Cars, Charging prices, Places, Trip rules, Notifications, Settings, People and Backup, then How to use CarTrak, Everything else in More, Report a problem or idea, and Sign out. The full More page is still one row down.

### 📑 Downloadable reports
- **More, Reports now has a report you can download.** The **standard report** is a mileage log like the ones other apps export: one row per trip with its value, a summary above it and a Totals By Label sheet, as an Excel file (or a CSV). Untick the columns you do not want.
- **FedEx report.** The mileage log in the layout it has always had (origin city, the cities parked in, description, miles, and the total), as Excel or CSV, from the same Reports page.
- **From and To are just the city**, and CarTrak names the cities first so the file has them.
- **Custom reports.** Save a layout with its own columns, headings and fixed columns (such as a company name), then run it whenever it is needed.
- Extra columns to add: **energy used (kWh)**, **efficiency** in the unit you prefer (mi/kWh, kWh per 100 km, Wh per mile, or distance per percent), **energy cost**, **cost per mile**, **MPGe** and **average speed**.
- Every time you run one, choose **all categories, some, or one**, and the **days** (first and last day, or calendar week numbers such as 26-28).
- See [docs/reports.md](docs/reports.md).

### 🏷️ Trip categories and modes
- **Medical, and categories of your own.** A trip can now be Personal, Work, **Medical**, or any category you add (More, Trip rules, Trip categories): a name, a color and an optional mileage rate for each. Trip rules, the trip filter, the review list, the year in review and the Mileage log all use them. Removing one of your own makes its trips Personal again.
- **Modes.** Vacation mode is now one of several: pick the days and a mode (Vacation, **Work trip**, **Medical**, or one of your categories) and every trip in them gets that label. If two modes cover the same day, the one you added last wins. Trips you set by hand keep what you chose.

### 🔗 Share a trip from more places
- **Share a whole day.** The Day view has a **Share this day** button: one public link for all of the day's trips, with the same choices as a trip (route trimmed or whole, or numbers only, cities, times, expiry). It never shows the trips' categories.
- A **Share** button on every trip in the Trips list, and **Share this trip** in a drive opened in the Day view (admins). Both open the trip with its share box ready. The Home Last trip card already had it.

## 0.36.0

### 🔔 A bell, a search and a help button in the header
- **Notifications (the bell).** Every alert CarTrak raises, for every car you can open, is kept in an inbox, newest first, grouped by day, with the car's name when you have more than one. Tap an alert to open the trip or page it is about. A red number on the bell counts what is new since you last looked, on this device. It works even when push notifications are off. If something needs a look (Status is not OK), the inbox says so at the top.
- **Search.** The magnifier opens a search box over any page, so a setting or a page is a word away instead of a trip through More.
- **Help (the ?).** One place for everyone: quick settings for units, theme and data use; a short getting-started checklist (connect the car, put CarTrak on the home screen, turn on notifications); what each part of the app is for; and links to the guide and Status. Admins also get a link to the full checklist.

### 🏠 More on Home
- **Today**: trips and distance so far today, what you charged and what it cost, with a way into the day's map and yesterday.
- **This month**: charging energy and cost so far this month, and the next service coming up. Tap it for statistics.
- **Share this trip** on the Last trip card opens the trip with its share box ready (admins).

### 🧰 Under the hood
- Fourteen source files that had grown past 200 lines were split by responsibility, and the list of pages moved out of the app's start-up file. Nothing you see changes.
- Alerts now remember the page they open, so the inbox can link to it (alerts from before the update are listed but do not open anything).

## 0.35.2

### 🧰 Under the hood
- Fourteen source files that had grown past 200 lines were split by responsibility, and the list of pages moved out of the app's start-up file. Nothing you see changes; it makes the code easier to read and to change safely.

## 0.35.1

### 🔧 Fixes
- **Fix a day now merges a paused charge.** A rebuild left the cable out of the readings it replayed, so it never joined a charge that paused and picked up again, and never joined a quick replug. Run **Fix a day** again over the days with split sessions.
- **Road estimates name the trips they could not do.** The message now lists each trip (tap it to open it) and why, for example that the road found was 11.2 mi but the odometer says 5.6 mi. A trip whose road came out a very different length is remembered and not asked for again, so the notice does not keep coming back.

### 📏 Miles and kilometers
- A hunt through every screen with the app set to miles found distances in kilometers in a few places. The **Data check** findings, the road-estimate message, the service reminder checks, the **Debug report** and the wind lookup help now follow the units set on the device. The Debug report names its units in its column headings.
- The **trips spreadsheet (CSV)** has two more columns at the end, distance in miles and top speed in mph.

## 0.35.0

### 🚗 Several cars and several people
- **More, Cars.** CarTrak can follow more than one car (up to 6). Each car has its own database, settings, alerts and Ford link, so one car's history can never show up in another's. A selector at the top switches cars; each browser tab remembers its own.
- **Each person sees only their cars.** Under **More, People** an administrator chooses, for each person, whether they see the first car and whether they may add a car of their own. A person who may add a car gives it a name and links it to their own Ford account. The owner of a car manages it (settings, Fix a day, backups, who can look) but not the people or remote access.
- **Administrators can open any car** to help, and the owner of that car is told that an administrator looked, and when.
- **Alerts go only to people who can see the car**, and name the car when you follow more than one.
- **Backups.** A car's backup holds that car and no accounts, and a restore replaces only that car. Administrators can download a zip of everything. Applying a restore stops CarTrak for everyone, so only an administrator of the installation can confirm it.
- **First start after updating.** CarTrak makes a shared database for the people, copies them in, and keeps a copy of your database from before, in the data folder under `backups`. Nothing else changes: you stay signed in and see the car you always did.
- Details in [docs/several-cars.md](docs/several-cars.md).

### 🔌 Charging
- **A quick unplug and replug is one stop.** A fast charger that stops and is plugged back in within 5 minutes, at the same place with the battery no lower, now continues the same session instead of starting a second one.
- **Slow-charging energy now counts what the car drew beyond the battery.** Overnight at home the car keeps drawing power at 100% (a top-up, warming the battery or cabin before leaving) that never shows as battery gain, so the energy and cost came out about a tenth low. CarTrak now adds up the power the car reports while plugged in and uses it when it is larger. Run **Fix a day** over older charging to apply it to past sessions. The Debug report lists it as "power kWh".

### 🛣️ Road estimates
- When some trips could not be estimated, the message now says why (the routing service answered an error, no route, or the road found was a different length from the odometer).

## 0.34.0

### 🗺️ Day view opens details in place
- Tap a drive or a charging session in the Day view and its details open right under it, so you stay on the day. A drive shows distance, average and top speed, energy, efficiency, battery and why the efficiency was what it was; a charge shows energy, cost, where, and its power curve. A link to the full trip or to Charging is still there when you want to edit.

### 🪲 Debug report
- The window now goes back **7 or 14 days** (it was 3), and lists more reading lines the longer the window.
- The per-session power line only lists battery levels the session actually climbed through; a session that began at 48% no longer shows "20%" and "40%".

## 0.33.0

### 🪲 A fuller Debug report
- **Power and energy on every reading.** Fast chargers report no power at all, so the report showed 0.0 kW for a whole fast charge. It now shows the power worked out from the battery's current and voltage (marked with a ~), and the energy left in kWh, so a charge can be checked against a receipt line by line.
- **Decisions that leave no trace.** A trip thrown away as a blip, a charging session picked up again after a pause, a charging stop and why it was called complete or interrupted, and a bump alert held back (and why) are now listed with their times. They used to vanish.
- **Polling.** How often CarTrak really asked Ford since it last started: the pace it planned, the actual time between answers, and each rate limit or error.
- **Settings snapshot.** The settings that shape behavior (poll intervals, prices, alert levels, whether the wind and road lookups are on, the time zone, the battery size and the named places) with no secrets and no positions.
- **Per-session detail.** Each charging session shows its charger type, what you paid if you entered it, and the power as the battery passed 20, 40, 60 and 80 percent.
- It lists up to 1100 reading lines instead of 700, so more of a three-day window fits.

### 🗺️ Day view
- **Trips → Day view** (and **Map of this day** on each day of the History feed, and **See the whole day** on a trip) shows a whole day on one map: every trip in its own color, the stops between them as numbered markers that match the numbers in the list underneath, and the day's totals.
- A written **Summary** says it in plain words: how long you drove over how many trips and how far, at what average and top speed; how many times you stopped and for how long in all (and the longest stop); how many times you charged (fast or slow), for how long, how many kWh were added, at what average and peak power, and what it cost. The cost says how much comes from receipts you entered and how much is worked out from a price, and sessions that could not be priced are called out. Below it: when you were out from and to, the battery at the start and the end, charging cost per mile, and the energy used driving.
- Routes use phone GPS first, then a kept estimated road, then the car's own positions.

### ✨ Estimated routes look better
- The estimated road is now drawn as a fine dotted line with a small **Estimated route** tag on the map, instead of long dashes, with a shorter note under it.

### 🔌 A paused charge is one session
- When a charger pauses and starts again (a fast charger handing over, or a hiccup), CarTrak used to close the session at the pause and start a new one, missing the energy that went in while it was between the two. A session that resumes within 20 minutes, at the same place, with the cable still plugged in, is now **one session**. Unplugging and plugging back in is still a new one. **Fix a day** applies this to older days.
- **Fast chargers are read at the charging pace.** Ford reports no power at a DC fast charger, only the charge state, and CarTrak was only speeding up when it saw power. With the ignition off it read the car every ten minutes for a whole fast charge, which made charging curves coarse. It now speeds up on the charge state too, and keeps the faster pace for 20 minutes after a pause while the car is plugged in, so a restart is seen when it happens.

### 🧾 What you really paid
- Each finished session on the Charging page can take the **kWh the charger reported** and **what you paid**, typed from the receipt. A session with a receipt cost shows "cost from your receipt" and its real cost goes into the monthly totals, the year in review and the CSV, instead of the flat price. The CSV has two new columns, whether the cost is from a receipt and the charger type.
- A rebuild (Fix a day) keeps the cost and kWh you entered.

### 🧭 Debug report: real route points
- The per-trip line now counts **different car positions**, not readings. The car repeats its last known position in every reading, so a trip with 283 readings can have only two distinct positions, which is why a car-only trip is a straight line.
## 0.32.0

### 🗺️ Estimated roads for trips without phone GPS (optional, off by default)
- A trip the car recorded with only its start and end is drawn as a straight line across the lake. Switch on **Road estimate for trips without phone GPS** in More, Settings and CarTrak asks a routing service for the likely road between the positions the car recorded, and draws it **dashed** with a note that it is an estimate. A phone GPS track always wins over it.
- **Older trips:** on the Trips page, **Estimate their roads** works through the older straight-line trips a few at a time. The answers are kept, so the routing service is asked once per trip.
- **Privacy:** it sends the start, the end and a few positions along the way to the routing service, which is why it is off by default. It uses the public OSRM demo server unless you give your own address (More, Settings, Routing service address), and a length check against the odometer drops any estimate that looks wrong.

### 🔒 Security
- **Go 1.27.2.** A scan of the project found 10 known vulnerabilities in the Go standard library (the web server and TLS code CarTrak is built on), all fixed in Go 1.27.2. CarTrak now builds with that version.
- **Automatic security scans.** A new weekly (and on every push to the main branch) check runs Go's official vulnerability scanner and GitHub's CodeQL code scanner for Go and JavaScript. Both are free for public projects; results appear in the repository's Security tab.

## 0.31.0

### 🔎 Field watcher
- **Car data → Field watcher** lists every kind of reading the car sends and what each one has been doing: its latest value, how many times it changed in the last 24 hours, 2 days or 7 days, and when it last changed. Each is marked with what CarTrak uses it for, **Car data only**, or **Not used**, so a field nobody is looking at stands out. Filter by Changing, Never changed or Not used, search by name, and tap one for its range and the car's own time for it. A new field Ford starts sending shows up here on its own.

### 🔋 Range at 100%
- Home shows the range the car expects on a full charge next to the energy left and the battery temperature.

### 🧭 Debug report: where a trip's route comes from
- Each trip in the **Debug report** now has a line saying how many readings the car sent during it, how many of them carried a position, and how many phone GPS points there were. It shows why one trip's route is a dense line (phone GPS) and another is a single straight line (the car sent only its start and end).

### 🔥 Remote start, on screen
- When the car is running from a **remote start**, **Home** and **Live** show a **Remote start · 12:30 left** chip that counts down each second, using the car's own countdown until it shuts itself off. It disappears when the countdown ends.
- Press the chip's area (hover or long-press) to see how many remote starts CarTrak has seen since the car was last driven. The car only allows a couple in a row, so this is a quick way to know where you stand. It is CarTrak's own count from the readings it saw, so a start that began and ended between two readings is missed.
- A remote start is still never recorded as a trip.
## 0.30.2

### 🐛 Fixes
- **Charging was cut off at the newest 100 sessions** in three places: the charging file in **Export everything**, the **History** feed for older months, and **Statistics** for a year. All three now include every session.
- **No more "Trip started" for things that are not trips.** A trip is announced only once it has gone somewhere (about 300 m or 8 km/h), so sitting in the car with the ignition on, or a blip that is thrown away, sends nothing, and neither does the "Trip ended" that used to follow. The first reading after the car wakes often has no speed; a trip now waits for the odometer to move before it starts.
- **No "Bump while parked" right after you park.** For five minutes after the car was last in use, jumps in the motion sensors are logged on the Parked watch page but do not send an alert: the door, the seat and the car settling look like a bump, and so does the sensor's first fresh reading after a drive.
- **Removed the car's regen counter.** It looked like energy in 0.30.0, but it barely moves between trips (99.5, then 98.2) while the range braking gave back falls by 40%, so it is not an energy count. The regen tiles go back to the range the car reports and the energy worked out from it, marked as an estimate.
- **Statistics and History pick the right month** whatever time zone the phone is in (the month and year are now named by the server, not worked out from the phone's clock).
## 0.30.1

### 🚗 Remote start is never a trip
- A trip only starts when the car is actually moving, so a remote start (ignition on, speed zero) was already ignored. One gap is now closed: if the car sends no speed at all during a remote start, the countdown until it shuts off tells CarTrak nobody is driving, so it still does not start a trip.

## 0.30.0

### 🔌 Charging alerts that tell the truth
- A charge that **stopped because of a fault**, or because the car **lost contact with the charger**, now sends **Charging interrupted** with the reason, instead of "Charging complete". The car's own charge state (FAULT, STATION_NOT_DETECTED) is used.
- A pause at a low level (46%, say) is no longer called complete. The session that has just ended was being counted as "where your sessions usually stop", which hid exactly this case.
- New alert **Slow fast charge**: when a fast charge that started below 60% averaged under 70% of your usual fast charges (needs three earlier ones), you hear it, so you can try a different charger next time. Off by default like the other alerts you pick.
- New alert **Car-reported problem**: the car does its own checks on the main battery, the tire pressure system and the 12 V load. Anything other than normal sends one alert a day for that check.

### ⚡ Level 2 or fast, from the car
- Each charging session now keeps what the car says the charger is (AC or DC). **Fastest and average for Level 2 and Level 3** use it, and fall back to the old peak-power guess for sessions recorded before this version.

### ♻️ The car's own regen energy
- The car keeps a count of the energy braking gave back. CarTrak now checks it against its own estimate on your recent trips, and when they agree on most trips it shows the car's number (marked "car") instead of the estimate (marked "estimate").

### 🔋 Range on a full charge
- **True battery level** now shows the range the car expects at 100%, with how it has moved over the history CarTrak has. It moves with the weather and how you drive, so it is a slow trend, a second battery-health signal next to the usable size.

### 🎛️ How firmly you drove
- **Why this efficiency?** says whether a trip was driven firmer or smoother than your usual, from the car's torque readings, with a short chip ("Firm driving", "Smooth driving"). There is no absolute scale in the readings, so firm means above what your own recent trips reached nine times out of ten.

### ⚖️ Speed comparison
- The two speed boxes and the trip length update as you type, and the comparison ends with a plain recommendation ("Driving faster saves time: 75 mph gets you there 20 min sooner…").
## 0.29.0

### 📜 History and statistics
- **History** (Trips, History feed): a month of driving, charging and parked time as one feed, newest first. Tabs for Charging, Driving, Parked or All; each row shows the time range, how long, the battery change on a thin bar, the place, and the cost or distance. Month totals sit on top and arrows move between months.
- **Statistics** (Charging, Statistics): bars by Day, Month or Year for distance, cost, kWh or driving time, with tap-to-read. Below it, **Where you charge** splits charging into Home, Work and Public as a ring and three tiles. A place named Work or Office (More, Places) counts as work.
- The segmented tabs are now rounded pills.

### ♻️ Regenerative braking on each trip
- **Why this efficiency?** shows **Range from regen** and **Energy from regen** as tiles. The energy is the range the car says braking gave back, valued at what that trip used per mile.

### ⚖️ Compare two speeds on Efficiency insights
- The **By speed** card on Efficiency insights now has **Compare two speeds** (70 and 75 mph to begin with, and you can type any speed). It shows **one hour at each speed**: how far you go, the energy used and how long it takes to fast charge that energy back. Choose a trip length (600 miles to begin with) and it also shows the driving time saved, the charging and stops added, whether the whole trip ends up quicker or slower, and how much more energy it uses, in kWh and as a percent per mile.

### 🧭 Easier to find
- **Plan a trip · 70 vs 75** buttons on the Trips and Charging pages open the planner, and searching for "70 vs 75" finds it.
## 0.28.0

### ⚡ 70 or 75 on a long drive, answered with your own charging
- The **Road-trip planner** now starts its charging speed from how long your own fast-charging sessions took to climb from your **reserve to your charge target** (for example 20% to 80%), instead of a whole-session average that includes the slow top end. The planner says how many sessions that came from, and you can still type your own number.
- **Best speed for this trip** has a **Compare** pair of pickers (70 and 75 mph to begin with, remembered on this device). It says what the faster speed saves in driving, what it adds in charging and stops, how much quicker or slower the whole trip ends up, and how much more energy it uses. On a 600-mile drive that is the answer to "is 75 really worth it?".

## 0.27.0

### ⏱️ The best speed for a trip
- The **Road-trip planner** has a **Best speed for this trip** card. Going faster saves driving time but uses more energy, and energy means charging time, so the fastest speed is not always the quickest trip. For your distance, starting charge, weather and charging speed it works out the total time (driving, charging and a few minutes lost at each stop) at each speed from 55 to 85 mph and names the one that gets you there soonest, for example "Going from 70 to 75 mph saves 6 min of driving but adds 9 min of charging, so it is 3 min slower overall and uses 11 kWh more." The energy at each speed comes from your own drives; speeds you have not driven are marked as estimates.

### 💨 Wind on your trips (optional, off by default)
- Switch on **Wind lookup for trips** in More, Settings and CarTrak asks a free weather service (Open-Meteo, no account) for the wind along each trip. A trip then says whether it had a **headwind**, a **tailwind** or a strong **crosswind** (with gusts), splits wind out of "Everything else" with a rough estimate of what it did to the energy used, and shows chips such as "Headwind +9%" and "Tailwind −7%". Wind also appears in **What costs you the most**.
- Privacy: it is off until you switch it on. It sends only a map cell half a degree wide (about 50 km), never your exact position, plus the date, and it remembers every answer so each place and day is asked once. The motion sensors cannot measure wind (they report the car's own movement in snapshots 30 seconds apart), which is why a weather service is used.

### 🎯 Your most efficient speed
- **Efficiency insights, By speed** now has a chart you can touch and drag along, with finer steps (about 6 mph apart), and says plainly **"Most efficient: 56–62 mph"** with how much more energy cruising faster used. Every speed range is under "Every speed range".

### 🧭 What costs you the most
- A new card adds up, across your last 60 days of trips, how much energy **speed**, **weather** and **everything else** (wind, hills, climate control, braking) cost or saved against your usual driving, in kWh.

### 🎨 Reason chips that tell the truth
- A reason chip is now red only when it really cost that trip energy. A short city trip full of stops and braking can be one of your most efficient (low speeds suit an EV), so there those chips are gray descriptions, not culprits. Speed and weather chips show how much they moved the result ("Highway speed +12%", "Low speeds −18%", "Mild weather −6%"), and a better-than-usual trip with no clear reason says by how much.

## 0.26.0

### 🧼 Old screens can no longer stick
- The app's files are now served from addresses that carry the version (such as /v0.26.0/js/app.js), and the page that points at them is never cached. A phone, a browser or a service such as Cloudflare in front of CarTrak can keep the files as long as it likes: a new version has new addresses, so an old copy can never be served for it. This was why some phones kept showing old screens even after clearing the app's data.
- **More, Display and data, If something looks old** shows the version of the screens on this phone next to the server's version, warns when they differ, and has a **Refresh the app** button that clears the saved app files and loads everything again.
- Nothing needs bumping by hand any more: the cache name follows the version.

### 📏 Feet in imperial
- The size of a place in **Trip rules** and **Places** was always in meters. It now follows your units (feet in imperial).

### 🔄 No more old screens after an update
- The **A new version is ready** bar now also appears when you come back to the app after it has been in the background and the server has been updated, not only while it is open. This is what was leaving a phone on old screens (Celsius in the road-trip planner) next to a new version number.

### 👆 Charts you can actually use on a phone
- Every chart now answers questions. **Touch and drag along a line** (the charging curve and the Health trends) and a readout shows the exact value at that spot, such as "48% · 105 kW" or "Oct 18 · 12.34 V". Labels are bigger, the axes say what they show (battery level, power in kW) and the highest point is marked. **Tap a bar** (Year in review months, battery size by month) and it tells you its value in words, such as "Oct: 700 mi in 40 trips".

### ⚡ Charging speed records
- **Charging** has a **Charging speed** card: your fastest charging power and your average charging speed for **Level 2** (slow, home or public) and **Level 3** (fast chargers, 30 kW or more at the peak), with the date and the named place of the fastest one. Average is the energy added over the time plugged in.

### 🕒 Driving today is more honest
- **Live, Driving today** no longer counts trips that went nowhere (sitting in the car with the ignition on), which made it read far too high.

### 🧹 Clear out empty trips
- **More, Data check** has an **Empty trips** card (admins) when there are finished trips that went nowhere (under 0.2 km, never over 8 km/h, no note, not in a journey), which older versions recorded when you sat in the car with the ignition on. One tap deletes them all, after a confirmation. For a whole day with several problems, **Fix a day** rebuilds it from the readings.

### 🔎 Smarter search on More
- The search box on **More** now suggests **where a thing is**, by name or by what it does. Type **vacation** and it points to Trip rules; **electricity price**, **quiet hours**, **tire**, **backup**, **dark**, **phone** or the name of any setting or alert and it says which page it is on, with a short description and a tap to open it. It knows the real names of your settings and alerts, and the menu below still narrows as you type.

## 0.25.0

### 📈 Charging curves
- Under each charging session, **Charging curve** draws power against battery level, with the peak, the average, how long 10 to 80 percent took, the power at 80 percent and the energy the car measured going in. **Fast chargers compared** on the Charging page lists your fast-charging places by name with their average power, and the road-trip planner now starts its charging power from what your own fast charges averaged. Fast-charger power comes from the battery's own current and voltage.

### 🚗 Speed against consumption
- **Efficiency insights** has a **By speed** card: your miles per kWh at each speed from your own drives, and how much more energy the fastest cruising uses than the band below it.

### 🧐 Why was this trip efficient (or not)?
- A trip now has **Why this efficiency?**, and the best and worst trips on Efficiency insights show short reason chips: **Freezing or cold weather, Hot weather, Highway speed, Stop and go, Idling and climate control, Hard braking, Quick starts**, or **Gentle driving** and **Mild weather** for a good one. It compares the trip with your own driving: how much of the difference the speeds explain, how much the weather explains, and what is left (wind, hills, climate control, braking). Hard braking and quick starts are inferred from big speed changes between readings, so they are a guide, not a measurement. The page also says how much range the car counted braking gave back (its own figure, in kilometers of range), with a **Regenerative braking helped** chip when that is a good share of the trip.

### 🔋 True battery level and tires
- **Health** shows the percentage the car displays next to the battery's own level, the usable battery size worked out from them (energy left divided by the true level), and its trend over the months.
- **Tire pressure and trend** shows each tire against the placard target the car reports and warns when one tire is losing air faster than the others, after allowing for the outside temperature.

### 🚪 Left-open alert and car-measured charging
- A new optional alert, **Left open**, warns once when the parked car has a door, the tailgate, the hood or a window open for ten minutes.
- **Right now** on Charging uses the car's own time to full and shows the energy the car counted going in this session.

### 🔄 A new version is ready
- When a new version is downloaded in the background, a bar says **A new version of CarTrak is ready** with a **Reload** button, so an old screen never sits next to a new server.

### 🏖️ Vacation mode
- **More, Trip rules** has a **Vacation** card. Choose the first and last day (and where, if you like) and every trip on those days counts as **Personal**, over your work rules, so you do not have to change a week of trips by hand. Trips already recorded in the stretch are relabelled at once, trips you set by hand keep your choice, and removing the vacation puts the rules back. Home shows "On vacation until …" while it is on, and the mileage log and exports follow the new labels.

### 🪑 Sitting in the car is not a trip
- Sitting in the car with the ignition or accessories on while it does not move no longer records a trip. A trip now starts when the car moves, and a finished trip with almost no distance and no real speed is dropped.

## 0.24.0

### ⚡ Fast-charger stops are now recognized
- Found in a real Debug report from a road trip: at a DC fast charger Ford reports the charge status **IN_PROGRESS** with 0.0 kW, and CarTrak only looked for the word "CHARGING" or an AC power figure. So fast-charger stops were never seen as charging: no charging sessions were recorded for them and the whole road trip stayed one long trip. CarTrak now treats IN_PROGRESS as charging, and when the onboard charger reports no power it works the power out from the battery's own current and voltage (about 41 kW in the report's example) while a cable is connected. This also fixes the Charging page, Live and Home showing "not charging" at a fast charger, and the Home Assistant feed.
- Run **More, Fix a day** over the road trip again after updating: it now splits the trip at each fast-charger stop and records the charging sessions.

### 🔔 Bump alerts that you can trust
- A bump is now a **change from the reading before it**, not the absolute reading. A car parked on a slope (or with a sensor offset) reads a steady value above the level, which used to send a "Bump while parked" alert every 10 minutes for hours.
- No bump alert while the **ignition is on** (someone is in the car, even while it charges), while the car is **unlocked**, or when a **paired phone** reported a position within 150 m of the car in the last 15 minutes. The motion is still listed on the Parked watch page.
- The **Car left unlocked** alert no longer counts a car you are sitting in with the ignition on while it charges.

### 🔌 Charging shows what is happening now
- **Charging** now starts with a **Right now** card: whether the car is plugged in, whether it is taking charge, the battery level, the power, and the time to 80, 90, 100 percent or your own target while it charges. It refreshes by itself every 30 seconds.

### 🌡️ Temperatures in your units
- The weather groups on **Efficiency insights**, the road-trip planner and the range line on Home were always written in Celsius ("0 to 10 °C") even when the rest of the app showed Fahrenheit. They now follow your units setting ("32 to 50 °F").

### ⚡ Battery and motor on Live
- **Live** has a **Battery and motor** panel with the battery temperature, pack voltage, battery power in or out, the coolant temperature, drive motor voltage and current, and the 12 V battery, updating with each reading. It shows only what the car reports, so a motor temperature appears if your car sends one and is left out if it does not (Ford's list has none for most cars; the Debug report lists every field your car sends).
- Values on Live are never cut short with "…" any more: text shrinks to fit its tile.

## 0.23.0

### 🩹 Fix a day
- **More, Fix a day** (admins) redoes the trips and charging of a stretch of time (Yesterday, Today so far, Last 3 days, or your own dates) from the readings CarTrak stored, using the current rules, which now split a road trip at the fast charger. It shows a **Now** and **After rebuilding** side by side first and changes nothing until you type REBUILD. Notes, categories you chose by hand and charger kWh you typed in are kept; public links to the old trips stop working. It will not touch a trip in progress, a stretch ending in the last 30 minutes, or trips that are part of a journey, and the swap happens in one step or not at all.

### 🧰 Debug report
- **More, Debug report** (admins) makes a plain-text report of the last 6 hours to 3 days: every reading (ignition, speed, plug, charge state, power, battery, odometer, lock), the raw lock and door fields the car sent each time the lock state changed, every field the car reports in its latest reading (so you can see everything the car offers), the trips and charging sessions CarTrak made, and the alerts it sent. Copy it or download it and send it when something looks wrong. It leaves out where the car was, sign-ins, keys and the VIN.

### 🔌 A road trip with the car left on
- A car that is taking charge and not moving now counts as parked even if the ignition was left on. Before, a road trip with the car left on at a fast charger was recorded as one long trip, and because the battery filled up during it the trip's energy use came out near zero (an absurd miles per kWh). It is now one trip before the charge and one after.

## 0.22.0

### 🐞 Fixes
- **More, Security** failed to open for admins ("Cannot access 'shares' before initialization") since 0.19.0. It opens again, with the sign-in log, signed-in devices and your shared links.
- A stray word "null" appeared on the **Backup** page (under Free up space and Restore a backup) and on a trip's **Route** card. Gone.

### 🔔 Two new alerts you can set a number for
- **Low battery**: pick a level under More, Settings (10 to 30 percent) and devices that turned the alert on are told once when the battery falls to it while the car is not charging. It resets once the battery is five points higher or the car charges, and stays quiet right after CarTrak restarts.
- **Charging budget**: enter a monthly amount under More, Settings and devices that turned the alert on are told once a month when that month's charging cost passes it. Needs your price per kWh.

### 🛠️ Fix a bad trip
- On a trip, admins now get **Fix this trip**: **Delete this trip** for a duplicate or junk one (any public link made from it stops working; odometer and battery history are unchanged), or **Close this trip** when it never ended (no readings for 30 minutes). A trip that is still in progress cannot be touched. Data check points you to these trips.

### 🔋 Battery health
- **Health** shows the car's own estimate of its battery size over the months, with a plain verdict (steady, a gentle decline, or worth watching) once there are at least two months of readings. The estimate moves with temperature, so only a lasting drop is flagged.

### 🧾 Mileage deduction estimate
- Set **Mileage deduction per mile** in More, Settings and the Mileage log shows an estimated deduction for the work miles, and Year in review shows it for the year. It is an estimate for planning, not tax advice.

### 🌙 Cheaper night rates
- If your plan has cheaper hours, set **Cheaper home price** and **Cheaper hours** (for example 22:00-06:00) in More, Settings. Home charging is then priced by how much of the session fell in those hours, and **What would a charge cost?** shows what charging in the cheaper hours would save.

### 📦 Export everything
- **More, Backup, Export to spreadsheets** downloads one zip with every trip, every charging session, your named places and your service reminders as CSV files. It is for reading elsewhere, not a backup.

### 🚗 Is this trip good?
- A trip now says how its efficiency compares with your average over the last year ("12% more efficient than your average"), once CarTrak has at least five trips with energy to compare against. Skipped on data saver.

### 💾 A gentle backup reminder
- A new optional alert, **Backup reminder**, tells you at most once a month when you have not downloaded a backup in the last month (and only once CarTrak has been recording for two weeks). It opens More, Backup.

### 👪 A simpler view for family
- Viewer accounts now see five tabs (Home, Trips, Live, Charging, More) and a short More page: the road-trip planner, efficiency insights, Year in review, Time, display settings, notifications and help. Admin tools such as Health, Data, maintenance, rules and connections stay out of the way. Viewers can still open any page by its address, and still cannot change anything.

## 0.21.0

### 🏠 A clearer Home
- Under the battery ring, small chips say at a glance whether the car is **Driving, Charging or Parked**, whether it is **plugged in**, and whether it is **locked**.
- A **Needs attention** card (only when there is something) lists service reminders that are due or coming up and trips waiting for a Work or Personal label, each one tap away.
- **Last 7 days** and **Last trip** (with its named places) now sit right under the battery instead of at the bottom.

### ✅ A fuller Getting started
- **Getting started** now has five optional steps after the basics: name your places, set your electricity price, turn on alerts, add a service reminder and make a backup, each with a button and a tick when done.

### 💵 Charging cost forecast
- **Charging** starts with **What would a charge cost?**: choose a start and end battery level (it begins at where the car is now, to 80 percent, with 80 and 100 buttons) and see the energy and the cost at home and at a fast charger, using your prices. Home includes the heat lost on slow charging.
- The Charging cost card shows **this month so far** and, after the third day, what the month is on pace for.

### ⚖️ Compare two trips
- On **Trips**, tap **Select trips**, tap exactly two and press **Compare 2** to see them side by side: route, distance, time, speeds, battery and energy used and efficiency, with a star on the better one and a line saying how much less energy one used.

### 🩺 Data check
- **More, Data check** looks through the last 90 days for things that seem wrong: a gap with no odometer readings, an odometer that went backwards, overlapping trips, trips that never closed, impossible speeds and trips with almost no distance. It only looks and never changes your data; each trip finding links to the trip.

### 📅 Year in review
- **More, Year in review** adds up a year: distance driven (from the odometer), trips, work and personal miles, energy used and efficiency, a bar for each month, your busiest month and longest trip, and charging (sessions, energy added, cost, share at home, and how much was lost as heat). Pick any year CarTrak has trips for, and **Copy summary** gives a few lines to paste.

### 🧭 Friendlier pages
- Empty lists (Trips, Charging, Places, Maintenance, Review) now say what to do next instead of a blank line.
- **More** has a search box that filters the pages and settings as you type.
- A trip now shows its average speed and has map links for both where it started and where it ended.
- Easier to use with a keyboard or screen reader: a Skip to content link, each page sets its title, bigger checkboxes, and animations are switched off if your device asks for less motion.

### 📤 More in the CSV exports
- **Charging** has an **Export CSV** button: every session with the energy added, the charger's own number and efficiency where entered, where it was, peak power and cost.
- The trips CSV now also has energy used, whether it is an estimate, and the named start and end places.

### 👁️ Easier to read in light mode
- A contrast check of every text and button color found that a few in light mode (links, green buttons and chips, amber and red labels) were slightly under the recommended contrast. They are a little deeper now. Dark mode already passed.

### 💽 Disk space
- **Status** now shows how much room is left on the disk CarTrak stores its data on (on the Docker image), warning when it is getting low and going red when it is nearly full, with a link to Free up space. A new optional alert, **Low disk space**, warns you at most once every three days. CarTrak adds readings all the time, so a full disk would stop everything.
- The **Car readings** line on Status now tells apart Ford answering from the car itself being asleep ("Ford is answering, but the car itself last reported 6 h ago. It is probably parked and asleep, which is normal.").

### 🔔 See what alerts were sent, and a guide to everything
- **More, Notifications** now ends with **Recent alerts** (admins): every alert CarTrak decided to send, and whether it reached a device, none had it turned on, or it was held back by quiet hours. If an alert you expected is not listed, CarTrak never decided to send it.
- **More, What CarTrak can do** lists every feature in one place with a line about each and a link, so nothing gets forgotten.

### 🚨 Parked watch (experimental)
- **More, Parked watch** uses the car's motion sensors to notice a bump on a parked car (another car's door, for example), and can alert you with **Bump while parked** and **Car alarm** (both off until you turn them on under More, Notifications). The bump level is a setting. Because the car only reports now and then, a short knock can be missed, so the page shows how many parked readings were checked, the strongest motion seen and what it noticed, to help you judge how well it works. See docs/places-and-maintenance.md.

### 📍 Trip rules can use your named places
- When you make a trip rule that applies only near a place, you can now pick one of the places you named under More, Places instead of typing coordinates.

### 🧾 Mileage log shortcuts
- **This week, Last week, This month** and **Last month** buttons build the log for that period in one tap.
- When a period covers more than one calendar week, the log also lists the miles for **each calendar week**, to check against a weekly sheet.

### ❄️ Range that knows about the weather
- The 20 to 80% line on Home now adds **"At today's 28°F: about 96 mi"** when today's temperature makes a noticeable difference from your recent average. It uses how far you really got on trips in that kind of weather (below freezing, 0 to 10, 10 to 20, and 20 °C and above).
- The **road-trip planner** has a **Weather** choice (today's temperature, your recent average, or any weather group), so you can plan a winter trip with winter range.
- **Trips** shows the range for each weather group next to the monthly table.

## 0.20.0

### 📺 Live fits one screen and follows what the car is doing
- The Live screen no longer needs scrolling. It shows the battery ring and six tiles that change with the car: **driving** (speed, this trip, trip time, miles to your reserve, range, driving today), **charging** (power, time to your target, energy added this session, range, energy left, outside temperature) or **parked** (range, energy left, doors, driving today, outside temperature, odometer). A small chip says which it is. The phone GPS tools moved into a collapsed section at the bottom.

### 🕐 Know when the car last reported
- The top of every page said "updated 6 minutes ago" even when that was just CarTrak asking Ford (for example after a server restart). It now says **car reported …**, using Ford's own update times, so it tells you when the car itself last sent something. A sleeping car simply shows an older time. Hover or press and hold it to see when CarTrak last checked Ford, and it says so if Ford is not answering. Live shows the same.

### 🧹 Free up space (optional)
- **More, Backup, Free up space** can remove the full raw Ford detail of older readings (you choose how many recent days to keep, and type TRIM to confirm) while keeping trips, charging, charts and each reading's main numbers. It works in small background steps, and the freed space is reused so the database stops growing for a while. See docs/backup.md.

### 🏠 Home: the last 7 days
- Home has a **Last 7 days** card: distance, trips, energy used and efficiency (tap it to change the unit).

### ♻️ Restore a backup from the app
- **More, Backup, Restore a backup:** upload a backup (or pick one you copied into the data folder's restore folder), see what is in it, type RESTORE, and CarTrak restarts with it. Your current data is kept beside it as `cartrak.db.before-restore-<time>`, so nothing is lost. See docs/backup.md.

### 🕒 Trip time and time driving today
- **Live** has two new tiles: **Trip time** (how long the current trip has been going, and when it started) and **Driving today** (all the time spent driving since midnight, including the trip in progress). Both keep counting between readings.

### ⏱️ Time: parked, driving and charging
- **More, Time** shows what share of the time the car was parked, driving or charging over the last 7 days, 30 days or year, a bar for each day, and where it was parked (Home, the office and other places you have named, or Elsewhere). It uses the trips and charging sessions CarTrak already recorded.

### 🗺️ Open the car in your map app
- The map on Home is now a link: tap it to open the car's spot in Google Maps. There are also **Open in Google Maps** and **Directions to the car** buttons (and **Open in Apple Maps** on iPhone), and the trip page has the same for where it ended. Nothing is sent to Google until you tap.

### 🔁 Tap an efficiency figure to change its unit
- Tap any efficiency number (on Trips, a trip, a journey or Efficiency insights) to switch all of them between miles per kWh, watt-hours per mile and miles per percent of battery. Metric devices get kWh per 100 km, Wh per km and km per percent. CarTrak remembers your choice.

### 🔌 Measure your charging loss
- On each charging session, admins can type in **kWh from the charger** (from your charger's app). CarTrak shows how much of it reached the battery, and after three sessions it uses your own measured average instead of a fixed 90% when it works out what charging cost.
- A new setting, **Slow-charging efficiency**, sets the number used until it has been measured.

## 0.19.0

### 🧳 Journeys
- On **Trips**, admins can select several trips and **combine** them into a journey (a road trip, or a work day with several stops). The journey page shows total distance, driving and elapsed time, the stops between legs (and how much you charged), energy used and efficiency, what charging cost, and a map of every leg. Trips stay as they are and can be removed from a journey at any time. See docs/road-trips-and-charging.md.

### 🔗 Share a trip or journey
- Admins can make a **public link** to a trip or a journey from its **Share** card. You choose the route (with the first and last quarter mile left out, the whole route, or numbers only), whether to show city names and the date and time, and when the link expires. The page needs no sign-in and shows only that, never your VIN, live position or other trips. The link is shown once; **More, Security, Shared links** lists every link and turns any of them off at once. See docs/sharing.md.

### 📍 Places, trip review and maintenance
- **More, Places:** name Home, the office and other places (from where the car is, or from a trip's start or end). Trips then read "Home → Office", and you can get an alert when the car arrives at or leaves a place.
- **Trips to review:** a banner on Trips and a quick screen to tap Work or Personal on trips that no rule could label.
- **More, Maintenance:** service reminders by distance, by time or both (tire rotation, cabin filter, brake fluid and so on), with an alert when one comes due and a Done button that starts the next interval.
- **Charging stopped early:** a new alert (off until you turn it on) for a charge that stops while plugged in at a lower level than your sessions usually end at. See docs/places-and-maintenance.md.

### 📈 Efficiency insights and a weekly summary
- **More, Efficiency insights:** your miles per kWh by outside temperature and by month, and your most and least efficient trips, so you can see what the cold really costs.
- **Weekly summary:** an optional Sunday evening notification with the week's distance, trips, charging and efficiency (turn it on under More, Notifications).

### 🧭 Road-trip planner
- **More, Road-trip planner:** enter a trip distance, your starting charge and your average charging power. It uses how far your battery really goes per percent (from your own driving), your reserve and your charge target, and says how many charging stops the trip needs, about how far in each one is, what to charge to, and about how long. It is a guide, not a route.

### 🏠 Home Assistant
- A read-only address for smart-home tools (`/api/ha/state`, read with a phone-tracking device token) gives the battery, range, plugged in, charging, locked and a few more values, with no location or VIN. See docs/home-assistant.md for sensor examples.

### ☁️ Update the Cloudflare connector from the app
- When Cloudflare publishes a newer connector, **More, Remote access** says so and has an **Update** button. It downloads the file from GitHub, checks it against the checksum GitHub publishes, runs it once, switches the tunnel over, and puts the old connector back on its own if the tunnel does not reconnect within 90 seconds.
- **More, About** now lists the versions of CarTrak, the connector, SQLite and Go, and which of them have an update.
- The repository gets a weekly Dependabot check for newer libraries and base images.

## 0.18.0

### 🅿️ Parked drain
- The Health page has a new **Parked drain** section for electric cars: how much charge the car loses while it sits (not driving, not charging) over the last 7 and 30 days, as a percent a day and in kWh, with a note on whether that is normal and a list of the recent parked stays. It needs about half a day of parked readings.

### 🔋 Charging history and cost
- The **Charging** tab now shows the energy each session added (in kWh), and a **Charging cost** card with every month: sessions, kWh, cost, miles driven and **cost per mile**.
- Set your price per kWh at home (and away from home, and a currency symbol) in More, Settings. Mark one session as **Home** and CarTrak recognizes home from then on. Slow chargers include about 10% lost as heat.

### 💾 Backup
- **More, Backup** downloads a copy of your data: a small backup without the raw Ford readings, or a full one. Sign-in sessions are never included. See docs/backup.md for restoring.

### 🔐 Security
- **More, Security** shows who is signed in right now, with a button to sign a device out, and a **sign-in log** of the last 100 attempts (name, success or failure, device, address). Passwords are never stored.

### 📱 Phone shortcuts
- On Android, press and hold the CarTrak icon to jump straight to Live, Charging, Trips or Health.

## 0.15.0

### 🔌 Plug in tonight?
- A new alert, off until you turn it on under More, Notifications: one reminder in the evening (9 pm by default) if the car is parked, not plugged in and the battery is below 60 percent. An admin can change the time and the level in Settings. It is sent at most once a day, and only if the car has reported in the last two hours.

### 🧾 Mileage log rounding
- The mileage log now rounds **each trip up to the next whole mile** before adding the day, so 0.1 mi counts as 1 and 26.2 as 27. You can switch to the old way (add the day, then round to the nearest mile) with the new **Miles** choice on the Mileage log page.

## 0.14.0

### 🎨 A new look
- CarTrak now has a darker, calmer look with one green accent. Light mode follows the same design with deeper colors, and it still follows your phone or your choice under More, Display and data.
- **Home** shows the battery as a ring with the percentage and range inside, then Energy left and Battery temp, your reserve and the 20 to 80% figure. Charging and Vehicle statuses now read as plain words (for example Not ready, Disconnected, Park) instead of Ford's capital letters.
- **Live** uses the same ring on its big Charge tile. The ring turns amber at 20% or less and red at 10% or less.
- The bottom bar has an icon above each label, and on a wide screen the left sidebar has them too.
- Cards, buttons, chips, the Trips filters, the More menu and the Health and Data pages share one set of colors, sizes and corners. The day chart on Data follows the theme.
- Nothing about what CarTrak records or how it works has changed.

## 0.13.0

### ✅ Mileage check
- The **Mileage log** now has a **Does it add up?** card. It compares the car's odometer, which is exact, with the trips CarTrak recorded for the same weeks or dates, and says **Matches** or how many miles are missing.
- Every stretch of driving that no trip covers is listed with its time window and where the car was. **Add as work trip** or **Add as personal trip** fills it in from the odometer in one tap. When the miles sit right before or after a recorded trip (it was noticed or closed a little late), **Add to that trip** widens that trip instead.
- Admins only. See docs/mileage.md.

## 0.12.0

### 🔎 Car data, explained
- The **Car data** page now sorts everything the car reports into tabs: Summary, Battery and charging, Driving, Body and security, Settings and schedules, and Car. Every value has a plain-English name, and a **?** next to it explains what it means and shows Ford's own name for it. **All fields** still lists every raw value, with search.
- **Look back in time.** Pick any day and time (or step Earlier and Later) to see the car as it was then. A chart of the day shows the battery, with bands for driving and charging; tap it to jump to that moment.
- Values that Ford updates only when the car wakes now say how old they are.

### 🧭 How far between two charge levels
- Home now shows how far a stretch of battery goes, for example **20 to 80%: 142 mi**, worked out from how you have really been driving lately (so it follows the seasons), not just from the car's guess. The low end is your road-trip reserve and the high end is your charging target.
- **Trips** has a **Range by month** table, so you can see the winter drop and the summer recovery. It fills in as trips build up.

### 🔋 Calmer 12 V battery warnings
- A parked car often shows a middling 12 V reading (around 12 V and 50 to 70 percent) until it wakes and recharges, so the Health page now explains that instead of flagging it. Real problems (below 11.9 V or 50 percent) are still flagged.
- If a shop has tested the battery and it is fine, an admin can tap **It was tested and is fine** to pause its warnings and alert for 90 days.
- Clearer wording on the main battery's hidden top and bottom buffer, and a note that the 12 V percentage is the car's own estimate.

## 0.11.0

### ⚡ Road trips and charging
- **Energy left** is now on the Live screen (kWh, and about how much the battery holds when full).
- **Miles until your reserve:** set the level you do not want to go below (default 20%) under More, Display and data. Home and Live show how far you can go before the battery reaches it.
- **Time until a charge level:** while charging, pick 80%, 90%, 100% or any percent and see how long it will take and around what time. For 100% it uses the car's own figure; otherwise it is an estimate from the charging power, allowing for fast chargers slowing down above 80%. On Live, tap the tile to switch between 80, 90 and 100.
- **Trip efficiency:** every trip shows the energy it used and its miles per kWh (or kWh per 100 km if your units are metric), in the trip, in the trips list and as a total for the period. Trips from before this update are estimated from the battery percentage and marked as approximate. While a trip is in progress, Live shows its efficiency so far.

### 📍 GPS in the background
- CarTrak now accepts the **OwnTracks** app (iPhone and Android) and the docs have step-by-step settings for **GPSLogger** (Android), so your phone can send its position while the app is closed, with nothing to keep open. The phone sign-in also accepts a user name and password (the token is the password), which these apps use. See docs/phone-tracking.md.

## 0.10.1

### 🛠 Health tab fix
- Fixed the Health tab seeming to do nothing on a phone. Every reading carries a large block of raw data, and on an install that had been running for a few days the charts read through all of it, which was slow enough to stall the whole app. The charted values now live in a small table of their own: new readings add to it as they arrive, and readings from before the update are copied across quietly in the background (the charts fill in as it goes, and the app stays responsive).
- The Health page now says "Loading vehicle health…" while it works, and shows a clear message with a Try again button if something goes wrong, so it never looks dead.
- Trip routes, the trip distance check and the Trips map read only positions and speed instead of every field, so they open faster too.

## 0.10.0

### 🔔 Alerts on your phone
- New More, Notifications: turn alerts on for this phone or computer and get a notification for charging complete, plugged in, unplugged, a low tire, a low 12 V battery, the car left unlocked, a trip started or ended, and no readings from the car for hours. Each device chooses its own alerts and can set quiet hours, and a button sends a test alert.
- Each alert is sent once, and a restart does not repeat what was already true.
- Private: alerts are encrypted for your device, say things like "Charging complete" and never include a location. They go through your browser maker's push service, which can see that one was sent but not what it says. CarTrak only contacts real push services.
- Needs the secure address (for example your Cloudflare address). On iPhone, add CarTrak to the Home Screen first. See docs/notifications.md.
- Status shows an Alerts line.

## 0.9.1

### 🛠 When Ford's servers return an error
- If Ford answers a request with a server error such as HTTP 500, CarTrak now shows what Ford said alongside it (with any vehicle ID removed) instead of only the number, so the cause can be seen on the Status page and in the log.
- It is treated as Ford having trouble, not as a problem with your setup: Status says so in plain words, and CarTrak tries again after a minute, then 2, 4 and 8 minutes, never waiting more than 10. Before, it waited for the normal parked interval, which could be much longer.
- When Ford answers normally again the message clears by itself.

## 0.9.0

### 🩺 A Status screen that says what to do next
- A new Status page (More, then Status) checks the whole setup in plain language: Ford settings entered, Ford linked, last reading and how old it is, remote access, and whether an update is available. Each line says what is wrong and has a button to fix it.
- Home shows a short strip with the same result, so a problem is visible without opening any menu. Anyone signed in can see it; it never shows keys or passwords.

### 🚗 A dashboard built from the real car data
- New Doors and locks card: whether the car is locked and which doors, the tailgate or the hood are open.
- New Health card: warning lights the car reports and low tire pressure alerts, or "No warnings".
- Tire pressures and the lock state now read Ford's real layout (a list per wheel and per door), so the Tires card and the Locks line fill in with a linked car.
- The built-in demo car now uses the same layout, so these cards can be previewed without a car.

### 🧾 Mileage log
- New More, Mileage log: pick calendar weeks (for example 26-28) or dates and get a business mileage log in your sheet's layout: Date, Origin City, Destination City, Description, Total Miles Driven, plus Total Mileage Recorded. One row per day; the destination list is the city where you parked after each trip, each city once and in order, without your starting city.
- One-tap copy, tab-separated so it pastes cell by cell, with or without the headings and total.
- City names come from OpenStreetMap's Nominatim service, asked by your server only when you build a log, with locations rounded to about 100 m, and every answer is remembered. It can be switched off in Settings (then only remembered places are named). See docs/mileage.md.

### 🗺️ Maps
- Trip pages now show the route on a real map with start and end markers (your phone's GPS track when there is one). Home shows a small map of where the car is, and Trips has a new Map view with every trip in the chosen range, work in amber and personal in blue; tap a route to open the trip.
- Drag to move, + and − buttons, mouse wheel or two-finger pinch to zoom.
- Private by design: your phone or computer never contacts the map provider. CarTrak's server downloads the tiles from OpenStreetMap, keeps a copy on disk and passes them on, so repeat views are fast. Only signed-in people can load tiles, and only real map images are passed on.
- Settings: Maps can be switched off (plain route lines, no map requests), and a different tile server and credit line can be set. See docs/maps.md.

### 🩺 Vehicle health tab
- New Health tab: voltages, battery health, oil and fuel, tires, trouble codes, Ford alerts and warning lights, with a "Needs a look" list at the top and small daily trends for the 12 V battery, the main battery's estimated capacity and oil life.
- Built for each kind of car. CarTrak now knows whether the car is electric, plug-in hybrid, hybrid or gas: it reads Ford's vehicle record first, falls back to the data the car reports, and you can choose the type under Settings, Vehicle type. Oil life and fuel show for cars with an engine; the main battery shows for electric and hybrid cars. Battery cars never show oil, even though Ford lists an oil life of 100 percent for them.
- Home follows the type too: gas and regular hybrid cars have no Charging tab or card, a gas car has no Battery card, and cars with an engine get a Fuel and oil card.
- Each value says how old it is when it is more than two hours old (a parked car sends some values only when it wakes). The Home battery card says so as well.
- Fixed: Home showed the outside temperature as 32 °F on the real car because Ford's ambient temperature field reads 0; the real outside temperature is used now. Door names no longer say "Unknown".
- The demo car now uses the real data layout and can act as an electric, gas, hybrid or plug-in hybrid car (CARTRAK_DEMO_POWERTRAIN).

### 📱 Phone tracking (optional)
- New More, Phone tracking page: pair a phone, get a one-time token, and copy ready-made request examples. A test button confirms the phone can reach CarTrak, and the page lists the latest messages so you can see an automation fired.
- When a phone says a trip started, CarTrak asks Ford right away and checks more often until Ford reports the trip; when it says the trip ended, CarTrak checks again so the trip closes promptly. Ford still decides what counts as a trip, so odometer and battery numbers stay exact.
- GPS points from the phone are drawn on the trip page, a finer route than Ford's readings.
- Live page: Record GPS from this phone sends the phone's location while the page is open (with a queue so a dropped signal loses nothing).
- Step-by-step recipes for Tasker, MacroDroid and iPhone Shortcuts are in docs/phone-tracking.md.
- Tokens are stored only as a hash and can be removed any time. Optional setting: delete phone GPS points after a number of days (off by default; trips are never deleted).
- Status shows a Phone tracking line.

### ✨ Tidier Home and sidebar
- The tabs in the left sidebar on a computer or wide tablet were stretched to fill the whole column. They are now compact, normal-sized rows.
- The Getting started card on Home now only shows while Ford is not linked yet. Once you are set up it is gone; the checklist stays available under More, Getting started.

### 🧹 Under the hood
- The Ford tests were split into smaller files.

## 0.8.2

### 🐢 Playing by Ford's request limit
- Ford only allows about one request per endpoint every 30 seconds, and answers HTTP 429 ("too many requests") to anything faster. CarTrak was making a burst of six requests right after linking and your Test Ford connection added six more, so every endpoint was refused. CarTrak now never asks the same endpoint more than once per 30 seconds, reuses a recent answer instead of asking again, adds only one of the slower endpoints per reading, and after a 429 stops sending anything and waits (30 s, then 1 min, 2 min, up to 10 min).
- Road trip mode now means a reading every 30 seconds (it was 10, which Ford would refuse). All polling intervals have a 30 second minimum.
- Test Ford connection says "Rate limited" with how long to wait, instead of a plain red error.

## 0.8.1

### 🔗 Connecting Ford (the real fix)
- CarTrak now starts the sign-in at Ford's real sign-in address (the same one other FordConnect Query apps use), so Ford shows its own sign-in and vehicle-consent pages. Earlier versions started at a back-end address that always failed with "RESTApiCallForUserInfo".
- The sign-in link now carries a short `state` (Ford rejects a long one) and no extra scope fields.
- The token step tries Ford's documented form and, if Ford refuses it with a 400, again without the scope field.
- CarTrak now understands the real telemetry layout (battery, range, odometer, location, ignition, plug and more), so the dashboard fills in once Ford is linked.

## 0.8.0

### ✨ A cleaner More page
- More is now a short grouped menu (Connections, Trips, This device, Admin, Help) with status chips, instead of one long page of cards. Each row opens its own page: Ford connection, Remote access, Display and data, People, Settings, Getting started, About and updates.
- On tablets and computers the menu uses two columns.
- Slightly darker gray text in light mode for easier reading.

### 🧹 Under the hood
- Source files were split so none is over 200 lines; behavior is unchanged.

## 0.7.3

### 🔗 Connecting Ford
- CarTrak now asks Ford which redirect address is registered on your app and uses one Ford accepts. Pressing Connect Ford checks first, so Ford is never sent an address it will reject (the repeating redirect_uri_mismatch error). The Connect Ford page shows what Ford said for each address.
- Clearer message when Ford accepts the address but reports its own sign-in problem.
- The Ford sign-in address setting accepts a link copied from Ford documentation, with {client_id}, {redirect_uri}, {state} and {scope} placeholders.

## 0.7.2

### 🔗 Connecting Ford
- The Connect Ford page now shows the exact address CarTrak asks Ford to return to, with one-tap choices: your site address, the address with /ford/callback, or http://localhost. Tap the one that matches what is registered on your Ford app. If the value is fixed by an environment variable or config.yaml, it says so.

## 0.7.1

### 🔗 Connecting Ford
- Ford apps registered with just the site address (no /ford/callback path) now work: CarTrak finishes the link when Ford returns to the site root.
- The Remote access and Admin pages have buttons for the address with and without the path, so it can match exactly what is registered on your Ford app.

## 0.7.0

### 🕒 Time zones that follow the car
- Each trip now uses the time zone where it **starts**, worked out from its location (no internet needed). "Weekdays 8am to 5pm" means 8 to 5 wherever you are driving, so a trip in New York is judged on New York time even if your server is in Minnesota.
- Trip times are shown in that local time with the zone name (for example "10:16 PM EDT") when it differs from your device. A per-device setting (More, Show trip times in) switches to your own device time.
- The time zone setting is now the **default**, used only when the car location is unknown. The CSV export gains local start time and time zone columns.

### 📴 Works without signal
- Home keeps the last readings on your device and shows them with an "offline" banner when there is no connection. Your server keeps collecting from Ford while your phone is offline, so nothing is lost; open the app again to see it all.

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
