package v1

import (
	"sync"
	"time"
	"encoding/json"

	"github.com/kowala-tech/kUSD/log"
)

const apiExpireTimeout = 5*time.Minute

type ChatAPI struct {
	c        *Chat
	mu       sync.Mutex
	rooms    map[string][]*Message
	lastPoll time.Time
	w        *watcher
}

func NewChatAPI(c *Chat) *ChatAPI {
	api := &ChatAPI{c: c, rooms: make(map[string][]*Message)}
	return api
}

func (api *ChatAPI) Version() string {
	return ProtocolVersionStr
}

func (api *ChatAPI) Post(mesg *Message) (bool, error) {
	log.Trace("cht.post", "mesg", mesg)
	if err := api.c.Send(mesg); err != nil {
		return false, err
	}
	return true, nil
}

func (api *ChatAPI) PollStr(room string) (r string, err error) {
	ms, err := api.Poll(room)
	if err != nil {
		return
	}
	b, err := json.Marshal(ms)
	r = string(b)
	return
}

func (api *ChatAPI) Poll(room string) (ms []*Message, err error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	log.Trace("cht.poll", "room", room, "rooms", api.rooms)

	api.lastPoll = time.Now()

	if x, ok := api.rooms[room]; ok {
		ms = x
		api.rooms[room] = x[:0]
	} else {
		api.rooms[room] = nil
	}

	if api.w == nil {
		api.w = &watcher{api}
		api.c.Subscribe(api.w)
		go api.expire()
	}

	return
}

func (api *ChatAPI) expire() {
	for {
		t := time.Now()
		<-time.After(time.Minute)
		api.mu.Lock()
		p := api.lastPoll
		api.mu.Unlock()
		if t.Before(p) {
			api.mu.Lock()
			api.c.Unsubscribe(api.w)
			api.w = nil
			api.mu.Unlock()
			return
		}
	}
}

type watcher struct {
	*ChatAPI
}

func (w *watcher) Watch(mesg *Message) {
	w.mu.Lock()
	defer w.mu.Unlock()

	log.Trace("cht.watch", "mesg", mesg)

	if x, ok := w.rooms[mesg.Room]; ok {
		w.rooms[mesg.Room] = append(x, mesg)
	}
}
