# Hourbase Hermod

Hermod is a high-compatibility alternative to gRPC and Protobuf that works across most browsers, mobile devices, and between servers. It uses the existing WebSocket protocol to establish high-performance connections, with full support for unary function calls as well as bi- and uni-directional streaming.

Hermod lets you use really simple YAML to define Units (i.e., `message MyMessage {}`) and Endpoints (i.e., `rpc MyFunction() returns ()`) without having to learn a new domain-specific language.

This repository contains the original Go version of Hermod. It contains code to compile YAML into Go, as well as the encoder and WebSocket server needed to host RPCs.
