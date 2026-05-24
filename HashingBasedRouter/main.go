package main

import (
	"crypto/md5"
	"encoding/binary"
	"hash/crc32"
	"log"
	"slices"
)

// MaxServer
const MAX_SERVERS = 2

// Use Bigger Server Number To See Different outputs

/*
WHAT THIS DOES:
It takes a string (like a username or IP address) and converts it into a 32-bit integer.

WHY CRC32 INSTEAD OF MD5/SHA256:
Unlike cryptographic hashes (MD5/SHA-256) which return bytes of length 16 and 32,
CRC32 directly returns a `uint32` number. This is extremely fast and perfect
for routing because we need a raw number to perform modulo arithmetic against
our server count.

EXAMPLE:
Input:  "user_123"
Output: 1667105369  (a fast, raw integer)

Input:  "user_456"
Output: 1522443002
*/
func crc32Hash(key string) int {
	//uint32 means 32 bit which is = 4 bytes and 2^32 = 4,294,967,296 possible values
	return int(crc32.ChecksumIEEE([]byte(key)))
}

func md5Hash(key string) int {
	// md5 returns 16 bytes long hash value
	// 16bytes * 8bits = 128 bits -> 2^128 possible values
	// 2^128 = 3.4e38 approx
	hashedBytes := md5.Sum([]byte(key))
	// log.Println("The Raw Bytes -> ", hashedBytes)
	//Convert it into an Integer
	// we need only first 4 bytes to get the 32 bit integer
	value := binary.BigEndian.Uint32(hashedBytes[:4])
	log.Println("The Integer -> ", value)
	return int(value)
}

// Approach One : Direct Hashing [This is a naive approach]
func directHashing(key string, hashFunc func(string) int) int {
	hashedNumber := hashFunc(key)
	// The Power of Modulo is that it  ensures output is in Range of 0->(N-1)
	// Where N is the number of servers
	// Possible outputs = 0 or 1 ( N = 2 )

	return hashedNumber % MAX_SERVERS

}

// Approach 2 : Consistent Hashing
var serverPositions []int

func getMaxValueBasedHash(key string) int {
	hashedValue := md5Hash(key)
	// 2^3=8 is the range of numbers [0-7]
	return hashedValue % 8

}

func AddServer(serverName string) {
	if len(serverPositions) == MAX_SERVERS {
		log.Println("Server positions are full")
		return
	}
	serverNamePosition := getMaxValueBasedHash(serverName)
	log.Println("The ServerName Position -> ", serverNamePosition)
	serverPositions = append(serverPositions, serverNamePosition)
	log.Println("The Server Positions -> ", serverPositions)
	// without sorting and max limit you can get duplicate server positions
	// The Server Positions ->  [5 2 6 4 5 2 6 4]
	// to make it like ring and clock wise we need to sort it
	slices.Sort(serverPositions)
	log.Println("The Sorted Server Positions -> ", serverPositions)
}

func GetServer(key string) int {
	if len(serverPositions) == 0 {
		log.Println("No servers are available")
		return -1
	}

	keyHash := getMaxValueBasedHash(key)

	// Linear search (normal loop) to find the first server position >= keyHash
	for i := 0; i < len(serverPositions); i++ {
		if serverPositions[i] >= keyHash {
			return serverPositions[i]
		}
	}

	// If no server position is >= keyHash, wrap around the ring to the first server
	return serverPositions[0]
}

func RemoveServer(serverName string) bool {
	if len(serverPositions) == 0 {
		log.Println("No servers are available")
		return false
	}

	keyHash := getMaxValueBasedHash(serverName)

	for start, end := 0, len(serverPositions)-1; start <= end; {
		mid := (start + end) / 2
		if keyHash == serverPositions[mid] {
			serverPositions = append(serverPositions[:mid], serverPositions[mid+1:]...)
			log.Println("The Updated Server Positions -> ", serverPositions)
			return true

		}

		if keyHash < serverPositions[mid] {
			end = mid - 1 // Search left half
		} else {
			start = mid + 1 // Search right half
		}
	}

	return false
}

func main() {
	log.Println("--- Testing Consistent Hashing ---")

	// Add servers to our hash ring
	AddServer("ServerA")
	AddServer("ServerB")

	// Fire requests (keys) to see which server position they are routed to
	keys := []string{"Request10", "Request2", "Request3", "TestUser"}
	
	for _, key := range keys {
		serverPos := GetServer(key)
		log.Printf("Request Key '%s' mapped to server at position: %d", key, serverPos)
	}
}
