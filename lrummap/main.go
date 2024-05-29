package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"
)

func generateRandomData() []byte {
	rand.Seed(time.Now().UnixNano())
	var buffer bytes.Buffer

	keys := make([]string, 300)
	for i := 0; i < 100; i++ {
		keys[i*3] = fmt.Sprintf("a%d", i)
		keys[i*3+1] = fmt.Sprintf("bbbbbbbbbbbbbbbbbbbb%d", i)
		keys[i*3+2] = fmt.Sprintf("ccc%d", i)
	}

	sort.Strings(keys)

	for _, key := range keys {
		value := fmt.Sprintf("value%d", rand.Intn(100))
		keyLen := len(key)
		valLen := len(value)
		binary.Write(&buffer, binary.BigEndian, uint32(keyLen))
		buffer.WriteString(key)
		binary.Write(&buffer, binary.BigEndian, uint32(valLen))
		buffer.WriteString(value)
	}

	return buffer.Bytes()
}

func parseKeys(data []byte) [][8]byte {
	var positions [][8]byte
	var pos [8]byte
	i := 0
	for i < len(data) {
		keyLen := binary.BigEndian.Uint32(data[i : i+4])
		binary.BigEndian.PutUint32(pos[:], uint32(i+4))
		positions = append(positions, pos)
		valLen := binary.BigEndian.Uint32(data[uint32(i)+4+keyLen : uint32(i)+4+keyLen+4])
		i += 4 + int(keyLen) + 4 + int(valLen)
	}
	return positions
}

func binarySearch(data []byte, positions [][8]byte, target string) int {
	low, high := 0, len(positions)-1
	for low <= high {
		mid := (low + high) / 2
		midKeyPosition := int(binary.BigEndian.Uint32(positions[mid][:]))
		keyLen := int(binary.BigEndian.Uint32(data[midKeyPosition-4 : midKeyPosition]))
		midKey := string(data[midKeyPosition : midKeyPosition+keyLen])
		if midKey == target {
			return midKeyPosition
		} else if midKey < target {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return -1
}

func getValue(data []byte, keyPos int) string {
	keyLen := int(binary.BigEndian.Uint32(data[keyPos-4 : keyPos]))
	valLenPos := keyPos + keyLen
	valLen := int(binary.BigEndian.Uint32(data[valLenPos : valLenPos+4]))
	valStart := valLenPos + 4
	return string(data[valStart : valStart+valLen])
}

// del removes the specified key and its value from the data buffer and updates the positions.
func del(data *[]byte, positions *[][8]byte, target string) bool {
	// Use binarySearch to find the position of the key.
	index := binarySearch(*data, *positions, target)
	if index == -1 {
		return false // Key not found
	}

	keyPos := int(binary.BigEndian.Uint32((*positions)[index][:]))
	keyLen := int(binary.BigEndian.Uint32((*data)[keyPos-4 : keyPos]))
	valLenPos := keyPos + keyLen
	valLen := int(binary.BigEndian.Uint32((*data)[valLenPos : valLenPos+4]))
	totalLen := 4 + keyLen + 4 + valLen

	// Shift the remaining data to overwrite the deleted key-value pair.
	copy((*data)[keyPos-4:], (*data)[keyPos-4+totalLen:])

	// Resize the data slice.
	*data = (*data)[:len(*data)-totalLen]

	// Update the positions slice.
	for i := index; i < len(*positions)-1; i++ {
		if i < len(*positions)-1 {
			(*positions)[i] = (*positions)[i+1]
		}
		pos := int(binary.BigEndian.Uint32((*positions)[i][:]))
		binary.BigEndian.PutUint32((*positions)[i][:], uint32(pos-totalLen))
	}
	*positions = (*positions)[:len(*positions)-1]

	return true
}

// set adds or updates a key-value pair in the data buffer and updates the positions.
func set(data *[]byte, positions *[][8]byte, key, value string) {
	index := binarySearch(*data, *positions, key)
	if index != -1 {
		// Key exists, delete it first
		del(data, positions, key)
	}

	// Prepare the new key-value pair
	var buffer bytes.Buffer
	keyLen := len(key)
	valLen := len(value)
	binary.Write(&buffer, binary.BigEndian, uint32(keyLen))
	buffer.WriteString(key)
	binary.Write(&buffer, binary.BigEndian, uint32(valLen))
	buffer.WriteString(value)

	// Append to the data buffer
	*data = append(*data, buffer.Bytes()...)

	// Update the positions slice
	var pos [8]byte
	binary.BigEndian.PutUint32(pos[:], uint32(len(*data)-buffer.Len()))
	*positions = append(*positions, pos)
}

// list prints all keys stored in the data buffer.
func list(positions [][8]byte, data []byte) {
	fmt.Println("Listing all keys:")
	for _, pos := range positions {
		keyPos := int(binary.BigEndian.Uint32(pos[:]))
		keyLen := int(binary.BigEndian.Uint32(data[keyPos-4 : keyPos]))
		key := string(data[keyPos : keyPos+keyLen])
		fmt.Println(key)
	}
}

func main() {
	data := generateRandomData()
	positions := parseKeys(data)

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Available commands: set <key> <value>, get <key>, del <key>, list, exit")

	for scanner.Scan() {
		input := scanner.Text()
		parts := strings.SplitN(input, " ", 3)

		switch parts[0] {
		case "exit":
			break
		case "set":
			if len(parts) != 3 {
				fmt.Println("Usage: set <key> <value>")
				continue
			}
			set(&data, &positions, parts[1], parts[2])
			fmt.Println("Set completed.")
		case "get":
			if len(parts) != 2 {
				fmt.Println("Usage: get <key>")
				continue
			}
			result := binarySearch(data, positions, parts[1])
			if result != -1 {
				value := getValue(data, result)
				fmt.Printf("Key '%s' found with value '%s'\n", parts[1], value)
			} else {
				fmt.Println("Key not found")
			}
		case "del":
			if len(parts) != 2 {
				fmt.Println("Usage: del <key>")
				continue
			}
			if del(&data, &positions, parts[1]) {
				fmt.Println("Deletion successful.")
			} else {
				fmt.Println("Key not found.")
			}
		case "list":
			list(positions, data)
		default:
			fmt.Println("Unknown command. Try: set, get, del, list, or exit.")
		}
		fmt.Println("Enter a command (type 'exit' to quit):")
	}
}
