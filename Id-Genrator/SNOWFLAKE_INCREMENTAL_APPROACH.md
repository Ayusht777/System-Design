# Snowflake ID Incremental Approach (simple notes)
This is a small step-by-step note for how this version was built.
Language style is kept simple like code comments.
## Step 1: basic version (timestamp only)
Start with time only.
Reason: time always moves forward, so IDs naturally keep increasing.
Example idea:
- `id = time.Now().UnixMilli()`
Limit:
- Same millisecond can generate duplicate IDs if many requests come together.
## Step 2: add machine part
Add machine id so multiple servers can generate IDs safely.
In this code:
- machine bits = 10
- machine range = `0-1023` (total `2^10 = 1024`)
Reason:
- if server A and server B generate at same millisecond, machine bits separate them.
## Step 3: add sequence/counter part
Add sequence for same machine + same millisecond case.
In this code:
- sequence bits = 12
- max sequence = `4095` (`2^12 - 1`)
Reason:
- one machine can create many IDs in one millisecond without collision.
## Step 4: bit layout and shifts
Layout:
- `[timestamp][machine][sequence]`
Bit sizes used here:
- timestamp ~ 41 bits currently (unix milli value today)
- machine = 10 bits
- sequence = 12 bits
So shifts are:
- `timestamp << 22` (reserve 10 + 12 right bits)
- `machine << 12` (reserve sequence bits)
## Step 5: combine with bitwise OR
Final compose:
- `(timestamp << 22) | (machine << 12) | sequence`
Why `|`:
- safe bit packing
- no carry behavior
- clearer intent than arithmetic `+`
## Step 6: add lock for concurrency safety
`counter` is shared state.
If many goroutines call generator at same time, race can happen.
So this code uses:
- `lock sync.Mutex`
- `lock.Lock()` / `defer lock.Unlock()`
Reason:
- counter update and ID build become one atomic critical section.
## Final result
Now this version supports:
- sortable IDs (time-first)
- multi-machine support
- many IDs per same millisecond
- safe concurrent generation in one process
