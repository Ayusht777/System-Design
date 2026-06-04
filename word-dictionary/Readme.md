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

