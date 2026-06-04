# Word Dictionary Service

## Requirements

We are designing a high-performance English Word Dictionary service that takes a word and returns its meaning. **We are currently building a prototype** to run locally and understand the mechanics of building custom storage engines without traditional databases.

### Core Requirements
- Handle **5 million requests per minute**.
- Manage a dataset of **1TB** containing **171,476 words**.
- Support **weekly updates** via a changelog (max 1,000 words).
- Maintain **high availability** and data **durability**.
- Ensure the system is **cost-effective** and easily **scalable**.
- Guarantee **thread safety**, no deadlocks, and minimal locking impact on throughput.

### Constraints
- **No Traditional Databases:** You cannot use MySQL, PostgreSQL, MongoDB, or similar databases.
- **Custom Storage Engine:** You must design a creative storage and retrieval system from scratch (our prototype uses an indexed CSV engine in Go).
- **Strict Data Consistency:** Data must never enter an inconsistent state during reads, writes, or changelog syncs.

## Sample Data

> **Note:** Data sorting is intentionally not considered in the current implementation. Sorting the data introduces additional overhead, and our custom index-based offset lookup guarantees O(1) reads without the overhead of maintaining sorted order.

### 1. Base CSV (`data.csv`)
This is the raw data file containing the words and meanings before any indexing.

| Word | Meaning |
|---|---|
| apple | a fruit |
| car | a vehicle |

### 2. Changelog (`changelog.log`)
Updates are received in a changelog file. This file contains new meanings for existing words or entirely new words [In our current implementation we will going to ignore new words].

> **Note:** The changelog can contain duplicate keys (e.g., multiple updates to the same word). During the sync process, the last occurrence of the word in the changelog takes precedence.

| Word | Meaning |
|---|---|
| apple | a red fruit |
| banana | a yellow fruit |
| apple | a sweet red fruit |

### 3. File After Sync
After the syncing process runs, the system processes the changelog and updates the base CSV data.

| Word | Meaning |
|---|---|
| apple | a sweet red fruit |
| car | a vehicle |
| banana | a yellow fruit |

### 4. Custom Storage Layout

To achieve O(1) lookups without a database, the system rewrites the base CSV into a custom binary/CSV hybrid format. 

**File Layout:**
```text
+-------------------+
| Reserved Header   | 256 bytes (Holds metadata like Index Start Position)
+-------------------+
| Data Block        | Variable size (The actual word and meaning pairs)
+-------------------+
| Index Block       | Variable size (Mapping of word -> byte offset)
+-------------------+
```

#### Why we chose `[Header] -> [Data] -> [Index]`
When designing a custom storage format, one might consider putting the index before the data (`Header -> Index -> Data`). However, our layout provides significant performance and architectural benefits:

- **Single-Pass Sequential Writing:** By appending the index at the end, we can process the 1TB of data sequentially in a single pass. We reserve a fixed 256 bytes for the header, stream and write the data block continuously without needing to know the index size upfront.
- **The Issue with `[Header] -> [Index] -> [Data]`:** If we placed the index first, we wouldn't know how many bytes the index requires until we have processed all 171,476 words. This would force us to either: (1) write the index twice, or (2) perform a costly pre-processing pass to calculate index size.
- **Fast O(1) Reads:** At runtime, the system only needs to read the 256-byte header to find the Index Block's starting position. It can instantly load the index into memory, giving us O(1) lookups without scanning the file.

## My Learnings

Here are some of the key takeaways and design choices made during the implementation, ordered progressively as the data flows through the system.

**Q: Why use `bytes.IndexByte(line, ',')` to extract the keyword instead of `strings.Split`?**
**A:** `bytes.IndexByte` efficiently finds the first comma directly on the byte array. Since we only need the keyword for the index, slicing bytes until the comma is significantly faster and uses less memory than splitting the entire line.

**Q: Why use `strings.TrimSpace` when reading lines from the file?**
**A:** Because functions like `ReadBytes('\n')` leave the newline character (`\n`) and sometimes carriage returns (`\r`) attached to the end of the string. `strings.TrimSpace` cleanly strips all trailing whitespace without allocating unnecessary buffers.

**Q: Why check for EOF at the end of the read loop rather than at the beginning?**
**A:** If a file doesn't end with a trailing newline, `ReadBytes('\n')` will return the last chunk of data along with an `io.EOF` error. If we checked for EOF at the top of the loop and broke immediately, we'd lose that final line of data.

**Q: Why open the changelog file with `os.O_WRONLY|os.O_APPEND`?**
**A:** When syncing changelogs, we only need to add new entries. `os.O_APPEND` guarantees that all writes are appended to the very end of the file without overwriting existing data, while `os.O_WRONLY` ensures we cannot accidentally read from it.

**Q: Why do we write data to a temporary file (`temp-data-*.csv`) instead of updating the base CSV directly?**
**A:** To ensure strict data consistency. If we modified the base file directly and the program crashed midway, the dictionary would be corrupted. Building the new file completely in a temporary location, syncing it, and only then renaming it ensures atomicity.

**Q: Why is `tmpFile.Sync()` necessary if `tmpFile.Write()` already succeeds?**
**A:** A successful `Write()` only means the data has been handed off to the operating system's memory cache. If the machine loses power, that data is lost. `Sync()` forces the OS to flush all buffered data to persistent storage on disk.

**Q: Why use the `Write Data -> Write Index -> Sync -> Close -> Rename` pattern?**
**A:** This is a classic storage-engine pattern for safe file updates. It guarantees that we never replace our original, good data file with a partially written or un-flushed file. Only after `Sync()` and `Close()` completes do we atomically rename the temp file into place.

## Back-of-the-Envelope Calculations

To ensure our system scales effectively, let's look at the rough storage and RAM requirements based on the given constraints.

### 1. Storage Calculations

**Raw Data:**
- We are given that **171,476 words** take up **1TB** of storage.
- Average size per entry: `1 TB / 171,476 ≈ 5.83 MB` per word meaning. (This implies meanings are extremely detailed, possibly containing encyclopedic content, HTML, or large metadata).

**Visual Storage Breakdown:**
```
┌──────────────────────────────────────────────────────────────┐
│                       1TB TOTAL DISK SPACE                    │
├──────────────────────────────────────────────────────────────┤
│                                                                │
│  ████████████████████████████████████████████████ 1TB Data    │
│  █ 171,476 words × ~5.83 MB/word                            █ │
│                                                                │
│  ▓ 4.28 MB Index        ▓ 256 B Header                        │
│                                                                │
└──────────────────────────────────────────────────────────────┘

Overhead Visualization:
┌─────────────────────────────────────────┐
│ Data:     1,000,000 MB  [99.9995%]     │
│ Index:          4.28 MB [ 0.0004%]     │
│ Header:       0.000256 MB [ <0.0001%]  │
└─────────────────────────────────────────┘
```

**Custom Index Overhead (Detailed Breakdown):**
- In our custom file format, the index maps each word to a byte offset (e.g., `apple,256\n`).
- Average word length: ~10 bytes
- Byte offset length (up to 1TB): ~13 characters (e.g., `1000000000000`)
- Comma + Newline: 2 bytes

| Component | Bytes | Calculation |
|-----------|-------|-------------|
| Word | 10 | avg word length |
| Offset | 13 | ~13 digits for 1TB range |
| Delimiter | 1 | comma separator |
| Newline | 1 | line ending |
| **Per Entry Total** | **25** | sum of above |
| × Number of Words | ×171,476 | total words |
| **Total Index Size** | **4.28 MB** | 171,476 × 25 bytes |

**Changelog Storage Growth:**
```
Weekly Updates Timeline:

Week 1:  [████] 5.83 GB
Week 2:  [████████] 11.66 GB
Week 3:  [███████████████] 17.49 GB
Month:   [██████████████████████████] ~23.32 GB

1,000 words/week × 5.83 MB/word = 5.83 GB/week growth
```

### 2. RAM (Memory) Calculations

**Holding the Index in Memory:**

To achieve instantaneous O(1) reads, the entire index is loaded into RAM as a hash map (`map[string]int64`).

| Component | Bytes | Breakdown |
|-----------|-------|-----------|
| String key (avg 15 chars) | 15 | word length |
| Go string header | 16 | internal metadata |
| int64 value (byte offset) | 8 | file position |
| Hash map overhead (Go internals) | 40 | bucket/pointer/padding |
| **Per Entry Total** | **79** | sum of above |
| × Number of Words | ×171,476 | total index entries |
| **Total RAM for Index** | **13.5 MB** | 171,476 × 79 bytes |

**Memory Efficiency Visualization:**
```
If Index Were 1TB (like data):

❌ NAIVE APPROACH:
┌──────────────────────────────────────┐
│ Holding ALL data in memory: 1TB RAM  │ IMPOSSIBLE ✗
│ (Would require $100K+ in servers)    │
└──────────────────────────────────────┘

✅ OUR APPROACH (Offset-based):
┌──────────────────────────────────────┐
│ Index only: 13.5 MB                  │ ✓ Fits in L3 cache
│ Per-request data: 5.83 MB            │ ✓ Fast disk access
│ Total active memory: ~19.33 MB       │ ✓ Incredibly efficient
└──────────────────────────────────────┘

Efficiency Gain: 1,000,000 MB ÷ 13.5 MB ≈ 74,074x memory savings
```

**Request Processing Memory Profile:**

```
Each Lookup Request:

1. Index Lookup (in RAM):
   Read map[string]int64  → 13.5 MB loaded once ✓
   One hash lookup        → O(1) operation
   Get byte offset        → ~50 nanoseconds

2. Disk Read (sequential):
   Seek to offset         → Direct file position
   Read meaning bytes     → ~5.83 MB average
   Return to client       → Stream output

Memory Timeline for Request:
┌──────────────────────────────────────┐
│ t=0ms: Index lookup    │ 13.5 MB      │
│ t=0.05ms: Seek pos     │ 13.5 MB      │
│ t=1ms: Stream data     │ 13.5 MB + ~5.83 MB
│ t=50ms: Complete       │ 13.5 MB (index stays)
└──────────────────────────────────────┘
```

**Maximum Concurrent Request Memory:**

At **5 million requests per minute** (83,333 req/sec):

```
Assuming average request takes 50ms to complete:

Concurrent requests = 83,333 req/sec × 0.05 sec = ~4,167 requests
Memory per request = 5.83 MB
Index (persistent) = 13.5 MB

┌──────────────────────────────────────────┐
│ Max concurrent RAM = (4,167 × 5.83 MB)   │
│                    + 13.5 MB (index)     │
│                    ≈ 24.3 GB             │
│                                          │
│ Feasible on modern servers with         │
│ 128GB+ RAM (uses ~19% capacity) ✓       │
└──────────────────────────────────────────┘
```

## Current Code Limitations (Real Issues in `main.go`)

Based on the current prototype implementation, here are the critical, real-world issues in `main.go` that need to be addressed before scaling:

- **File Descriptor Exhaustion (The Biggest Bottleneck):** In `getKeyValueFromIndex`, the code calls `os.Open()` and closes it for *every single read request*. At 5 million requests per minute, opening and closing files repeatedly exhausts file descriptors and causes severe I/O contention. Solution: Keep the file descriptor open in memory or use memory-mapped I/O.
- **Double File Processing During Sync:** When `syncChangelogs` runs, it writes out the entire 1TB file with the updated rows. But right after, it calls `BuildBaseCsvIndex`, which reads and writes that same 1TB file again. This doubles I/O cost during syncs. Solution: Build the index while writing the data in a single pass.
- **Global State & Thread Safety Risks:** `indexMapper` is a package-level global map. While concurrent reads in Go are safe, if a background process ever triggers an index reload (e.g., after a sync completes), there's a race condition where some goroutines might read from an old or partially updated index. Solution: Use a sync.RWMutex to protect index reloads, or implement copy-on-write semantics.
