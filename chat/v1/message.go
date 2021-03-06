package v1

import (
	"bytes"
	"errors"
	"time"
	"crypto/sha1"

	"github.com/kowala-tech/kUSD/common"
)

type Hash [20]byte

func (h Hash) String() string {
	return common.Bytes2Hex(h[:])
}

type Message struct {
	Room      string `json:"room"`
	Nickname  string `json:"nickname"`
	Text      string `json:"text"`
	TTL       uint32 `json:"ttl,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"` // ignored on seal
	// currently message signing is not implemented
	// Identity   string    `json:"identity,omitempty"`
}

func (mesg *Message) EqualNoStamp(a *Message) bool {
	return mesg.Room == a.Room &&
		mesg.Nickname == a.Nickname &&
		mesg.Text == a.Text &&
		mesg.TTL == a.TTL
}

func (mesg *Message) Hash() Hash {
	var m message
	m.encode(mesg, mesg.Timestamp)
	return m.hash()
}

type message struct {
	cDeathtime int64  // cached death time
	cHash      *Hash  // cached message hash
	body       []byte
}

func (m *message) validate() error {
	return nil
}

func (m *message) deathTime() int64 {
	if m.cDeathtime == 0 {
		ts, _ := m.fetchTimestamp()
		ttl, _ := m.fetchTTL()
		m.cDeathtime = ts + int64(ttl)
	}
	return m.cDeathtime
}

func (m *message) hash() Hash {
	if m.cHash == nil {
		h := Hash(sha1.Sum(m.body))
		m.cHash = &h
	}
	return *m.cHash
}

func (m *message) encode(mesg *Message, timestamp int64) error {
	var l byte
	var l2 uint16
	var b bytes.Buffer

	// timestamp
	for n := 0; n < 8; n++ {
		b.WriteByte(byte(timestamp >> (uint(n) * 8)))
	}

	// ttl
	for n := 0; n < 4; n++ {
		b.WriteByte(byte(mesg.TTL >> (uint(n) * 8)))
	}

	// room
	l = byte(len(mesg.Room))
	b.WriteByte(l)
	if l > 0 {
		b.Write([]byte(mesg.Room)[:l])
	}

	// nickname
	l = byte(len(mesg.Nickname))
	b.WriteByte(l)
	if l > 0 {
		b.Write([]byte(mesg.Nickname)[:l])
	}

	// text
	l2 = uint16(len(mesg.Text))
	b.WriteByte(byte(l2))
	b.WriteByte(byte(l2 >> 8))
	if l2 > 0 {
		b.Write([]byte(mesg.Text)[:l2])
	}

	m.body = b.Bytes()
	return nil
}

func (m *message) seal(mesg *Message) error {
	t := time.Now().Unix()
	return m.encode(mesg, t)
}

var badMessageError = errors.New("bad message")

func (m *message) fetchTimestamp() (int64, error) {
	var ts int64
	b := m.body
	if len(b) < 8 {
		return 0, badMessageError
	}
	for n := 0; n < 8; n++ {
		ts |= int64(b[n]) << (uint(n) * 8)
	}
	return ts, nil
}

func (m *message) fetchTTL() (uint32, error) {
	var ttl uint32
	if len(m.body) < 8+4 {
		return 0, badMessageError
	}
	b := m.body[8:]
	for n := 0; n < 4; n++ {
		ttl |= uint32(b[n]) << (uint(n) * 8)
	}
	return ttl, nil
}

func (m *message) open() (*Message, error) {
	var l int
	var b []byte
	var err error

	mesg := &Message{}

	// timestamp
	mesg.Timestamp, err = m.fetchTimestamp()
	if err != nil {
		return nil, err
	}

	// ttl
	mesg.TTL, err = m.fetchTTL()
	if err != nil {
		return nil, err
	}

	b = m.body[12:]

	// room
	l = int(b[0])
	if len(b) < l+1 {
		return nil, badMessageError
	}
	mesg.Room = string(b[1 : l+1])
	b = b[l+1:]

	// nickname
	l = int(b[0])
	if len(b) < l+1 {
		return nil, badMessageError
	}
	mesg.Nickname = string(b[1 : l+1])
	b = b[l+1:]

	// text
	l = int(b[0]) + (int(b[1]) << 8)
	if len(b) < l+1 {
		return nil, badMessageError
	}
	mesg.Text = string(b[2 : l+2])

	return mesg, nil
}
