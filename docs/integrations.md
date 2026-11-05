# Integrations

**More, Integrations** lists every outside service CarTrak uses, one card each, so keys and switches are in one place.

Each card says what the service is for, **what it sends** (so you can decide), and whether it is **On**, **Off**, **Set up** (needed and missing) or on **its own page**. A service that has settings shows them on the card: tap **Settings**, change them and **Save**. A key is never shown again once saved (leave the box blank to keep it), and a value set in an environment variable or `config.yaml` shows as locked.

| Card | What it is for | Needs |
| --- | --- | --- |
| Ford (the car) | The car's readings, trips and charging | Ford client ID and secret (its own page) |
| Remote access (Cloudflare) | Reaching CarTrak away from home, and the address links use | Its own page |
| Phone GPS | The exact route of a drive | Its own page |
| Maps (OpenStreetMap) | Map tiles under routes | On or off |
| City names (OpenStreetMap) | Naming places, address search | On or off |
| Road estimates | Real road distance and time | On or off |
| Weather (Open-Meteo) | Wind, rain and temperature | On or off |
| Gas prices (EIA) | Comparing charges with gas prices | A free key |
| Superchargers (supercharge.info) | Which Tesla sites other cars can use | Nothing: the public list is downloaded |
| Electrify America (NLR charger list) | Adds Electrify America chargers to long trips | A free key from developer.nlr.gov/signup |
| Push alerts | Alerts on this phone | Its own page |

## Data updates
At the top of the page, **Data updates** lists the lists CarTrak downloads and keeps: the Supercharger list (supercharge.info), the Electrify America list (the NLR charger list, needs its key) and the weekly gas prices (EIA, needs its key). Each shows when it was last updated and how many entries it holds. CarTrak checks once a day, when the app is opened, whether something newer is out: the Supercharger list is compared with the server's own identifier for the file, the others by age (a month for chargers, eight days for gas prices). A list with something newer shows **Update available** and puts the red dot on More; **Update** fetches it and **Check now** looks immediately. Nothing is fetched until you tap, except the Supercharger list refreshing itself when you plan a long trip and it is over a week old.

The same card says whether a newer CarTrak is out (this uses the update check switch in More, About). CarTrak cannot update itself: stop and start it on the server.

## Test connection
Weather, road estimates, city names, gas prices and Electrify America have **Test connection**: it asks the service once and says whether it answered and the key works. The last result stays on the card.

## Needs attention
A small red dot appears on the **More** tab (administrators only) when Ford is not set up, or when the last test of a service that is on failed. It clears when the problem is fixed or a new test works.
