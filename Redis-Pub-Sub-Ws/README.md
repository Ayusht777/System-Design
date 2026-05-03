# Redis Pub/Sub WebSocket Scaling

## 🚀 What Problem Does This Solve?

This project demonstrates a low-level replication technique for scaling WebSockets using **Redis Pub/Sub**. 

*   **Massive Connection Scaling:** Horizontally scales WebSocket servers to handle an n number of concurrent connections across multiple server instances.
*   **Avoids Server Mesh:** Uses Redis as a centralized message broker, eliminating the need for complex server-to-server mesh communication.
*   **Real-Time Delivery:** Messages are delivered instantly with extremely low latency.
*   **Ephemeral by Default:** This implementation is purely real-time without persistence (if a client is offline, they miss the message).
*   **Extensible to Kafka:** If message persistence, history, or guaranteed delivery is required later, we can publish messaage to kafka as well.

---

## 🏃 How to Run

You can run both servers (Publisher and Subscriber) simultaneously. Hot-reloading is fully enabled for both methods.

| Method | Step 1 (Run WS1 on Port 8000) | Step 2 (Run WS2 on Port 8001) |
| :--- | :--- | :--- |
| **Via Terminal (`uv`)** | `uv run python ws1.py ` | `uv run python ws2.py` (in a new tab) |
| **Via VS Code Debugger** | Open Run & Debug (`Ctrl+Shift+D`) | Select **"Run Both Websockets"** and press Play (`F5`) |

### Step-by-Step Testing Guide:
1. **Start both servers** using your preferred method from the table above.
2. **Connect User 1**: Open a WebSocket client (like Postman or a browser console) and connect to `ws://localhost:8000/ws`.
3. **Connect User 2**: Open a second WebSocket client and connect to `ws://localhost:8001/ws`.
4. **Chat**: Send a message from User 1. It will go through `ws1`, be published to Redis, picked up by `ws2`, and delivered to User 2 instantly!
