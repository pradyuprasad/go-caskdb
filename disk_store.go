package caskdb

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// DiskStore is a Log-Structured Hash Table as described in the BitCask paper. We
// keep appending the data to a file, like a log. DiskStorage maintains an in-memory
// hash table called KeyDir, which keeps the row's location on the disk.
//
// The idea is simple yet brilliant:
//   - Write the record to the disk
//   - Update the internal hash table to point to that byte offset
//   - Whenever we get a read request, check the internal hash table for the address,
//     fetch that and return
//
// KeyDir does not store values, only their locations.
//
// The above approach solves a lot of problems:
//   - Writes are insanely fast since you are just appending to the file
//   - Reads are insanely fast since you do only one disk seek. In B-Tree backed
//     storage, there could be 2-3 disk seeks
//
// However, there are drawbacks too:
//   - We need to maintain an in-memory hash table KeyDir. A database with a large
//     number of keys would require more RAM
//   - Since we need to build the KeyDir at initialisation, it will affect the startup
//     time too
//   - Deleted keys need to be purged from the file to reduce the file size
//
// Read the paper for more details: https://riak.com/assets/bitcask-intro.pdf
//
// DiskStore provides two simple operations to get and set key value pairs. Both key
// and value need to be of string type, and all the data is persisted to disk.
// During startup, DiskStorage loads all the existing KV pair metadata, and it will
// throw an error if the file is invalid or corrupt.
//
// Note that if the database file is large, the initialisation will take time
// accordingly. The initialisation is also a blocking operation; till it is completed,
// we cannot use the database.
//
// Typical usage example:
//
//		store, _ := NewDiskStore("books.db")
//	   	store.Set("othello", "shakespeare")
//	   	author := store.Get("othello")
type DiskStore struct {
	fileName string
	db_map   map[string]KeyEntry
	fileRef  *os.File
}

func NewDiskStore(fileName string) (*DiskStore, error) {
	var f *os.File
	var err error
	f, err = os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	diskstore := &DiskStore{fileName, map[string]KeyEntry{}, f}

	position := 0
	for {
		headerBytes := make([]byte, headerSize)
		n, err := diskstore.fileRef.ReadAt(headerBytes, int64(position))
		if err != nil {
			if errors.Is(err, io.EOF) && n == 0 {
				// we do n == 0 because that means that we have actually gotten zero bytes left (and its not just EOF for some random reason)
				break
			}
			return nil, err
		}

		_, keySize, valueSize := decodeHeader(headerBytes)
		totalSize := headerSize + keySize + valueSize

		recordBytes := make([]byte, totalSize)
		_, err = diskstore.fileRef.ReadAt(recordBytes, int64(position))
		if err != nil {
			return nil, err
		}

		timestamp, key, _ := decodeKV(recordBytes)
		diskstore.db_map[key] = NewKeyEntry(timestamp, uint32(position), totalSize)
		position += int(totalSize)
	}
	return diskstore, nil
}

func (d *DiskStore) Get(key string) string {
	entry, exists := d.db_map[key]
	if !exists {
		fmt.Println("no value for", key)
		return ""
	}

	bytes := make([]byte, entry.totalSize)
	d.fileRef.ReadAt(bytes, int64(entry.position))
	timestamp, newKey, value := decodeKV(bytes)
	if newKey != key {
		panic(errors.New("NewKey is not the same as the original key"))
	}

	if timestamp != entry.timestamp {
		panic(errors.New("timestamps don't match"))
	}

	return value

}

func (d *DiskStore) Set(key string, value string) {
	fileinfo, err := d.fileRef.Stat()
	if err != nil {
		panic(err)
	}

	position_start := uint32(fileinfo.Size())
	curr_time := uint32(time.Now().Unix())
	bytesCount, bytesToWrite := encodeKV(uint32(curr_time), key, value)
	numBytesWritten, err := d.fileRef.Write(bytesToWrite)
	if err != nil {
		panic(err)
	}

	if numBytesWritten != bytesCount {
		panic(errors.New("numBytesWritten is not the same as BytesCount"))
	}

	keyEntry := NewKeyEntry(uint32(curr_time), position_start, uint32(bytesCount))
	d.db_map[key] = keyEntry
	err = d.fileRef.Sync()
	if err != nil {
		panic(err)
	}

}

func (d *DiskStore) Close() bool {
	err := d.fileRef.Close()
	return err == nil
}
