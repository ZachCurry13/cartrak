# Bug reports from the app

At the foot of every page (for administrators) is **Send a bug report**. It keeps a **text copy of the page** with a short note, inside CarTrak, so a problem can be read later instead of described. Nothing leaves your server unless you make a link.

## What is kept
- Your note.
- The page's **words**, the page address and the app version.
- The **screen**: size, phone, tablet or computer, touch or mouse, installed app or browser tab, theme, and browser details.
- **Layout facts**: whether the page scrolls sideways, what is wider than the window, what is cut off, which buttons are under 38 px high and which text is hard to read. They stand in for a picture.
- Optionally (ticked by default) the **debug report** for the last 24 hours (the same one as More, Debug report).

No picture is taken. Reports are plain text.

## More, Bug reports
- Every report, newest first. Open one to read it, or delete it.
- **Keep reports** until you clear them, or for 30, 60 or 90 days (older ones are removed by themselves).
- **Clear older than 30 days** and **Clear all**.

## A link to read them
**Make a link** on the Bug reports page (works for 1, 7 or 30 days). The address is shown once, with a Copy button; give it to whoever is helping. Opening it shows every report as plain text, with no sign-in, so it can be read by a tool as well as a person. It needs CarTrak's public address (More, Remote access).

The secret is in the address: anyone who has it can read every report, including the debug report, until it expires or you press **Stop the link**. Making a new link replaces the old one, and the page shows how many times it was opened. A wrong guess counts against the same lockout as wrong passwords. Each report is also at `/r/<secret>/<number>`.

Reports can contain your car's readings and what was on your screen, so make the link only when you need it and stop it afterwards.
