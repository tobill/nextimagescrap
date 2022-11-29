package imports

import "time"

// Media defines the storage form for source-media objects
type SourceMedia struct {
	Key          string
	Path         string
	Mimetype     string
	Checksum     string
	CreationDate time.Time
	Id           int
}

type SourceChecksum struct {
	Key     string
	Sources []string
}
