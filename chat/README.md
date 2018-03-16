# P2P Chat Board

The chat is a go-ethereum service implementing decentralized chat board over ethereum p2p network.

The P2P Chat copies board messages over p2p network and stores temporarily on every node running the Chat service. All messages are anonymous, however, a message can be signed to identify the sender. A user can choose any nickname to represent his identity with or without a signature.

Every message has a room selector, it can be used by interface applications and bots to grouping messages into chatting spaces.

The P2P Chat has public API in cht namespace available by RPC on every node starting the Chat and the RPC services.

There are four principal entities: Chat, ring, peer, and ChatAPI. 

The Chat is an Ethereum service. It can be registered to start on the Ethereum node. 

The peer is an representation for connected p2p peers and can communicate by p2p ethereum protocol. 

Logically ring is an infinity queue of messages passing through the p2p node. Of cause, actually, there is no infinity queue. There is a cycle-buffer containing a limited count of messages. Also, the ring contains hashes of all passed and not expired messages to prevent double passing. ChatAPI and peers can enqueue new messages to the ring using the ring.enqueue function. Also, peers can get messages from the ring using ring.get function. 

The ring.get function has one argument - index of the message. Since ring does not really contain all messages it returns the first presented message from the specified index and index of the next message. So peers can iterate all messages from the ring starting from 0. 

When a peer connected to the p2p node, peer.broadcast goroutine starts to send messages from the ring starting at 0 to the connected peer. It has some performance problem and can be optimized, but for simple messaging service with small ring size, it's not so important. 

The chat application which user uses to communicate over p2p can access to messages by that ChatAPI. It has two methods: ChatAPI.Post and ChatAPI.Poll. To starts gather messages user need to call ChatAPI.Poll('room to pool') firstly. It starts watcher for the selected room. Then the user can send messages to the room by ChatAPI.Post and getting messages by ChatAPI.Poll.

The ChatAPI is availabe in cht namespace
```text
>go run node.go
Welcome to the kUSD JavaScript console!

instance: test-cht/v1.0.0-unstable/windows-amd64/go1.10
 modules: admin:1.0 cht:1.0 debug:1.0 rpc:1.0 web3:1.0

> cht.poll(".")
null
> cht.post({room:".",text:"hello!",ttl:10000})
true
> cht.poll(".")
[{
    nickname: "",
    room: ".",
    text: "hello!",
    timestamp: 1521218876,
    ttl: 10000
}]
```
