# Smallchats matching server

<img width="208" alt="image" src="https://github.com/dinosoupy/smallchats/assets/47291303/4752a3e9-09f4-4ce3-93bb-9e4a03761688">

Smallchats is a video chat app where you hop on 15 min calls with strangers who share similar interests. This repository contains the matching server that pairs up users using algorithm based matchmaking.

## Key features
- Supports multiple users over websocket
- Event driven architecture
- Per-connection go routines for reading and writing event messages
- Asynchronous matching function
