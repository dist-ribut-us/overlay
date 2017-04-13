## Dev Notes

Node shared doesn't make sense. It should either be associated with a Server
and not take a priv or it should not cache the result. Also, now I think I'm
going to create a shared key for each connection, but I'll still need a
handshake key.

Need to figure out how services will talk about nodes. By ID, or address?
Either? I'm just saying, we have options.

Overlay uses port 7667 for network, but that should be configurable.

Handshake should not be exposed. We should silently do the handshake if there
is an attempt to send to node with no connection or an expired connection.

It might make sense to put more of the node data in message.Header - then any
service could fill out the information and Overlay could pull that out and
do AddNode automatically.

I might be setting TTL twice when receiving a SessionData query.

I could stack Headers like Russian nesting dolls to do a ToNet send...