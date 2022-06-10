package framing

import (
	"crypto/sha256"
	"github.com/palkerecsenyi/hermod/encoder"
)

const (
	Data                         = 0
	ClientSessionRequest         = 1
	ClientSessionRequestWithAuth = 0b10000001
	ServerSessionAck             = 2
	Close                        = 3
	ErrorClientID                = 4
	ErrorSessionID               = 5
	Authentication               = 6
	AuthenticationAck            = 7
)

// AuthenticationEndpoint is a phantom endpoint that's used to signify an authentication message
const AuthenticationEndpoint = 0xFFFF

type MessageFrame struct {
	EndpointId uint16
	Flag       uint8
	SessionId  uint32
	Data       []byte
}

func CreateErrorClient(endpointId uint16, clientId uint32, message string) []byte {
	errorFrame := MessageFrame{
		EndpointId: endpointId,
		SessionId:  clientId,
		Flag:       ErrorClientID,
		Data:       []byte(message),
	}

	return errorFrame.Encode()
}

func CreateErrorSession(endpointId uint16, sessionId uint32, message string) []byte {
	errorFrame := MessageFrame{
		EndpointId: endpointId,
		SessionId:  sessionId,
		Flag:       ErrorSessionID,
		Data:       []byte(message),
	}

	return errorFrame.Encode()
}

func (frame MessageFrame) Encode() []byte {
	var data []byte
	data = *encoder.Add16ToSlice(frame.EndpointId, &data)
	data = append(data, frame.Flag)
	data = *encoder.Add32ToSlice(frame.SessionId, &data)
	data = append(data, frame.Data...)
	return data
}

func (frame *MessageFrame) Close() []byte {
	m := MessageFrame{
		EndpointId: frame.EndpointId,
		Flag:       Close,
		SessionId:  frame.SessionId,
		Data:       []byte{},
	}
	encoded := m.Encode()
	return encoded
}

type SessionFrame struct {
	EndpointId uint16
	Flag       uint8
	ClientId   uint32
	Token      string
	SessionId  *uint32
}

func (frame *SessionFrame) Encode() []byte {
	var data []byte
	data = *encoder.Add16ToSlice(frame.EndpointId, &data)
	data = append(data, frame.Flag)
	data = *encoder.Add32ToSlice(frame.ClientId, &data)

	if frame.SessionId != nil {
		data = *encoder.Add32ToSlice(*frame.SessionId, &data)
	} else if frame.Flag == ClientSessionRequestWithAuth {
		data = append(data, []byte(frame.Token)...)
	}

	return data
}

func (frame *SessionFrame) Ack(sessionId uint32) *[]byte {
	m := SessionFrame{
		EndpointId: frame.EndpointId,
		Flag:       ServerSessionAck,
		ClientId:   frame.ClientId,
		SessionId:  &sessionId,
	}
	encoded := m.Encode()
	return &encoded
}

type AuthenticationAckFrame struct {
	TokenHash [32]byte
}

func NewAuthenticationAck(token string) AuthenticationAckFrame {
	return AuthenticationAckFrame{
		TokenHash: sha256.Sum256([]byte(token)),
	}
}

func (frame *AuthenticationAckFrame) Encode() *[]byte {
	var data []byte
	data = *encoder.Add16ToSlice(AuthenticationEndpoint, &data)
	data = append(data, AuthenticationAck)
	data = append(data, frame.TokenHash[:]...)
	return &data
}
