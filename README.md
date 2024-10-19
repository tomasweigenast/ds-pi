# Distributed System PI Decimal Calculator

## Flow Diagram
1. `master` listens for UDP connections at broadcast address, port **9933**.
2. `worker` who wants to join the party dials with UDP the broadcast address, at the same port, searching for a connection.
3. When a connection is done, `worker` sends the payload `BEGIN [name]`, where *[name]* is the name of the worker (up to 10-characteres).
4. If the `master` is accepting connections and the `worker` is not already in, will reply with `OK [name]`, being *[name]* the name of the worker who requested to join. If the `worker` is already registered, `master` will reply with `REJECT`.
5. Once the `worker` receives the `OK` message, it knows the IP address of the `master` and now will dial a RPC client at that IP with port **9944**. `worker` UDP socket can be closed. 
6. bla bla bla idk how to continue
