package lru

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	KeySizeOffset   = 0
	ValueSizeOffset = 4
	KeyOffset       = 8
)

type MMapHandler struct {
	file *os.File
	data []byte
	size int
}

func NewMMapHandler(filename string, size int) (*MMapHandler, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	if err = f.Truncate(int64(size)); err != nil {
		_ = f.Close()
		return nil, err
	}
	data, err := unix.Mmap(int(f.Fd()), 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	binary.LittleEndian.PutUint64(data[:8], 8)
	return &MMapHandler{file: f, data: data, size: size}, nil
}

func (h *MMapHandler) Close() error {
	if err := syscall.Munmap(h.data); err != nil {
		return err
	}
	return h.file.Close()
}

func (h *MMapHandler) Sync() error {
	return unix.Msync(h.data, syscall.MS_SYNC)
}

func (h *MMapHandler) Data() []byte {
	return h.data
}

type LRUCache struct {
	handler *MMapHandler
	maxSize int
}

func NewLRUCache(filename string, size int) (*LRUCache, error) {
	fmt.Println("Initializing new LRU Cache...")
	handler, err := NewMMapHandler(filename, size)
	if err != nil {
		return nil, err
	}
	return &LRUCache{handler: handler, maxSize: size}, nil
}

func (cache *LRUCache) Close() error {
	return cache.handler.Close()
}

func (cache *LRUCache) Get(key string) (string, bool) {
	fmt.Println("Attempting to retrieve key:", key)
	data := cache.handler.Data()
	idx, found := cache.binarySearch(data, key)
	if !found {
		fmt.Println("Key not found:", key)
		return "", false
	}
	value := cache.readValue(data, idx)
	fmt.Println("Retrieved value for key:", key, "value:", value)
	return value, true
}

func (cache *LRUCache) Set(key string, value string) error {
	fmt.Println("Setting value for key:", key, "value:", value)
	data := cache.handler.Data()
	idx, found := cache.binarySearch(data, key)
	if found {
		fmt.Println("Key found. Updating entry...")
		return cache.updateEntry(data, idx, key, value)
	} else {
		fmt.Println("Key not found. Inserting new entry...")
		return cache.insertEntry(data, key, value)
	}
}

func (cache *LRUCache) Delete(key string) error {
	fmt.Println("Deleting key:", key)
	data := cache.handler.Data()
	idx, found := cache.binarySearch(data, key)
	if !found {
		fmt.Println("Key not found during delete:", key)
		return errors.New("key not found")
	}
	cache.removeEntry(data, idx)
	fmt.Println("Key deleted:", key)
	return nil
}

func (cache *LRUCache) binarySearch(data []byte, key string) (int, bool) {
	// Assume the first 8 bytes are for storing the data size
	end := binary.LittleEndian.Uint64(data[:8])
	low, high := uint64(8), end

	for low < high {
		mid := low + (high-low)/2
		mid -= mid % KeyOffset // Align to entry boundary
		if midKey, valid := cache.readKey(data, int(mid)); valid {
			if midKey == key {
				return int(mid), true
			} else if midKey > key {
				high = mid
			} else {
				low = mid + KeyOffset
			}
		} else {
			return 0, false
		}
	}
	return 0, false
}

func (cache *LRUCache) readKey(data []byte, idx int) (string, bool) {
	if idx+KeyOffset > len(data) {
		fmt.Println("Index out of bounds while reading key at:", idx)
		return "", false
	}
	keySize := binary.BigEndian.Uint32(data[idx+KeySizeOffset : idx+ValueSizeOffset])
	if keySize == 0 || idx+KeyOffset+int(keySize) > len(data) {
		fmt.Println("Key size is zero or out of bounds for data at index:", idx)
		return "", false
	}
	key := string(data[idx+KeyOffset : idx+KeyOffset+int(keySize)])
	if key == "" {
		fmt.Println("Empty key read at index:", idx)
		return "", false
	}
	fmt.Println("Key read:", key)
	return key, true
}

func (cache *LRUCache) readValue(data []byte, idx int) string {
	fmt.Println("Reading value at index:", idx)
	keySize := binary.BigEndian.Uint32(data[idx+KeySizeOffset : idx+ValueSizeOffset])
	valueSize := binary.BigEndian.Uint32(data[idx+ValueSizeOffset : idx+KeyOffset])
	valueStart := idx + KeyOffset + int(keySize)
	if valueStart+int(valueSize) > len(data) {
		return ""
	}
	value := string(data[valueStart : valueStart+int(valueSize)])
	fmt.Println("Value read:", value)
	return value
}

func encodeEntry(key, value string) []byte {
	keyLen := uint32(len(key))
	valueLen := uint32(len(value))
	buf := make([]byte, 8+keyLen+valueLen) // 4 bytes for key length, 4 for value length

	binary.BigEndian.PutUint32(buf[0:4], keyLen)
	binary.BigEndian.PutUint32(buf[4:8], valueLen)
	copy(buf[8:], key)
	copy(buf[8+keyLen:], value)

	return buf
}

// updateEntry updates an existing entry in the data
func (cache *LRUCache) updateEntry(data []byte, idx int, key, value string) error {
	fmt.Println("Updating entry for key:", key)
	startIdx := idx
	keySize := int(binary.BigEndian.Uint32(data[idx+KeySizeOffset : idx+ValueSizeOffset]))
	valueSize := int(binary.BigEndian.Uint32(data[idx+ValueSizeOffset : idx+KeyOffset]))
	endIdx := idx + KeyOffset + keySize + valueSize

	newEntry := encodeEntry(key, value)
	if len(newEntry) > endIdx-startIdx {
		fmt.Println("Error: Updated entry size exceeds original size.")
		return errors.New("updated entry size exceeds original size, reinsert needed")
	}

	// Clear the old data
	for i := startIdx; i < endIdx; i++ {
		data[i] = 0
	}
	// Copy new entry into the data
	copy(data[startIdx:], newEntry)
	fmt.Println("Entry updated successfully.")
	return nil
}

// removeEntry removes an entry from the data
func (cache *LRUCache) removeEntry(data []byte, idx int) {
	fmt.Println("Removing entry at index:", idx)
	keySize := binary.BigEndian.Uint32(data[idx+KeySizeOffset : idx+ValueSizeOffset])
	valueSize := binary.BigEndian.Uint32(data[idx+ValueSizeOffset : idx+KeyOffset])
	entrySize := KeyOffset + int(keySize) + int(valueSize)

	// Adjust the following entries and data size
	copy(data[idx:], data[idx+entrySize:])
	newDataSize := cache.getCurrentDataSize() - uint64(entrySize)
	cache.setCurrentDataSize(newDataSize)
	fmt.Println("Entry removed successfully.")
}

// Helper function to get the current data size from mmap file
func (cache *LRUCache) getCurrentDataSize() uint64 {
	return binary.LittleEndian.Uint64(cache.handler.Data()[:8])
}

// Helper function to set the current data size in mmap file
func (cache *LRUCache) setCurrentDataSize(size uint64) {
	binary.LittleEndian.PutUint64(cache.handler.Data()[:8], size)
}

// Adjust insertEntry to manage the data size
func (cache *LRUCache) insertEntry(data []byte, key, value string) error {
	fmt.Println("Inserting new entry for key:", key)
	newEntry := encodeEntry(key, value)
	currentDataSize := cache.getCurrentDataSize()
	newEntrySize := uint64(len(newEntry))

	if currentDataSize+newEntrySize+8 > uint64(cache.maxSize) { // +8 for the data size field
		fmt.Println("Error: Cache is at maximum capacity.")
		return errors.New("cache is at maximum capacity")
	}

	// Append new entry
	copy(data[8+currentDataSize:], newEntry)
	cache.setCurrentDataSize(currentDataSize + newEntrySize)
	fmt.Println("New entry inserted successfully.")
	return nil
}
