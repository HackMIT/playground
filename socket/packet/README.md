# packet

This is where we keep all of the packets that can be sent in Playground! Our packets generally fall into two classes (and they can belong to both):
- **client**: A packet sent by clients to the server
  - e.g. `get_messages` will be sent by a client who needs to load their private messages with someone else
- **server**: A packet sent by the server to clients
  - e.g. `messages` will be returned to a client after they send a `get_messages` packet

One example of a packet that belongs to both classes is `chat`, which a client can send to the server in order to send a chat message to the room. The server will in turn send this same packet back to the other clients.

## Adding a new client packet
1. Create a struct for this packet (see existing files for examples). Make sure to implement `PermissionCheck`, `MarshalBinary`, and `UnmarshalBinary`.
   - `PermissionCheck` returns true when the sender has permission to send that packet
2. Add the new packet to `parse.go` along with its identifier
3. Write the handler for this packet in hub.go

## Adding a new server packet
1. Create a struct for this packet. Make sure to implement `MarshalBinary` and `UnmarshalBinary`
2. Create an `Init` function with parameters that mirror what needs to be sent to the client
3. Use the `Init` function to create a new packet, and pass it to `h.Send` to send it to clients