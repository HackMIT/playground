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
        if room["id"] == "home":
            tile_start_x = 0.374
            tile_start_y = 0.552
            tile_separator = 0.0305

            for i in range(4):
                for j in range(7):
                    room["elements"].insert(0, {
                        "x": tile_start_x + (i + j) * tile_separator,
                        "y": tile_start_y + ((4 - i) + j) * tile_separator,
                        "tile": True
                    })

        for element in room["elements"]:
            if "tile" in element:
                del element["tile"]
                element["width"] = 0.052
                element["path"] = "tiles/blue1.svg"
                element["changingImagePath"] = True
                element["changingPaths"] = "tiles/blue1.svg,tiles/blue2.svg,tiles/blue3.svg,tiles/blue4.svg,tiles/green1.svg,tiles/green2.svg,tiles/pink1.svg,tiles/pink2.svg,tiles/pink3.svg,tiles/pink4.svg,tiles/yellow1.svg"
                element["changingInterval"] = 2000

            element_id = str(uuid.uuid4())
            element_data = {k: str(element[k]) for k in element}

            client.hmset("element:" + element_id, element_data)
            client.rpush("room:" + room["id"] + ":elements", element_id)

    if "hallways" in room:
        for hallway in room["hallways"]:
            hallway_id = str(uuid.uuid4())
            hallway_data = {k: str(hallway[k]) for k in hallway}

            client.hmset("hallway:" + hallway_id, hallway_data)
            client.sadd("room:" + room["id"] + ":hallways", hallway_id)
