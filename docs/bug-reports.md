# Bug reports from the app

At the foot of every page (for administrators) is **Send a bug report**. It captures what is on screen, takes a short note, and sends
the lot to a **private** GitHub repository you set up once, so a problem can be looked at without a pile of screenshots in a chat.

## What is sent
- Your note.
- A **picture of the page**, drawn by the app from the page itself, as long as the page.
- The page's **text**, the page address, the app version, and the **screen it was on**: size, phone, tablet or computer, touch or mouse,
  installed app or browser tab, theme, and browser details.
- Optionally (ticked by default) the **debug report** for the last 24 hours (the same one as More, Debug report).

It goes from your CarTrak server to GitHub; the browser never sees the token. A report can contain your car's readings and what is on
your screen, so use a **private** repository, never the public CarTrak one.

## Set it up once
1. On GitHub, create a **private** repository, for example `cartrak-reports`. Tick **Add a README** so it has a first commit.
2. Create a **fine-grained personal access token** (GitHub, Settings, Developer settings, Personal access tokens, Fine-grained tokens):
   repository access **only that repository**, permissions **Contents: Read and write** and **Issues: Read and write**. Copy the token.
3. In CarTrak open **More, Settings** and fill in **Bug report repository** (`your-name/cartrak-reports`) and **Bug report token**.
   The token is stored on the server only and is never shown again.

Until both are set, the button explains this instead of sending.

## Reading a report (for whoever helps)
Each report is an **issue** in the reports repository and a folder `reports/<id>/` with `report.md` (note, screen, page text),
`screen.png` and `debug.txt`. With the gh tool signed in as the repository's owner:

```bash
gh issue list -R OWNER/cartrak-reports
gh issue view 12 -R OWNER/cartrak-reports
gh api repos/OWNER/cartrak-reports/contents/reports --jq '.[].name'
gh api repos/OWNER/cartrak-reports/contents/reports/ID/screen.png --jq .content | base64 -d > screen.png
```
Read `screen.png`, `report.md` and `debug.txt` before changing anything. Never copy a report's contents into the public repository.

## If sending fails
The message says why: the token expired (create a new one), the token cannot write to that repository (check Contents and Issues are
set to Read and write and the repository is the one named), or GitHub could not be reached.
