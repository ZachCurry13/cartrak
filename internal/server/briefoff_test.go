package server

// Every other test reads fresh answers: they change the database directly and read it back within the reuse time.
func init() { briefDisabled = true }
