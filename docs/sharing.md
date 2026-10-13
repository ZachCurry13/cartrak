# Sharing a trip or journey

Admins can make a public link to one trip or one journey (several trips combined). Anyone with the link can open a small page without signing in. It shows only what you chose when you made the link.

## Make a link
Open a trip or a journey and use the **Share** card:
- **Route:** *Route, leaving out the first and last quarter mile* (the default, so your home or destination is not shown), *Numbers only*, or *The whole route*. A trip too short to trim shows no route.
- **City names:** optionally the city where it started and where it ended (looked up once, when you make the link).
- **Date and time:** optional. Without it, only the driving time is shown.
- **Expiry:** 7, 30 or 90 days, or never.

The page shows the distance, driving time (and for a journey the elapsed time and number of stops), the energy used and the efficiency, and the route on a map. It never shows your VIN, the car's model, the car's live position, your other trips or anything about your accounts.

You see the link **once**, when you make it, and can copy it then. CarTrak keeps only a check value of the link, so it cannot show it again; make a new link if you lose it.

## Turn a link off
**More, Security, Shared links** lists every link with its title, what it shows, when it was made, when it stops working and how many times it was opened. **Turn off** stops it immediately.

## How it is protected
- The link contains a long random secret. Wrong guesses are slowed down and locked out the same way wrong passwords are.
- The page and its data are not cached, are marked so search engines skip them, and send no referrer.
- The map tiles come through CarTrak itself, only for the area around the shared route and not zoomed in closer than street level, so the link cannot be used as a general map service.
- Revoked and expired links answer "not found", the same as a link that never existed.

## Using it from outside your home
The link only works where CarTrak can be reached. To share with people outside your home, set up **Remote access** first (see [Cloudflare](cloudflare.md)); then the link uses your public address. If you protect the address with Cloudflare Access, add a rule that lets the paths starting with `/s/` through without signing in, or the people you share with will be asked to log in. Do not open anything else.
