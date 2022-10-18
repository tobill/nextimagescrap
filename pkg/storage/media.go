package storage

// Media defines the storage form for source-media objects
type DbSourceMedia struct {
	Key      string
	Path     string
	Mimetype string
	Checksum string
}

type DbSourceChecksum struct {
	Key     string
	Sources []string
}
