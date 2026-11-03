package server

// upstreamFailed is the status for "a service CarTrak asked (routing, address lookup, push) did not give an answer", with the reason in
// the message. It is not 502: the app reads 502, 503 and 504 as "CarTrak itself is restarting" and replaces the message with that.
const upstreamFailed = 424
