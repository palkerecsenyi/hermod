# Hermod Protocol
The Hermod Protocol is a messaging and RPC protocol built on top of WebSocket. In theory, it works with any full-duplex transmission medium, but this document specifically describes it using WebSocket.

It involves a simple handshake, bi-directional messaging, error handling, and closing. The idea is to support many sessions between a server and client over a single WebSocket connection to reduce overhead.

## Basic message structure
All binary WebSocket messages must include the Endpoint ID (a 16-bit unsigned integer). This is defined in the Hermod YAML file. Both the server and client must have the same compiled Hermod YAML configuration so that they mutually understand the Endpoint that an ID refers to.

Messages must also contain a Flag that describes the intent or content of the message. This is an 8-bit number with pre-defined meanings as follows:

- `0000 0000` `Data`
- `0000 0001` `ClientSessionRequest` — Client requesting a new session from server
- `0000 0010` `ServerSessionAck` — Server confirming the new session to the client
- `0000 0011` `Close` — Server/Client requesting other party to close the session
- `0000 0100` `CloseAck` — Server/Client confirming to have closed the session in response to `Close`
- `0000 0101` `ErrorClientID` — Server sending an error message during the handshake process before a Session ID has been assigned
- `0000 0110` `ErrorSessionID` — Server sending an error message after a Session ID has been communicated to the client

An 8-bit number is used to allow for future extensions.

All text-based WebSocket messages are to be interpreted as error messages.

## Handshake
First, a client requests to open a session for a particular Endpoint.

To open a session, the client must generate a Client ID. This is an unsigned 32-bit number. No other in-process handshake may use the same Client ID. As long as this condition of uniqueness is met, the client may use any unsigned 32-bit number as the Client ID.

The client must send the following transmission:

| Endpoint ID (16 bits) | Flag: `ClientSessionRequest` | Client ID (32 bits) |
|-----------------------|------------------------------|---------------------|

The server must initiate a call to the associated user-declared Endpoint Handler corresponding with the Endpoint ID. If no Endpoint Handler has been declared for the Endpoint ID, the server must respond with a text-based WebSocket message with the content `endpoint not found` and discontinue the handshake.

If the Endpoint Handler is successfully located and called, the server must generate a Session ID in response. This is also an unsigned 32-bit number. No other Session within the WebSocket connection may use the same Session ID. As long as this condition of uniqueness is met, the server may use any unsigned 32-bit number as the Session ID.

The server must send the following response to a successfully handled ClientSessionRequest:

| Endpoint ID (16 bits) | Flag: `ServerSessionAck` | Client ID (32 bits) | Session ID (32 bits) |
|-----------------------|--------------------------|---------------------|----------------------|

The handshake is now complete, and the client may now disregard the Client ID or re-use it for future handshakes. The client must, however, store the Session ID assigned by the server and use it in future messages.

## Data messages
Regular data messages may be sent after a successful handshake. They may be sent either client-to-server or server-to-client. They must be formatted as follows:

| Endpoint ID (16 bits) | Flag: `Data` | Session ID (32 bits) | Encoded Hermod Unit |
|-----------------------|--------------|----------------------|---------------------|

The Encoded Hermod Unit must be an instance of the corresponding `in` unit if the message is sent from the client to the server, and an instance of the corresponding `out` unit if the message is sent from the server to the client.

## Error messages
Errors can be transmitted in two ways:

- A text WebSocket message must always be regarded as a fatal error by the client. Upon receiving a text WebSocket message, the client must terminate the WebSocket connection. Text-based errors are used only when an error concerns the entire WebSocket connection (not just a particular session) or when the server is unable to identify the Endpoint or the Client.
- A binary WebSocket message using either the `ErrorClientID` or `ErrorSessionID` flag, formatted as shown below:

| Endpoint ID (16 bits) | Flag: `ErrorClientID` or `ErrorSessionID` | Session ID or Client ID (depending on flag) (32 bits) | String error message |
|-----------------------|-------------------------------------------|-------------------------------------------------------|----------------------|

Upon receiving a binary error message, the client must terminate the session (but should not terminate the WebSocket connection). The client does not need to send a `Close` message to close the session in this case.

## Closing a session
When the Endpoint Handler finishes executing or when the client requests closing a session, a message using the `Close` flag can be sent to notify the other party of the intent to close the session. This is formatted as follows:

| Endpoint ID (16 bits) | Flag: `Close` | Session ID (32 bits) |
|-----------------------|---------------|----------------------|

After sending this message, the sending party must disregard all proceeding incoming messages, except for `CloseAck` and `ErrorSessionID`. The sending party must not stop listening for incoming messages with the specified Session ID until the `CloseAck`  

Upon receiving this message, the receiving party should stop sending outgoing messages, except for `CloseAck`. It should send a single `CloseAck` message formatted exactly as the `Close` message above, except with the `CloseAck` flag.

Upon receiving a `CloseAck` message, the receiving party must stop listening for incoming messages with the specified Session ID.

If a `ErrorSessionID` message is received before a `CloseAck` message, the receiving party must terminate the session immediately.

### Closure of underlying protocol
The client or the server may terminate the underlying WebSocket connection. This terminates all sessions within the connection immediately. The party closing the connection should not send `Close` messages to each session within the connection. 
