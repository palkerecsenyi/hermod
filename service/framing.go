package service

import (
	"crypto/sha256"
	"github.com/palkerecsenyi/hermod/encoder"
)

const (
	Data                 = 0
	ClientSessionRequest = 1
	ServerSessionAck     = 2
	Close                = 3
	ErrorClientID        = 4
	ErrorSessionID       = 5
	Authentication       = 6
	AuthenticationAck    = 7
)

// AuthenticationEndpoint is a phantom endpoint that's used to signify an authentication message
const AuthenticationEndpoint = 0xFFFF

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

type authenticationAckFrame struct {
	tokenHash [32]byte
}

func authenticationAck(token string) authenticationAckFrame {
	return authenticationAckFrame{
		tokenHash: sha256.Sum256([]byte(token)),
	}
}

func (frame *authenticationAckFrame) encode() *[]byte {
	var data []byte
	data = *encoder.Add16ToSlice(AuthenticationEndpoint, &data)
	data = append(data, AuthenticationAck)
	data = append(data, frame.tokenHash[:]...)
	return &data
}
