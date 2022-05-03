package service

import (
	"github.com/palkerecsenyi/hermod/encoder"
)

const (
	Data                 = 0
	ClientSessionRequest = 1
	ServerSessionAck     = 2
	Close                = 3
	CloseAck             = 4
	ErrorClientID        = 5
	ErrorSessionID       = 6
)

type messageFrame struct {
	endpointId uint16
	flag       uint8
	sessionId  uint32
	data       []byte
}

func createErrorClient(endpointId uint16, clientId uint32, message string) []byte {
	errorFrame := messageFrame{
		endpointId: endpointId,
		sessionId:  clientId,
		flag:       ErrorClientID,
		data:       []byte(message),
	}

	return errorFrame.encode()
}

func createErrorSession(endpointId uint16, sessionId uint32, message string) []byte {
	errorFrame := messageFrame{
		endpointId: endpointId,
		sessionId:  sessionId,
		flag:       ErrorSessionID,
		data:       []byte(message),
	}

	return errorFrame.encode()
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
