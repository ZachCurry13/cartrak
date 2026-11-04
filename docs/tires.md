# Tires and winter tires

**More, Tires** keeps a log of the tire sets the car has worn, and compares winter tires with the others at the same temperature.

## The log
Add each set with a name, a kind (winter, summer, all-season, other) and the day it went on; the date can be in the past. A set stays
on until the next one is added. **Winter tires on today** and **Back to summer tires today** add a set with one tap. Every trip then
says which tires it was on (under its date). Removing a set puts the one before it back on for those dates. Each car has its own log.

## Do the winter tires change the range?
Cold air costs energy by itself, and winter tires go on when it turns cold, so a plain before and after would measure the season. The
comparison avoids that:
- Trips on winter tires and on other tires are put in groups by **outside temperature** (bands about 9 F wide) and by **kind of
  driving** (city, or highway at an average of 45 mph or more).
- Only groups where **both** kinds of tire have at least 20 miles and two trips are compared.
- Trips on wet or snowy roads are left out, because winter tires meet more of them (this needs the weather lookup to be on).
- The result is the winter tires' energy use against the other tires' in those groups, weighted by how much each group has driven,
  with the lowest and highest group shown.

It says **not enough overlap yet** until the matched driving reaches about 100 miles, and calls the answer **fair** until it reaches
about 250 miles across three or more temperatures, then **good**. A result more than 15% either way is flagged, because tires rarely
change efficiency that much.

Honest limits: CarTrak's log starts when you start using it, so a first winter has no cold-weather driving on the old tires to compare
with; the overlap comes from the days around the swap, and next winter is much better. Tire pressure falls in the cold and wind, load
and route also move efficiency, so a few percent either way is noise.

## The reminder
**Tire swap reminder** (Notifications, off until you pick it) tells you once a year in autumn when the daily high has stayed under
45 F for three days and winter tires are not logged as on, and once in spring when it has stayed over 50 F for five days with winter
tires on. It uses the car's own outside temperature, and stays quiet unless you keep a tire log.
