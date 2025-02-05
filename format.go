package caskdb

import (
	"encoding/binary"
	"fmt"
)

// format file provides encode/decode functions for serialisation and deserialisation
// operations
//
// format methods are generic and does not have any disk or memory specific code.
//
// The disk storage deals with bytes; you cannot just store a string or object without
// converting it to bytes. The programming languages provide abstractions where you
// don't have to think about all this when storing things in memory (i.e. RAM).
// Consider the following example where you are storing stuff in a hash table:
//
//    books = {}
//    books["hamlet"] = "shakespeare"
//    books["anna karenina"] = "tolstoy"
//
// In the above, the language deals with all the complexities:
//
//    - allocating space on the RAM so that it can store data of `books`
//    - whenever you add data to `books`, convert that to bytes and keep it in the memory
//    - whenever the size of `books` increases, move that to somewhere in the RAM so that
//      we can add new items
//
// Unfortunately, when it comes to disks, we have to do all this by ourselves, write
// code which can allocate space, convert objects to/from bytes and many other operations.
//
// This file has two functions which help us with serialisation of data.
//
//    encodeKV - takes the key value pair and encodes them into bytes
//    decodeKV - takes a bunch of bytes and decodes them into key value pairs
//
//**workshop note**
//
//For the workshop, the functions will have the following signature:
//
//    func encodeKV(timestamp uint32, key string, value string) (int, []byte)
//    func decodeKV(data []byte) (uint32, string, string)

// headerSize specifies the total header size. Our key value pair, when stored on disk
// looks like this:
//
//	┌───────────┬──────────┬────────────┬─────┬───────┐
//	│ timestamp │ key_size │ value_size │ key │ value │
//	└───────────┴──────────┴────────────┴─────┴───────┘
//
// This is analogous to a typical database's row (or a record). The total length of
// the row is variable, depending on the contents of the key and value.
//
// The first three fields form the header:
//
//	┌───────────────┬──────────────┬────────────────┐
//	│ timestamp(4B) │ key_size(4B) │ value_size(4B) │
//	└───────────────┴──────────────┴────────────────┘
//
// These three fields store unsigned integers of size 4 bytes, giving our header a
// fixed length of 12 bytes. Timestamp field stores the time the record we
// inserted in unix epoch seconds. Key size and value size fields store the length of
// bytes occupied by the key and value. The maximum integer
// stored by 4 bytes is 4,294,967,295 (2 ** 32 - 1), roughly ~4.2GB. So, the size of
// each key or value cannot exceed this. Theoretically, a single row can be as large
// as ~8.4GB.
const headerSize = 12

// KeyEntry keeps the metadata about the KV, specially the position of
// the byte offset in the file. Whenever we insert/update a key, we create a new
// KeyEntry object and insert that into keyDir.
type KeyEntry struct {
}

func NewKeyEntry(timestamp uint32, position uint32, totalSize uint32) KeyEntry {
	panic("implement me")
}

func encodeHeader(timestamp uint32, keySize uint32, valueSize uint32) []byte {
	final_buff := make([]byte, headerSize)
	binary.BigEndian.PutUint32(final_buff, timestamp)
	// via https://stackoverflow.com/a/29062148/12096319
	binary.BigEndian.PutUint32(final_buff[4:], keySize)

	binary.BigEndian.PutUint32(final_buff[8:], valueSize)
	// write to different part of buffer by https://stackoverflow.com/a/41833364/12096319
	fmt.Println(final_buff)
	return final_buff
}

func decodeHeader(header []byte) (uint32, uint32, uint32) {
	timestamp_int := (binary.BigEndian.Uint32(header[:4]))
	keySize_int := (binary.BigEndian.Uint32(header[4:8]))
	valueSize_int := (binary.BigEndian.Uint32(header[8:12]))
	// from https://stackoverflow.com/a/21851632/12096319

	return timestamp_int, keySize_int, valueSize_int

}

func encodeKV(timestamp uint32, key string, value string) (int, []byte) {
	keyBytes := []byte(key)
	valueBytes := []byte(value)
	var keyByteslen = uint32(len(keyBytes))
	var valueByteslen = uint32(len(valueBytes))
	headerBytes := encodeHeader(timestamp, keyByteslen, valueByteslen)
	headerBytes = append(headerBytes, keyBytes...)
	headerBytes = append(headerBytes, valueBytes...)
	return len(headerBytes), headerBytes

}

func decodeKV(data []byte) (uint32, string, string) {
	headerBytes := data[:headerSize]
	timestamp, keySize, valueSize := decodeHeader(headerBytes)
	keyEnd := headerSize + keySize
	valueEnd := keyEnd + valueSize
	keyString := string(data[headerSize:keyEnd])
	valueString := string(data[keyEnd:valueEnd])
	return timestamp, keyString, valueString
	// from https://stackoverflow.com/a/40673073/12096319

}
