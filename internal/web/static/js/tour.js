// The guided tour: what each step says, which page it opens and which part of the screen it points at. A step with no target
// is a plain note. Steps for administrators only are left out for everyone else.
const first = (...sels) => () => { for (const s of sels) { const e = document.querySelector(s); if (e) return e; } return null; };
const tab = (href) => first(`#tabs a[href="${href}"]`);

export function tourSteps(session) {
  const admin = session.me.role === 'admin';
  const steps = [
    { icon: '👋', title: 'Welcome to CarTrak', text: 'CarTrak keeps a private record of your car: every drive, every charge, the battery and the tires. It collects from Ford all day, so nothing is missed when your phone has no signal. This guide takes about two minutes.', route: '#/' },
    { icon: '🔋', title: 'Home', text: 'The ring is the battery. Under it, small chips say whether the car is driving, charging or parked, plugged in, and locked. Below are today, this month, the last trip, and anything that needs your attention.', route: '#/', target: first('#view .dash .card', '#view .card') },
    { icon: '📍', title: 'Trips', text: 'Every drive is recorded on its own, with no button to press. Open the Trips tab to see them.', route: '#/trips', target: tab('#/trips') },
    { icon: '🧾', title: 'A trip card', text: 'Each card shows the route, how far and how long, where it started and ended (a place you named, otherwise the city), the battery it used, and how efficient it was against your other drives. Tap a card for the full trip.', route: '#/trips', target: first('#view .tcard') },
    { icon: '🔎', title: 'Filter, map and export', text: 'Narrow the list by Work or Personal and by 7 days, 30 days or a year. Switch to Map to see every route at once, and use Export CSV for a spreadsheet.', route: '#/trips', target: first('#view .seg') },
    { icon: '⚡', title: 'Live', text: 'What the car is doing right now: speed and power while driving, charging speed and time left while plugged in, and a parked summary otherwise.', route: '#/live', target: tab('#/live') },
    { icon: '🔌', title: 'Charging', text: 'Every charging session with the energy added, the cost, how fast it went and where. Enter what you paid, or the kWh the charger says, and CarTrak works out the real loss.', route: '#/charging', target: tab('#/charging') },
    { icon: '🩺', title: 'Health', text: 'Battery health, the 12 V battery, tires, and any warning the car reports, in plain words. Calm unless something needs attention.', route: '#/health', target: tab('#/health') },
    { icon: '🗂️', title: 'Data', text: 'Every value the car reports, grouped and explained, with how old each one is. A time machine lets you look at any past moment.', route: '#/data', target: tab('#/data') },
    { icon: '⋯', title: 'More', text: 'Everything else lives here: Reports, Cars, Places, trip rules, notifications and backup. Tiles that need an administrator are hidden for other people.', route: '#/settings', target: tab('#/settings') },
    { icon: '🔍', title: 'Search', text: 'Find any page or setting by name, from anywhere.', route: '#/', target: first('#srch') },
    { icon: '🔔', title: 'Inbox', text: 'The alerts CarTrak has sent, across your cars, with the number you have not opened. Tap one to jump to what it is about.', route: '#/', target: first('#bell') },
  ];
  if (admin) {
    steps.push(
      { icon: '📊', title: 'Reports', text: 'Download a spreadsheet of your trips or charging, build your own with just the columns you want, or make a mileage log for expenses.', route: '#/reports', target: first('#view h2') },
      { icon: '📣', title: 'Notifications', text: 'Choose which alerts you want on this phone: trip start and end, charging, a door left open, low battery and more. Turn on notifications here first.', route: '#/notifications', target: first('#view h2') },
      { icon: '🚙', title: 'Cars and people', text: 'Add another car, rename one, and choose who can see it. People with the viewer role can look but not change anything.', route: '#/cars', target: first('#view h2') },
    );
  }
  steps.push(
    { icon: '🔗', title: 'Share a trip or a day', text: 'The share button on a trip, or on a whole day, makes a short private link. Anyone with it sees just that, and you can turn it off any time.', route: '#/trips', target: first('#view .tcard .ico') },
    { icon: '📴', title: 'No signal?', text: 'CarTrak keeps collecting in the background. On your phone, the screens you have opened remember their last answer, so they still open with a small "Offline" note. Changes need a connection.', route: '#/' },
    { icon: '✅', title: 'That is it', text: 'Tap the ? at the top any time to run this guide again. The Help page has quick settings and an explanation of each part of the app.', route: '#/', target: first('#help') },
  );
  return steps;
}
