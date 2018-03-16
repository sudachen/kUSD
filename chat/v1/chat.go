package v1

import (
	"errors"
	"sync"
	"time"

	"github.com/kowala-tech/kUSD/log"
	"github.com/kowala-tech/kUSD/p2p"
	"github.com/kowala-tech/kUSD/rpc"
	ethn "github.com/kowala-tech/kUSD/node"
)

const (
	statusCode           = 0
	messagesCode         = 1
	NumberOfMessageCodes = 128
)

const (
	ProtocolVersion    = uint64(1) // Protocol version number
	ProtocolVersionStr = "1.0"     // The same, as a string
	ProtocolName       = "cht"     // Nickname of the protocol in geth
	expireTimeout      = 2 * time.Second
	watchTimeout       = 100 * time.Millisecond
	messageQueueLimit  = 1024
	messageRingLength  = 1024
)

type Watcher interface {
	Watch(*Message)
}

type Chat struct {
	protocol p2p.Protocol

	wmu      sync.Mutex
	watchers []Watcher
	pmu      sync.Mutex
	peers    map[*peer]struct{}
	ring     *ring
	quit     chan struct{}

	cfg Config
}

func New(cfg *Config) *Chat {
	if cfg == nil {
		cfg = &DefaultConfig
	}

	c := &Chat{
		cfg:      *cfg,
		ring:     newRing(),
		quit:     make(chan struct{}),
		watchers: make([]Watcher, 0),
		peers:    make(map[*peer]struct{}),
	}

	c.protocol = p2p.Protocol{
		Name:    ProtocolName,
		Version: uint(ProtocolVersion),
		Length:  NumberOfMessageCodes,
		Run:     c.handlePeer,
		NodeInfo: func() interface{} {
			return map[string]interface{}{
				"version":        ProtocolVersionStr,
				"maxMessageSize": uint32(cfg.MaxP2pMessageSize),
			}
		},
	}

	return c
}

func (c *Chat) MaxChatMessageSize() int {
	return c.cfg.MaxChatMessageSize
}

func (c *Chat) MaxP2pMessageSize() uint32 {
	return uint32(c.cfg.MaxP2pMessageSize)
}

func (c *Chat) Protocols() []p2p.Protocol {
	return []p2p.Protocol{c.protocol}
}

func (c *Chat) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: ProtocolName,
			Version:   ProtocolVersionStr,
			Service:   NewChatAPI(c),
			Public:    true,
		},
	}
}

func (c *Chat) watch() {
	delay := time.NewTicker(watchTimeout)
	var index uint64
	var m *message
	for {
		if done2(c.quit, delay.C) {
			return
		}

		index, m = c.ring.get(index)
		for m != nil {

			log.Trace("watch", "hash", m.hash())

			if mesg, err := m.open(); err != nil {
				log.Error("mesg open error", "m", m, "err", err)
			} else {
				c.watchMesg(mesg)
			}
			if done(c.quit) {
				return
			}
			index, m = c.ring.get(index)
		}
	}
}

func (c *Chat) watchMesg(mesg *Message) {
	c.wmu.Lock()
	defer c.wmu.Unlock()

	log.Trace("watchMesg", "mesg", mesg)

	if len(c.watchers) > 0 {
		for _, w := range c.watchers {
			w.Watch(mesg)
		}
	}
}

func (c *Chat) handlePeer(p2 *p2p.Peer, rw p2p.MsgReadWriter) error {
	p := newPeer(c.ring, p2, rw, &c.cfg)
	if err := p.handshake(); err != nil {
		return err
	}
	c.attach(p)
	defer c.detach(p)
	return p.loop()
}

func (c *Chat) Start(server *p2p.Server) error {
	log.Info("started chat v." + ProtocolVersionStr)
	go c.watch()
	go c.ring.dequeue(c.quit)
	go c.ring.expire(c.quit)
	return nil
}

func (c *Chat) Stop() error {
	close(c.quit)
	return nil
}

var AlreadySubscribedError = errors.New("already subscribed")

func (c *Chat) Subscribe(w Watcher) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	for _, x := range c.watchers {
		if w == x {
			return AlreadySubscribedError
		}
	}
	c.watchers = append(c.watchers, w)
	return nil
}

var NotSubscribedError = errors.New("not subscribed")

func (c *Chat) Unsubscribe(w Watcher) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	for i, x := range c.watchers {
		if w == x {
			L := len(c.watchers) - 1
			if L > 0 && i != L {
				c.watchers[i] = c.watchers[L]
			}
			c.watchers = c.watchers[:L]
			return nil
		}
	}
	return NotSubscribedError
}

func (c *Chat) enqueue(m *message) {
	c.ring.enqueue(m)
}

func (c *Chat) Send(mesg *Message) error {
	m := &message{}
	log.Trace("send", "mesg", mesg)
	if err := m.seal(mesg); err != nil {
		return err
	}
	if len(m.body) > c.MaxChatMessageSize() {
		return errors.New("message to long")
	}
	log.Trace("enqueue", "m", m)
	c.enqueue(m)
	return nil
}

func (c *Chat) attach(p *peer) {
	c.pmu.Lock()
	defer c.pmu.Unlock()
	c.peers[p] = struct{}{}
}

func (c *Chat) detach(p *peer) {
	c.pmu.Lock()
	defer c.pmu.Unlock()
	delete(c.peers, p)
}

func (c *Chat) get(oldIndex uint64) (uint64, *message) {
	return c.ring.get(oldIndex)
}

func (c *Chat) RegisterService(stack *ethn.Node) error {
	if err := stack.Register(func(n *ethn.ServiceContext) (ethn.Service, error){
		return c, nil
	}); err != nil {
		return err
	}
	return nil
}
