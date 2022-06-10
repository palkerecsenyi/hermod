package service

import (
	"fmt"
	"github.com/palkerecsenyi/hermod/framing"
	"sync"
)

func newSessionsStruct() connectionSessions {
	return connectionSessions{
		sessions: map[uint32]sessionData{},
	}
}

// connectionSessions keeps track of all sessions within a single WebSocket connection.
// Uses sync.RWMutex since the sessions map is written to/read from concurrently by a goroutine that is launched
// for each endpoint function call (which occurs once for each session).
type connectionSessions struct {
	sync.RWMutex
	sessions map[uint32]sessionData
}

type sessionData struct {
	channel chan *[]byte
	auth    *authProvider
}

func (c *connectionSessions) createNewSession() (uint32, error) {
	c.Lock()
	defer c.Unlock()

	sessionId := uint32(0)
	for {
		if _, ok := c.sessions[sessionId]; ok {
			sessionId += 1
		} else {
			break
		}

		if sessionId > 0xffffffff {
			return 0, fmt.Errorf("no more session IDs available")
		}
	}

	_, found := c.sessions[sessionId]
	// just to double-check
	if found {
		return 0, fmt.Errorf("session id %d already in use", sessionId)
	}

	c.sessions[sessionId] = sessionData{
		channel: make(chan *[]byte),
	}
	return sessionId, nil
}

func (c *connectionSessions) setSessionAuth(sessionId uint32, auth *authProvider) error {
	c.Lock()
	defer c.Unlock()

	sd, ok := c.sessions[sessionId]
	if !ok {
		return fmt.Errorf("session id %d not found", sessionId)
	}

	sd.auth = auth
	c.sessions[sessionId] = sd
	return nil
}

func (c *connectionSessions) getSessionData(sessionId uint32) (*sessionData, error) {
	c.RLock()
	defer c.RUnlock()

	sd, ok := c.sessions[sessionId]
	if !ok {
		return nil, fmt.Errorf("session id %d not found", sessionId)
	}

	return &sd, nil
}

func (c *connectionSessions) endSession(sessionId uint32) error {
	sd, err := c.getSessionData(sessionId)
	if err != nil {
		return err
	}

	c.Lock()
	defer c.Unlock()
	// these must happen together to avoid duplicate channel closure (a fatal error)
	// therefore, these calls should only be made through a call to this endSession function
	close(sd.channel)
	delete(c.sessions, sessionId)
	return nil
}

func (c *connectionSessions) initiateNewSession(req *Request, res *Response, frame framing.MessageFrame, endpoint func(*Request, *Response)) {
	sd, err := c.getSessionData(frame.SessionId)
	if err != nil {
		errorFrame := framing.CreateErrorSession(frame.EndpointId, frame.SessionId, err.Error())
		res.Send(&errorFrame)
		return
	}

	localAuthProvider := req.Auth
	if localAuthProvider == nil && sd.auth != nil {
		localAuthProvider = &AuthAPI{
			sd.auth,
		}
	}

	forwardReq := Request{
		Context: req.Context,
		Data:    sd.channel,
		Headers: req.Headers,
		Auth:    localAuthProvider,
	}
	forwardRes := Response{
		sendFunction: func(dataToSend *[]byte, error bool) {
			if error {
				errorFrame := framing.CreateErrorSession(frame.EndpointId, frame.SessionId, string(*dataToSend))
				res.Send(&errorFrame)
				return
			}

			responseFrame := framing.MessageFrame{
				EndpointId: frame.EndpointId,
				SessionId:  frame.SessionId,
				Flag:       framing.Data,
				Data:       *dataToSend,
			}
			encoded := responseFrame.Encode()
			res.Send(&encoded)
		},
	}

	go func() {
		endpoint(&forwardReq, &forwardRes)
		// if there's an error, the session has probably been manually closed elsewhere
		_ = c.endSession(frame.SessionId)
		closeMessage := frame.Close()
		res.Send(&closeMessage)
	}()
}
