import asyncio
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
pubsub  = redis_client.pubsub()

async def redis_receive(ws):
        await pubsub.subscribe('chat_channel') # same channel will cause duplicate message, so diff chat_channel1 for ws1 and chat_channel2 for ws2


        try:
            async for message in pubsub.listen():
                if message['type'] == 'message':
                    await ws.send_text(message['data'])
        except Exception as e:
            print(f"Redis receive error: {e}")

@app.websocket('/ws')
async def websocket_endpoint(ws:WebSocket):
    await ws.accept()

    redis_task = asyncio.create_task(redis_receive(ws))
    try:
        while True:
            data = await ws.receive_text()
            await redis_client.publish('chat_channel2', data)
    except Exception as e:
        print("Client disconnected", e)
    finally:
        redis_task.cancel()
        await pubsub.unsubscribe('chat_channel')


if __name__ == "__main__":
    uvicorn.run("ws1:app", host="0.0.0.0", port=8000, reload=True)

