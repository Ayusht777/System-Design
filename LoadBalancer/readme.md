══════════════════════════════════════════
   Load Balancer — Stress Tester
══════════════════════════════════════════

  Target URL    : http://localhost:8080/
  Total Requests: 10000
  Concurrency   : 1000

  [   1s]  req/sec: 4272   |  completed: 4272    |  avg req/sec: 4271.7
  [   2s]  req/sec: 3857   |  completed: 8129    |  avg req/sec: 4060.0

══════════════════════════════════════════
          LOAD TEST RESULTS
══════════════════════════════════════════

  Total Requests  :  10000
  Successful      :  10000
  Failed          :  0
  Total Duration  :  2.651s

  Throughput
    Requests/sec  :  3772.05
    Requests/min  :  226323

  Latency
    Min           :  4.198ms
    Avg           :  234.143ms
    P50 (median)  :  134.878ms
    P90           :  572.944ms
    P99           :  770.838ms
    Max           :  873.333ms

══════════════════════════════════════════