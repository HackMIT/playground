# Database Schema

- `character:<character_id>` (hash)
- `conversation:<character_id>:<character_id>` (list)
  - Ordering of character IDs comes from a hash of each ID -- check `socket/hub.go` for more details
  - List of message IDs (in chronological order)
- `element:<element_id>` (hash)
- `hallway:<hallway_id>` (hash)
- `locations` (set)
- `location:<location_id>` (hash)
- `message:<message_id>` (hash)
- `quillToCharacter` (hash)
  - Mapping of Quill user IDs to Playground character IDs
- `room:<room_id>` (hash)
  - `room:<room_id>:elements` (set)
  - `room:<room_id>:hallways` (set)
  - `room:<room_id>:characters` (set)
- `rooms` (set)
- `song:<song_id>` (hash)
- `songs` (list)
