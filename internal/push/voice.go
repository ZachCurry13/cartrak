package push

// Holiday voices, for people who turn them on: around a holiday the friendly alerts talk in its voice (Santa at Christmas, a friendly
// ghost at Halloween, a leprechaun on St. Patrick's Day). The facts stay exactly as they were: only the title changes, and a short
// opener goes in front of the body. Spooky, never scary. Warnings never use a voice.

type voice struct {
	Opener string            // goes first in the body of an alert with no line of its own
	Titles map[string]string // a title in the voice, for the alerts most often seen
}

func lines(complete, in, out, start, end string) map[string]string {
	return map[string]string{ChargeComplete: complete, PlugIn: in, PlugOut: out, TripStart: start, TripEnd: end}
}

// voices are keyed by the holiday's emoji (the same one the alert already carries).
var voices = map[string]voice{
	"🎄":  {"Ho ho ho!", lines("Ho ho ho! Fully charged", "Ho ho ho! Plugged in", "Off to deliver cheer: unplugged", "Jingle all the way: trip started", "Ho ho home! Trip ended")},
	"🎃":  {"Boo! (a friendly one)", lines("Boo-tifully charged", "Plugged in, spook-tacular", "Unplugged: off to haunt the road", "Boo! Off we go", "Trip ended: treats, no tricks")},
	"☘️": {"Top o' the mornin'!", lines("Fully charged, pot o' gold ready", "Plugged in, top o' the mornin'", "Off ye go: unplugged", "Off on a wee adventure", "Home safe, bless ye: trip ended")},
	"🐣":  {"Hoppy day!", lines("Hop to it: fully charged", "Plugged in, hoppy days", "Unplugged: hop along", "Off we hop", "Hopped home: trip ended")},
	"💕":  {"Aww!", lines("Full of love: charged", "Plugged in, love is in the air", "Unplugged: off to spread the love", "Off on a date with the road", "Home sweet home: trip ended")},
	"🎆":  {"Boom!", lines("Fully charged, ready to sparkle", "Plugged in, fireworks soon", "Unplugged: rockets' red glare", "Off with a bang", "Trip ended with a bang")},
	"🦃":  {"Gobble gobble!", lines("Thankful for a full charge", "Plugged in, feeling grateful", "Unplugged: off to the feast", "Off to the feast", "Home for the feast: trip ended")},
	"🎉":  {"Cheers!", lines("Fully charged for the new year", "Plugged in: cheers!", "Unplugged: into the new year", "Off to ring it in", "Trip ended: cheers!")},
}

// speak puts an alert in the holiday's voice (identified by its emoji), leaving its facts alone.
func speak(a Alert, emoji string) Alert {
	v, ok := voices[emoji]
	if !ok {
		return a
	}
	if t, own := v.Titles[a.Kind]; own {
		a.Title = t
		return a
	}
	a.Body = joinLine(v.Opener, a.Body)
	return a
}
