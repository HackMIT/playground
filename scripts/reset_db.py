from redis.client import Redis

import json
import uuid

client = Redis()
client.flushdb()

with open("config/rooms.json") as f:
    rooms_data = json.loads(f.read())

for room in rooms_data:
    client.sadd("rooms", room["id"])
    client.hmset("room:" + room["id"], {
        "background": room["background"],
        "sponsor": str(room["sponsor"])
    })

    if "elements" in room:
        for element in room["elements"]:
            element_id = str(uuid.uuid4())
            element_data = {k: str(element[k]) for k in element}

            client.hmset("element:" + element_id, element_data)
            client.sadd("room:" + room["id"] + ":elements", element_id)

    if "hallways" in room:
        for hallway in room["hallways"]:
            hallway_id = str(uuid.uuid4())
            hallway_data = {k: str(hallway[k]) for k in hallway}

            client.hmset("hallway:" + hallway_id, hallway_data)
            client.sadd("room:" + room["id"] + ":hallways", hallway_id)
