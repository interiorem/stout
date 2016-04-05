package cocaine12

import (
	"fmt"
)

const (
	v1Handshake = 0
	v1Heartbeat = 0
	v1Invoke    = 0
	v1Write     = 0
	v1Error     = 1
	v1Close     = 2
	v1Terminate = 1

	v1UtilitySession = 1
)

type v1Protocol struct {
	maxSession uint64
}

func newV1Protocol() protocolDispather {
	return &v1Protocol{
		maxSession: 1,
	}
}

func (v *v1Protocol) onMessage(p protocolHandler, msg *Message) error {
	if msg.Session == v1UtilitySession {
		return v.dispatchUtilityMessage(p, msg)
	}

	if v.maxSession < msg.Session {
		// It must be Invkoke
		if msg.MsgType != v1Invoke {
			return fmt.Errorf("new session %d must start from invoke type %d, not %d\n",
				msg.Session, v1Invoke, msg.MsgType)
		}

		v.maxSession = msg.Session
		return p.onInvoke(msg)
	}

	switch msg.MsgType {
	case v1Write:
		p.onChunk(msg)
	case v1Close:
		p.onChoke(msg)
	case v1Error:
		p.onError(msg)
	default:
		return fmt.Errorf("an invalid message type: %d, message %v", msg.MsgType, msg)
	}
	return nil
}

func (v *v1Protocol) isChunk(msg *Message) bool {
	return msg.MsgType == v1Write
}

func (v *v1Protocol) dispatchUtilityMessage(p protocolHandler, msg *Message) error {
	switch msg.MsgType {
	case v1Heartbeat:
		p.onHeartbeat(msg)
	case v1Terminate:
		p.onTerminate(msg)
	default:
		return fmt.Errorf("an invalid utility message type %d", msg.MsgType)
	}

	return nil
}

func (v *v1Protocol) newHandshake(id string) *Message {
	return newHandshakeV1(id)
}

func (v *v1Protocol) newHeartbeat() *Message {
	return newHeartbeatV1()
}

func (v *v1Protocol) newChoke(session uint64) *Message {
	return newChokeV1(session)
}

func (v *v1Protocol) newChunk(session uint64, data []byte) *Message {
	return newChunkV1(session, data)
}

func (v *v1Protocol) newError(session uint64, category, code int, message string) *Message {
	return newErrorV1(session, category, code, message)
}

func newHandshakeV1(id string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: v1UtilitySession,
			MsgType: v1Handshake,
		},
		Payload: []interface{}{id},
	}
}

func newHeartbeatV1() *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: v1UtilitySession,
			MsgType: v1Heartbeat,
		},
		Payload: []interface{}{},
	}
}

func newInvokeV1(session uint64, event string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: v1Invoke,
		},
		Payload: []interface{}{event},
	}
}

func newChunkV1(session uint64, data []byte) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: v1Write,
		},
		Payload: []interface{}{data},
	}
}

func newErrorV1(session uint64, category, code int, message string) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: v1Error,
		},
		Payload: []interface{}{[2]int{category, code}, message},
	}
}

func newChokeV1(session uint64) *Message {
	return &Message{
		CommonMessageInfo: CommonMessageInfo{
			Session: session,
			MsgType: v1Close,
		},
		Payload: []interface{}{},
	}
}
