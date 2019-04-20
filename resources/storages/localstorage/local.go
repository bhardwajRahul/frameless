package localstorage

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"github.com/adamluzsi/frameless"
	"github.com/adamluzsi/frameless/resources"
	"github.com/adamluzsi/frameless/resources/queries"
	"io/ioutil"
	"strconv"

	"github.com/adamluzsi/frameless/iterators"
	"github.com/adamluzsi/frameless/reflects"
	"github.com/boltdb/bolt"
)

func NewLocal(path string) (*Local, error) {
	db, err := bolt.Open(path, 0600, nil)

	return &Local{DB: db, CompressionLevel: gzip.DefaultCompression}, err
}

type Local struct {
	DB               *bolt.DB
	CompressionLevel int
}

func (storage *Local) Purge() error {
	return storage.DB.Update(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			return tx.DeleteBucket(name)
		})
	})
}

func (storage *Local) Save(entity interface{}) error {
	return storage.DB.Update(func(tx *bolt.Tx) error {

		if currentID, ok := queries.LookupID(entity); !ok || currentID != "" {
			return fmt.Errorf("entity already have an ID: %s", currentID)
		}

		bucketName := storage.BucketNameFor(entity)
		bucket, err := tx.CreateBucketIfNotExists(bucketName)

		if err != nil {
			return err
		}

		uIntID, err := bucket.NextSequence()

		if err != nil {
			return err
		}

		encodedID := strconv.FormatUint(uIntID, 10)

		if err = queries.SetID(entity, encodedID); err != nil {
			return err
		}

		value, err := storage.Serialize(entity)

		if err != nil {
			return err
		}

		return bucket.Put(storage.uintToBytes(uIntID), value)

	})
}

func (storage *Local) Update(ptr interface{}) error {
	encodedID, found := queries.LookupID(ptr)

	if !found || encodedID == "" {
		return fmt.Errorf("can't find ID in %s", reflects.FullyQualifiedName(ptr))
	}

	ID, err := storage.IDToBytes(encodedID)

	if err != nil {
		return err
	}

	value, err := storage.Serialize(ptr)

	if err != nil {
		return err
	}

	return storage.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := storage.BucketFor(tx, ptr)

		if err != nil {
			return err
		}

		return bucket.Put(ID, value)
	})
}

func (storage *Local) Delete(Entity interface{}) error {
	ID, found := queries.LookupID(Entity)

	if !found || ID == "" {
		return fmt.Errorf("can't find ID in %s", reflects.FullyQualifiedName(Entity))
	}

	return storage.DeleteByID(Entity, ID)
}

func (storage *Local) FindAll(Type interface{}) frameless.Iterator {
	r, w := iterators.NewPipe()

	go func() {
		defer w.Close()

		err := storage.DB.View(func(tx *bolt.Tx) error {

			bucket := tx.Bucket(storage.BucketNameFor(Type))

			if bucket == nil {
				return nil
			}

			return bucket.ForEach(func(IDbytes, encodedEntity []byte) error {
				entity := reflects.New(Type)

				if err := storage.Deserialize(encodedEntity, entity); err != nil {
					return err
				}

				return w.Encode(entity) // iterators.ErrClosed will cancel ForEach execution
			})

		})

		if err != nil {
			w.Error(err)
		}
	}()

	return r
}

func (storage *Local) FindByID(ID string, ptr interface{}) (bool, error) {
	var found = false

	key, err := storage.IDToBytes(ID)

	if err != nil {
		return false, nil
	}

	err = storage.DB.View(func(tx *bolt.Tx) error {
		bucket, err := storage.BucketFor(tx, ptr)

		if err != nil {
			return err
		}

		encodedValue := bucket.Get(key)
		found = encodedValue != nil

		if encodedValue == nil {
			return nil
		}

		return storage.Deserialize(encodedValue, ptr)
	})

	return found, err
}

func (storage *Local) DeleteByID(Type interface{}, ID string) error {

	ByteID, err := storage.IDToBytes(ID)

	if err != nil {
		return err
	}

	return storage.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := storage.BucketFor(tx, Type)

		if err != nil {
			return err
		}

		if v := bucket.Get(ByteID); v == nil {
			return fmt.Errorf("%s is not found", ByteID)
		}

		return bucket.Delete(ByteID)
	})

}



// Close the Local database and release the file lock
func (storage *Local) Close() error {
	return storage.DB.Close()
}

func (storage *Local) Exec(quc resources.Query) frameless.Iterator {
	switch quc := quc.(type) {
	case queries.Save:
		return iterators.NewError(storage.Save(quc.Entity))

	case queries.FindByID:
		entity := reflects.New(quc.Type)

		ok, err := storage.FindByID(quc.ID, entity)

		if err != nil {
			return iterators.NewError(err)
		}

		if !ok {
			return iterators.NewEmpty()
		}

		return iterators.NewSingleElement(entity)

	case queries.FindAll:
		return storage.FindAll(quc.Type)

	case queries.DeleteByID:
		return iterators.NewError(storage.DeleteByID(quc.Type, quc.ID))

	case queries.DeleteEntity:
		return iterators.NewError(storage.Delete(quc.Entity))

	case queries.UpdateEntity:
		return iterators.NewError(storage.Update(quc.Entity))

	case queries.Purge:
		return iterators.NewError(storage.Purge())

	default:
		return iterators.NewError(frameless.ErrNotImplemented)

	}
}

func (storage *Local) BucketNameFor(e frameless.Entity) []byte {
	return []byte(reflects.FullyQualifiedName(e))
}

func (storage *Local) BucketFor(tx *bolt.Tx, e frameless.Entity) (*bolt.Bucket, error) {
	bucket := tx.Bucket(storage.BucketNameFor(e))

	var err error

	if bucket == nil {
		err = fmt.Errorf("No entity created before with type %s", reflects.FullyQualifiedName(e))
	}

	return bucket, err
}

func (storage *Local) IDToBytes(ID string) ([]byte, error) {
	n, err := strconv.ParseUint(ID, 10, 64)

	if err != nil {
		return nil, fmt.Errorf("ID is not acceptable for this storage: %s", ID)
	}

	return storage.uintToBytes(n), nil
}

// uintToBytes returns an 8-byte big endian representation of v.
func (storage *Local) uintToBytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func (storage *Local) Serialize(e frameless.Entity) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(e); err != nil {
		return nil, err
	}
	return storage.compress(buf.Bytes())
}

func (storage *Local) Deserialize(CompressedAndSerialized []byte, ptr frameless.Entity) error {
	serialized, err := storage.decompress(CompressedAndSerialized)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(serialized)
	dec := gob.NewDecoder(buf)
	return dec.Decode(ptr)
}

func (storage *Local) compress(serialized []byte) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	writer, err := gzip.NewWriterLevel(buffer, storage.CompressionLevel)
	if err != nil {
		return nil, err
	}
	_, err = writer.Write(serialized)
	writer.Flush()
	writer.Close()
	return buffer.Bytes(), err
}

func (storage *Local) decompress(compressedData []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return ioutil.ReadAll(reader)
}
