package service

import "github.com/palkerecsenyi/hermod/encoder"

const (
	// Data flag is not just 0b00000000 so that it can clearly be distinguished in logs
	Data                 = 0b10101010
	ClientSessionRequest = 0b00001111
	ServerSessionAck     = 0b11110000
	Close                = 0b11111111
	CloseAck             = 0b11111110
)

type messageFrame struct {
	endpointId uint16
	flag       uint8
	sessionId  uint32
	data       []byte
}

func (frame *messageFrame) encode() []byte {
	var data []byte
	data = *encoder.Add16ToSlice(frame.endpointId, &data)
	data = append(data, frame.flag)
	data = *encoder.Add32ToSlice(frame.sessionId, &data)
	data = append(data, frame.data...)
	return data
}

func (frame *messageFrame) close() *[]byte {
	m := messageFrame{
		endpointId: frame.endpointId,
		flag:       Close,
		sessionId:  frame.sessionId,
		data:       []byte{},
	}
	encoded := m.encode()
	return &encoded
}

func (frame *messageFrame) closeAck() *[]byte {
	m := messageFrame{
		endpointId: frame.endpointId,
		flag:       CloseAck,
		sessionId:  frame.sessionId,
		data:       []byte{},
	}
	encoded := m.encode()
	return &encoded
}

type sessionFrame struct {
	endpointId uint16
	flag       uint8
	clientId   uint32
	sessionId  uint32
}

func (frame *sessionFrame) encode() []byte {
	var data []byte
	data = *encoder.Add16ToSlice(frame.endpointId, &data)
	data = append(data, frame.flag)
	data = *encoder.Add32ToSlice(frame.clientId, &data)
	data = *encoder.Add32ToSlice(frame.sessionId, &data)
	return data
}

func (frame *sessionFrame) ack(sessionId uint32) *[]byte {
	m := sessionFrame{
		endpointId: frame.endpointId,
		flag:       ServerSessionAck,
		clientId:   frame.clientId,
		sessionId:  sessionId,
	}
	encoded := m.encode()
	return &encoded
}
