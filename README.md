# p2p-msg - minimalistic app for messaging over udp p2p connection

## featuring STUN-like communication with protobuf over upd

So it basically went like this:

"Oh that's a cool article about piercing NAT and connecting peers directly, i should probably try it. Wonder if my ISP would block udp packets (spoiler: they do) or if my NAT is configured in such way that i won't be able to reach hosts in it (spoiler: it is)"

Since ~~nothing fucking works anymore~~ i couldn't get transport to work, there is no point in implementing actual messaging logic with encryption and shared keys and stuff. Maybe i'll bother to rewrite it to tcp later, who knows

