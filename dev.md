## Dev Notes

Node shared doesn't make sense. It should either be associated with a Server
and not take a priv or it should not cache the result. Also, now I think I'm
going to create a shared key for each connection, but I'll still need a
handshake key.

Need to figure out how services will talk about nodes. By ID, or address?
Either? I'm just saying, we have options.

Overlay uses port 7667 for network, but that should be configurable.