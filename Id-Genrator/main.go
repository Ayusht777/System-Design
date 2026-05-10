package main

import (
	"fmt"
	"sync"
	"time"
)

/*
WHY WE USE "|" (bitwise OR) IN SNOWFLAKE INSTEAD OF "+"

Snowflake ID structure:

[timestamp][machineID][sequence]

We shift bits so every field occupies separate positions.

Example:

timestamp = 5
machine   = 3
sequence  = 2

Binary:

timestamp = 101
machine   = 11
sequence  = 10

--------------------------------------------------
STEP 1: SHIFT INTO THEIR POSITIONS
--------------------------------------------------

Suppose:
- machine uses 2 bits
- sequence uses 3 bits

Then:

timestamp << 5 //left shifts just add zeros at the end

101 -> 10100000

machine << 3

11 -> 00011000

sequence stays at end

10 -> 00000010

--------------------------------------------------
STEP 2: COMBINE USING "|"
--------------------------------------------------

10100000
00011000
00000010
---------
10111010

This safely merges all regions.

WHY "|" IS SAFE:
- no carry happens
- bits simply combine
- perfect for bit packing

--------------------------------------------------
WHY "+" CAN BE DANGEROUS
--------------------------------------------------

Binary addition can create carry.

Example:

1 + 1 = 10 (binary)

carry moved to next bit.

Another example:

0011
0001
----
0100

Notice:
carry changed neighboring bits.

In Snowflake this can corrupt fields
if bit regions overlap accidentally.

--------------------------------------------------
WHY "|" IS PREFERRED
--------------------------------------------------

Snowflake is NOT arithmetic.

We are:
- packing bits
- assembling regions

So "|" is semantically correct.

Industry standard:

id :=
    (timestamp << 22) |
    (machineID << 12) |
    sequence

NOT:

id :=
    (timestamp << 22) +
    (machineID << 12) +
    sequence

Even if "+" sometimes works,
"|" is safer and clearer.
*/

// why this is constant = because we have total 1024 machine
// 0-1023 machine id we can 2^10 is = 1024
const MACHINED_ID int64 = 1023

const MAX_COUNTER int64 = 4095 // 2^12
var (
	counter int64 = 0
	lock    sync.Mutex
)

// [timestamp][machine][sequence]
// timestamp = 42 bit
// machine = 10 bit
// sequence = 12 bit
func GenerateSnowflakeId() int64 {
	lock.Lock()
	defer lock.Unlock()

	// Unix Milliseconds: 13 digits (current era)
	// one number of 4 bit
	timeInMillisecond := time.Now().UnixMilli()
	fmt.Println("the  millisecond ", timeInMillisecond)
	//why 22 = because we have left  10 for machine id and 12 for sequence number
	fmt.Println("the left shift ", timeInMillisecond<<22)

	if counter < MAX_COUNTER {

		counter = counter + 1
	} else {
		counter = 0

	}

	/*
		WHY timestamp IS SHIFTED BY 22 AND NOT 41

		Snowflake layout:

		[timestamp][machine][sequence]

		Bit sizes:

		timestamp = 41 bits
		machine   = 10 bits
		sequence  = 12 bits

		--------------------------------------------------
		IMPORTANT RULE
		--------------------------------------------------

		Shift amount means:

		"How much empty space to reserve on RIGHT side"

		NOT:

		"How many bits the value itself has"

		--------------------------------------------------
		WHY timestamp << 22
		--------------------------------------------------

		timestamp sits above:

		[machine][sequence]

		So timestamp must leave space for BOTH.

		machine bits  = 10
		sequence bits = 12

		Total space needed:

		10 + 12 = 22

		So:

		timestamp << 22

		This moves timestamp left
		and reserves 22 empty bits.

		Example:

		101010

		after << 22:

		1010100000000000000000

		Now lower 22 bits are empty.

		Those bits are used for:

		[machine][sequence]

		--------------------------------------------------
		WHY NOT timestamp << 41
		--------------------------------------------------

		41 is the SIZE of timestamp itself.

		It is NOT the shift amount.

		If we do:

		timestamp << 41

		then we create 41 empty bits on right side,
		which is unnecessary.

		We only need space for:

		machine + sequence = 22 bits

		So correct shift is:

		timestamp << 22

		--------------------------------------------------
		SAME LOGIC FOR MACHINE
		--------------------------------------------------

		machine sits above sequence only.

		sequence uses 12 bits.

		So:

		machine << 12

		This reserves 12 bits for sequence.

		--------------------------------------------------
		FINAL FORMULA
		--------------------------------------------------

		id :=
		    (timestamp << 22) |
		    (machineID << 12) |
		    sequence

		Meaning:

		- timestamp in upper region
		- machine in middle region
		- sequence in lower region
	*/
	// id := timeInMillisecond<<22 + MACHINED_ID<<12 + counter
	id := (timeInMillisecond << 22) | (MACHINED_ID << 12) | (counter)
	return id

}

func main() {
	id := GenerateSnowflakeId()
	fmt.Println("the generated id is ", id)
}
