package storage

import (
	"bytes"
	"encoding/gob"
	bolt "go.etcd.io/bbolt"
	"log"
	"nextimagescrap/pkg/imports"
	"path/filepath"
)

type DbSourceStorage struct {
	dbClient *bolt.DB
}

const mediaSourceKeyPrefix = "source:"
const checksumKeyPrefix = "checksum:"

var mediaSourceBucket = []byte("source")
var mediaCheckSumBucket = []byte("checksum")

const dbSubPath = ".boltdb/source.db"

func getBucket(bucketname []byte, tx *bolt.Tx) (*bolt.Bucket, error) {
	bucket := tx.Bucket(bucketname)
	if bucket != nil {
		return bucket, nil
	}

	bucket, err := tx.CreateBucket(bucketname)
	if err != nil {
		return nil, err
	}
	return bucket, err
}

// NewDbStorage create new storage for data
func NewSourceDbStorage(sourcePath string) (*DbSourceStorage, error) {
	dbPath := filepath.Join(sourcePath, dbSubPath)

	dbClient, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}

	s := DbSourceStorage{
		dbClient: dbClient,
	}
	// create bucket if not exists
	err = s.dbClient.Update(func(txn *bolt.Tx) error {
		_, err := getBucket(mediaSourceBucket, txn)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &s, err
}

// CloseDb closes link to db
func (s *DbSourceStorage) CloseDb() error {
	err := s.dbClient.Close()
	return err
}

func (s *DbSourceStorage) AddFile(filePath string) (string, error) {
	key := mediaSourceKeyPrefix + filePath
	err := s.dbClient.Update(func(txn *bolt.Tx) error {
		bucket, err := getBucket(mediaSourceBucket, txn)
		if err != nil {
			log.Printf("%v", err)
			return err
		}

		// convert to storage model
		sMedia := &DbSourceMedia{
			Key:  key,
			Path: filePath,
		}

		d, errint := sMedia.marshalMedia()
		if errint != nil {
			return errint
		}
		bkey := []byte(key)
		errint = bucket.Put(bkey, d)
		return errint
	})
	if err != nil {
		log.Printf("%v", err)
		return "", err
	}
	return key, nil
}

func (s *DbSourceStorage) HasFile(filePath string) (bool, error) {
	hasFile := false
	err := s.dbClient.View(func(txn *bolt.Tx) error {
		bucket, err := getBucket(mediaSourceBucket, txn)
		if err != nil {
			return err
		}
		key := mediaSourceKeyPrefix + filePath
		item := bucket.Get([]byte(key))
		if item != nil {
			hasFile = true
		}
		return err
	})
	return hasFile, err
}
func (s *DbSourceStorage) GetFilesByMimetypeFilter(filter []string) ([]*imports.SourceMedia, error) {
	var me []*imports.SourceMedia
	err := s.dbClient.View(func(txn *bolt.Tx) error {
		bucket, err := getBucket(mediaSourceBucket, txn)
		if err != nil {
			return err
		}

		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() { //sm := imports.SourceMedia{}
			dbsm := DbSourceMedia{}
			errint := dbsm.unmarshalMedia(v)
			if errint != nil {
				return errint
			}
			for i := range filter {
				if filter[i] == dbsm.Mimetype {
					sm := &imports.SourceMedia{
						Key:      dbsm.Key,
						Path:     dbsm.Path,
						Mimetype: dbsm.Mimetype,
						Checksum: dbsm.Checksum,
					}
					me = append(me, sm)
					break
				}
			}
		}
		return err
	})
	return me, err
}

func (s *DbSourceStorage) GetAllFiles() ([]*imports.SourceMedia, error) {
	var me []*imports.SourceMedia
	err := s.dbClient.View(func(txn *bolt.Tx) error {
		bucket, err := getBucket(mediaSourceBucket, txn)
		if err != nil {
			return err
		}

		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() { //sm := imports.SourceMedia{}
			dbsm := DbSourceMedia{}
			errint := dbsm.unmarshalMedia(v)
			if errint != nil {
				return errint
			}
			sm := &imports.SourceMedia{
				Key:          dbsm.Key,
				Path:         dbsm.Path,
				Mimetype:     dbsm.Mimetype,
				Checksum:     dbsm.Checksum,
				CreationDate: dbsm.CreationDate,
			}

			me = append(me, sm)
		}
		return nil
	})
	return me, err
}

func (s *DbSourceStorage) AddChecksum(media *imports.SourceMedia) error {
	if media.Checksum == "" {
		return nil
	}
	err := s.dbClient.Update(func(txn *bolt.Tx) error {
		bucket, err := getBucket(mediaCheckSumBucket, txn)
		if err != nil {
			log.Printf("%v", err)
			return err
		}
		var errint error
		key := checksumKeyPrefix + media.Checksum
		item := bucket.Get([]byte(key))
		if item == nil {
			checksum := &DbSourceChecksum{
				Key:     key,
				Sources: []string{media.Path},
			}
			item, errint = checksum.marshalChecksum()
		} else {
			checksum := DbSourceChecksum{}
			errint = checksum.unmarshalChecksum(item)
			if errint != nil {
				return errint
			}
			checksum.Sources = append(checksum.Sources, media.Path)
			item, errint = checksum.marshalChecksum()
		}
		if errint != nil {
			return errint
		}
		bkey := []byte(key)
		errint = bucket.Put(bkey, item)
		return errint
	})
	return err
}

func (s *DbSourceStorage) SaveMedia(media *imports.SourceMedia) (string, error) {
	key := media.Key
	err := s.dbClient.Update(func(txn *bolt.Tx) error {
		bucket, err := getBucket(mediaSourceBucket, txn)
		if err != nil {
			log.Printf("%v", err)
			return err
		}

		// convert to storage model
		sMedia := &DbSourceMedia{
			Key:          key,
			Path:         media.Path,
			Mimetype:     media.Mimetype,
			Checksum:     media.Checksum,
			CreationDate: media.CreationDate,
		}

		d, errint := sMedia.marshalMedia()
		if errint != nil {
			return errint
		}
		bkey := []byte(key)
		errint = bucket.Put(bkey, d)
		return errint
	})
	if err != nil {
		log.Printf("%v", err)
		return "", err
	}
	return key, nil
}

func (s *DbSourceStorage) GetFileByKey(path string) (*imports.SourceMedia, error) {
	var media *imports.SourceMedia
	err := s.dbClient.View(func(txn *bolt.Tx) error {
		bucket, err := getBucket(mediaSourceBucket, txn)
		if err != nil {
			return err
		}
		key := mediaSourceKeyPrefix + path
		item := bucket.Get([]byte(key))
		if item != nil {
			dbsm := DbSourceMedia{}
			errint := dbsm.unmarshalMedia(item)
			if errint != nil {
				return errint
			}
			media = &imports.SourceMedia{
				Key:          dbsm.Key,
				Path:         dbsm.Path,
				Mimetype:     dbsm.Mimetype,
				Checksum:     dbsm.Checksum,
				CreationDate: dbsm.CreationDate,
			}

		}
		return err
	})
	return media, err
}

func (s *DbSourceStorage) GetAllCheckSum() (error, []*imports.SourceChecksum) {
	var cs []*imports.SourceChecksum
	err := s.dbClient.View(func(txn *bolt.Tx) error {
		bucket, err := getBucket(mediaCheckSumBucket, txn)
		if err != nil {
			return err
		}

		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() { //sm := imports.SourceMedia{}
			dbsm := DbSourceChecksum{}
			errint := dbsm.unmarshalChecksum(v)
			if errint != nil {
				return errint
			}
			s := &imports.SourceChecksum{
				Key:     dbsm.Key,
				Sources: dbsm.Sources,
			}
			cs = append(cs, s)
		}
		return nil
	})
	return err, cs
}

func (m *DbSourceMedia) marshalMedia() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err := enc.Encode(m)
	return b.Bytes(), err
}

func (m *DbSourceMedia) unmarshalMedia(d []byte) error {
	b := bytes.NewBuffer(d)
	dec := gob.NewDecoder(b)
	err := dec.Decode(m)
	return err
}

func (m *DbSourceChecksum) marshalChecksum() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err := enc.Encode(m)
	return b.Bytes(), err
}

func (m *DbSourceChecksum) unmarshalChecksum(d []byte) error {
	b := bytes.NewBuffer(d)
	dec := gob.NewDecoder(b)
	err := dec.Decode(m)
	return err
}
