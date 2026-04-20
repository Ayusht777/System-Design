from fastapi import FastAPI,Query,status
from redis import Redis
from datetime import datetime
redis_client = Redis(
    host='redis-10008.c17.us-east-1-4.ec2.cloud.redislabs.com',
    port=10008,
    decode_responses=True,
    username="default",
    password="569PS6T1nk7DFLckFattxyxA899OLxXo",
)

app = FastAPI(title="Offline Online Store")


@app.get('/status',status_code=status.HTTP_200_OK)
def status(users:list[str]=Query(...)):

    result={}
    for user in users:
        res = redis_client.get(user)
        ttl = redis_client.ttl(user) # Check how many seconds are left
        print(f"User: {user}, Value: {res}, TTL: {ttl}")
        if res:
            if float(res) < (datetime.now().timestamp() - 100):
                result[user]=False  # offline
            else:
                result[user]=True # online
        else:
            result[user]=False # offline

    return {"data":result}

@app.post('/status/{user_id}')
def update_status(user_id:str):
    """
    This endpoint will update the status of the user in the database.
    """
    res = redis_client.set(user_id,datetime.now().timestamp(),ex=100)
    return {"data":True if res else False}

    