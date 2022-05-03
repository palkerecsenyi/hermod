# Hourbase Hermod

Hermod is a high-compatibility alternative to gRPC and Protobuf that works across most browsers, mobile devices, and between servers. It uses the existing WebSocket protocol to establish high-performance connections, with full support for unary function calls as well as bi- and uni-directional streaming.

Hermod lets you use really simple YAML to define Units (i.e., `message MyMessage {}`) and Endpoints (i.e., `rpc MyFunction() returns ()`) without having to learn a new domain-specific language.

This repository contains the original Go version of Hermod. It contains code to compile YAML into Go, as well as the encoder and WebSocket server needed to host RPCs.

## Why?
gRPC is an amazing framework, and is perfect for providing blazing-fast highly-optimised RPC communication. However, it relies on a proxy for web client use, which is hard to justify for smaller projects.

Hermod uses WebSocket rather than HTTP/2 (which isn't always available). It's a proven and widely-used protocol with [very little framing](https://www.rfc-editor.org/rfc/rfc6455.html#section-5.2), meaning little overhead. Hermod lets you use a single WebSocket connection between a client and a server, and simply open and close lightweight 'sessions' within it. For more details, see the [Hermod Protocol](https://github.com/palkerecsenyi/hermod/blob/main/PROTOCOL.md).

Hermod tries to be better suited to building a full browser-facing API than gRPC is at the moment (although it works for server-server communication, too). It's much less complex or full-featured than other RPC frameworks, but includes all the essential features you need!

Hermod is also a [Norse messenger god](https://en.wikipedia.org/wiki/Herm%C3%B3%C3%B0r), unlike gRPC.

## Installation
To download the Hermod compiler, just go to [the Releases page](https://github.com/palkerecsenyi/hermod/releases). Download the compiler for your platform, rename it to `hermod`, and move it to a directory in your `PATH`.

For information about using the compiler, run `hermod --help`.

To install the necessary libraries used by generated Go code:

```bash
go get github.com/palkerecsenyi/hermod
```

## Concepts

- A `Unit` is similar to a 'message' in Protobuf. It defines a data structure and its fields. Both the client and server need to know the Unit's details to be able to encode and decode binary messages.

- A `Service` is like a service in Protobuf. At the moment, it just groups `Endpoint`s together, but additional functionality may be added in the future (e.g., customising configs per service).

- An `Endpoint` is a specific RPC function that clients can call. It has an input argument, and an output argument (both of which are optional). Both arguments can be streamed to allow multiple to be sent. An argument must be associated with a specific `Unit`, which the server and client expect to be exchanged.

To read more about Hermod's concepts and how to define YAML files, see the [YAML documentation](https://github.com/palkerecsenyi/hermod/blob/main/YAML.md).

## License
MIT
