# Reports

**More, Reports** (also a button on the Trips page) has a report you can download, and links to everything else that adds your driving up.

## Download a report

Choose the report, then what goes in it, then download:

- **Standard report.** The usual mileage log, one row per trip: Value, Miles, Mileage Rate, From, To, Date, Purpose, Track Method, Notes, Vehicle, Time Started, Time Ended, Duration (hours), Time Zone. Untick any column you do not want, or add more: day of week, **energy used (kWh)**, **efficiency**, **energy cost**, **cost per mile (or kilometre)**, **MPGe**, **average speed**, battery and odometer at each end, top speed and trip ID. An Excel file has a summary above the table (miles and value for each category, and a total) and a second sheet, **Totals By Label**. A CSV has just the table.
- **Custom reports.** A saved layout: its own columns in its own order, its own headings, columns with a fixed value on every row (a company name, a cost centre) and whether the summary is included. Make one with **New custom report**, then run it whenever it is needed. Only administrators make or change them; anyone can run them.

Whichever report, you choose each time:

- **Categories**: all, one, or several (Work, Personal, Medical and your own).
- **Days**: a first and last day, or calendar week numbers such as `26-28` for a year.
- **File type**: Excel or CSV, and miles or kilometres (it follows the units set on the device).

**Efficiency** is shown the way you chose on the report page, which starts as the unit set on the device: distance per kWh (miles per kWh, or kWh per 100 km), watt-hours per mile (or km), or distance per percent of battery.

**Energy cost** is the energy the trip used times what a kWh actually cost you over the days chosen: the cost of the charging sessions in them divided by the energy they added (receipts you entered count), or your home price if none of them has a cost. **Cost per mile** is that divided by the distance. **MPGe** is miles per gallon-equivalent (33.7 kWh to a gallon).

**Value** is the distance times the mileage rate for the trip's category: the category's own rate, or for Work the general mileage rate under More, Settings. A category with no rate shows 0.

**From** and **To** show the named place (More, Places) where there is one, otherwise a city CarTrak has already looked up for the Mileage log. Nothing is fetched while a report is made, so a trip with neither is left empty.
