| Algorithm   | Output Size | Speed (Go, 1MB)     | Security                     | Routing Suitability       | Best For                                   |
| ----------- | ----------- | ------------------- | ---------------------------- | ------------------------- | ------------------------------------------ |
| **CRC32**   | 32 bits     | ~12 GB/s (fastest)  | None                         | ⚠️ Poor (collision-prone) | Error detection, not routing               |
| **MD5**     | 128 bits    | ~528 MB/s           | Broken (collisions possible) | ✅ Good                    | Fast routing where security doesn't matter |
| **SHA-1**   | 160 bits    | ~525 MB/s           | Broken                       | ✅ Good                    | Legacy compatibility                       |
| **SHA-256** | 256 bits    | ~239 MB/s           | Secure                       | ✅✅ Excellent              | Production routing (recommended)           |
| **SHA-512** | 512 bits    | ~285 MB/s           | Very Secure                  | ✅✅ Excellent              | 64-bit systems, max security               |

