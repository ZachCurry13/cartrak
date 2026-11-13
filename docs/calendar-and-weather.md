# Calendar, weather warnings and free plugs

## Calendar link
Under **More → Integrations → Calendar (ICS link)**, paste the private address of the calendar your appointments are on:

- **Google Calendar**: Settings, pick the calendar, **Integrate calendar**, **Secret address in iCal format**.
- **Outlook**: Settings, Calendar, Shared calendars, **Publish a calendar**, then copy the ICS link. Some workplaces turn publishing off; a personal calendar with the appointments copied in works the same way.

Anyone with that address can read the calendar, so CarTrak keeps it on the server like a key: it is never shown again, never sent to the browser and never logged. CarTrak downloads the calendar at most every half hour and only reads it; nothing is sent to the calendar service.

**What it does**
- **Home** shows **Today** (from 6 pm, **Tomorrow**): the appointments with the place each one goes to, and **Plan today's route**.
- The route planner's **Stops** tab has **Today's appointments** and **Tomorrow's**. They fill the stops in the order of the day, untick **Best order**, and set the leaving time to half an hour before the first appointment.
- A trip that ends within about 400 m of an appointment's place, from an hour and a half before it starts until it ends, gets the appointment's title as its note. With **Trips to calendar appointments** set to **Work**, it is counted as work too (a choice you can still change on the trip).
- At 7 pm, the **Tomorrow's appointments** alert (if you chose it under Notifications) gives the first appointment, the low and any snow there (with Weather on), any road warnings at the appointments, and how much battery the day takes with what to charge to.

**How an appointment finds its place**: first among **Your accounts** (Paste a list) and your saved places, by the appointment's location or title; otherwise the location is searched like an address near Home, once, and the answer is kept. A location that finds nothing is tried again the next day. All-day appointments have no place.

Repeating appointments are followed for daily, weekly (including every other week and several days a week), monthly (by date or "the second Tuesday") and yearly rules, with an end date or a count, dates taken out, and single occurrences moved or cancelled.

## Weather warnings
**More → Integrations → Weather warnings** switches on the US National Weather Service warnings (free, no account). Only a position rounded to a tenth of a degree (about 10 km) is sent, at most every ten minutes for each area.

They show on Home (warnings where the car is), on a planned route (each road warning once, at the first stop it covers) and in the evening heads-up. The **Weather warning** alert sends the road warnings where the car is (winter storm, ice, snow, flood, fog, high wind, tornado and the like, at advisory level or above), each one once. Heat and air-quality notices are shown but not sent. Outside the United States the service has nothing to give.

## Free plugs now
With the Google Places key (the one Paste a list uses), CarTrak can ask Google how many plugs are free at chargers: **Chargers near the car** on the Charging page (within about 5 miles), and **Free plugs now?** under each planned charging stop of the Long trip planner and the route planner. Google gets the counts from the networks that report them; where a network does not, the plugs are listed without a count.

Each check is one lookup billed to your Google key, in the pricier tier of Google's Places pricing, so CarTrak never asks by itself: only when you tap. The answer is kept for five minutes. Set a small budget alert on the key in Google Cloud.
