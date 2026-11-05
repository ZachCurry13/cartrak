//go:build race

package rebuild

// raceOn says the tests run under the race detector, which makes the SQLite replay about twenty times slower.
const raceOn = true
