from fastapi import FastAPI ,WebSocket
from redis.asyncio import Redis
import uvicorn

redis_client = Redis(
    host='redis-10008.c17.us-east-1-4.ec2.cloud.redislabs.com',
    port=10008,
    decode_responses=True,
    username="default",
    password="569PS6T1nk7DFLckFattxyxA899OLxXo",
)

app = FastAPI()

@app.websocket('/ws')
async def websocket_endpoint(ws:WebSocket):
    await ws.accept()
    try:
        while True:
            data = await ws.receive_text()
            await redis_client.publish('chat_channel', data)
    except Exception as e:
        print("Client disconnected", e)



if __name__ == "__main__":
    uvicorn.run("ws1:app", host="0.0.0.0", port=8000, reload=True)

