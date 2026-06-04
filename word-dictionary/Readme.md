# Word Dictionary Service

## Requirements

We are designing a high-performance English Word Dictionary service that takes a word and returns its meaning. **We are currently building a prototype** to run locally and understand the mechanics of custom storage formats.

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

> **Note:** Data sorting is intentionally not considered in the current implementation. Sorting the data introduces additional overhead, and our custom index-based offset lookup guarantees O(1) read access without the need for the underlying data to be sorted.

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

- **Single-Pass Sequential Writing:** By appending the index at the end, we can process the 1TB of data sequentially in a single pass. We reserve a fixed 256 bytes for the header, stream and write the data directly to disk while tracking the byte offsets in memory, and then dump the index at the very end. Finally, we rewind to the 0th byte and write the exact offset where the index started into the header.
- **The Issue with `[Header] -> [Index] -> [Data]`:** If we placed the index first, we wouldn't know how many bytes the index requires until we have processed all 171,476 words. This would force us to either read the massive 1TB dataset twice (once to calculate the index size, once to write) or buffer all the data in temporary files. For a 1TB file, these approaches are extremely slow, memory-intensive, and I/O heavy.
- **Fast O(1) Reads:** At runtime, the system only needs to read the 256-byte header to find the Index Block's starting position. It can instantly load the index into memory, giving us O(1) lookups that jump straight to the correct byte offset in the Data Block.
