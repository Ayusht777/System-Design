# Concurrency in Seat Booking: A Visual Guide

In a highly concurrent environment like an airline booking system, multiple users might attempt to book the exact same seat at the exact same millisecond. The way we query the database significantly affects the outcome. 

Here is an in-depth look at the three different querying strategies implemented in our code.

---

## 1. Query Without Lock (`QUERYWITHOUTLOCK`)

**The Query:**
```sql
SELECT id, name, trip_id, user_id FROM seat 
WHERE trip_id = 1 AND user_id IS NULL 
ORDER BY id LIMIT 1 
```

**How it works:**
This query simply reads the first available seat. It places absolutely no restrictions or locks on the row it reads.

**The Result (Race Condition):**
If 10 users run this simultaneously, all 10 transactions will "see" the exact same seat (e.g., Seat 1) as available. They will all proceed to the `UPDATE` step. The database executes these updates one after another, and the last one to `UPDATE` will "win" the seat, silently overwriting the others. This leads to **double-booking**.

### Visualizing the Flow

| Time | User A (Tx 1) | User B (Tx 2) | Database State |
| :--- | :--- | :--- | :--- |
| **t=1** | `SELECT` (Sees Seat 1 is empty) | | Seat 1 is free |
| **t=2** | | `SELECT` (Sees Seat 1 is empty) | Seat 1 is free |
| **t=3** | `UPDATE` Seat 1 to User A | | Seat 1 belongs to User A |
| **t=4** | `COMMIT` | `UPDATE` Seat 1 to User B | Seat 1 belongs to User B |
| **t=5** | | `COMMIT` | 🚨 **Double Booked! User A's booking was overwritten by User B.** |

---

## 2. Query With Update Lock (`QUERYWITHUPDATELOCK`)

**The Query:**
```sql
SELECT id, name, trip_id, user_id FROM seat 
WHERE trip_id = 1 AND user_id IS NULL 
ORDER BY id LIMIT 1 
FOR UPDATE
```

**How it works:**
The `FOR UPDATE` clause tells PostgreSQL: *"I am about to update this row. Lock it exclusively for me until my transaction finishes."* 

**The Result (Bottleneck / Contention):**
While this strictly prevents double-booking, it creates a massive bottleneck. When User A locks Seat 1, User B's attempt to select an available seat will try to evaluate Seat 1, see the lock, and **block/wait** until User A finishes. If 100 users try to book at once, 99 of them are stuck waiting in line. Once User A commits, User B unblocks, but since Seat 1 is now taken, User B's query will likely return no rows, forcing them to retry the whole process.

### Visualizing the Flow

| Time | User A (Tx 1) | User B (Tx 2) | Database State |
| :--- | :--- | :--- | :--- |
| **t=1** | `SELECT FOR UPDATE` (Locks Seat 1) | | Seat 1 is locked by Tx 1 |
| **t=2** | `UPDATE` Seat 1 | `SELECT FOR UPDATE` (Sees lock on Seat 1, **WAITS**) | Seat 1 is locked by Tx 1 |
| **t=3** | `COMMIT` (Releases lock) | ... still waiting ... | Seat 1 belongs to User A |
| **t=4** | | (Unblocks) Re-evaluates Seat 1. It is no longer `NULL`. Returns 0 rows. | Seat 1 belongs to User A |

---

## 3. Query With Skip Locked (`QUERYWITHSKIPLOCKED`)

**The Query:**
```sql
SELECT id, name, trip_id, user_id FROM seat 
WHERE trip_id = 1 AND user_id IS NULL 
ORDER BY id LIMIT 1 
FOR UPDATE SKIP LOCKED
```

**How it works:**
This is the **golden standard** for highly concurrent systems. It tells PostgreSQL: *"Lock the row I'm selecting FOR UPDATE. However, if another transaction already has a lock on a row, don't wait for it. **Just skip that row entirely** and give me the next available one."*

**The Result (High Throughput):**
If User A locks Seat 1, User B's query instantly skips Seat 1 and locks Seat 2. User C locks Seat 3. All this happens concurrently. There is **zero waiting**, **zero blocking**, and **zero double-booking**. Every user gets a unique seat instantly.

### Visualizing the Flow

| Time | User A (Tx 1) | User B (Tx 2) | Database State |
| :--- | :--- | :--- | :--- |
| **t=1** | `SELECT ... SKIP LOCKED` (Locks Seat 1) | | Seat 1 is locked by Tx 1 |
| **t=2** | `UPDATE` Seat 1 | `SELECT ... SKIP LOCKED` (Skips Seat 1, Locks Seat 2) | Seat 1 locked, Seat 2 locked |
| **t=3** | `COMMIT` (Seat 1 confirmed) | `UPDATE` Seat 2 | Seat 1 -> User A, Seat 2 locked |
| **t=4** | | `COMMIT` (Seat 2 confirmed) | ✅ **Success! Seat 1 -> User A, Seat 2 -> User B. No waiting.** |
