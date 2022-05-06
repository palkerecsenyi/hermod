package service

import (
	"errors"
	"fmt"
	"sync"
)

func newSessionsStruct() connectionSessions {
	return connectionSessions{
		sessions: map[uint32]chan *[]byte{},
	}
}

// connectionSessions keeps track of all sessions within a single WebSocket connection.
// Uses sync.RWMutex since the sessions map is written to/read from concurrently by a goroutine that is launched
// for each endpoint function call (which occurs once for each session).
type connectionSessions struct {
	sync.RWMutex
	sessions map[uint32]chan *[]byte
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
			return 0, errors.New("no more session IDs available")
		}
	}

	_, found := c.sessions[sessionId]
	// just to double-check
	if found {
		return 0, errors.New(fmt.Sprintf("session id %d already in use", sessionId))
	}

	c.sessions[sessionId] = make(chan *[]byte)
	return sessionId, nil
}

func (c *connectionSessions) getChannel(sessionId uint32) (chan *[]byte, error) {
	c.RLock()
	defer c.RUnlock()

	channel, ok := c.sessions[sessionId]
	if !ok {
		return nil, errors.New(fmt.Sprintf("session id %d not found", sessionId))
	}

	return channel, nil
}

func (c *connectionSessions) endSession(sessionId uint32) error {
	channel, err := c.getChannel(sessionId)
	if err != nil {
		return err
	}

	c.Lock()
	// these must happen together to avoid duplicate channel closure (a fatal error)
	// therefore, these calls should only be made through a call to this endSession function
	close(channel)
	delete(c.sessions, sessionId)
	c.Unlock()
	return nil
}

func (c *connectionSessions) initiateNewSession(req *Request, res *Response, frame messageFrame, endpoint func(*Request, *Response)) {
	channel, err := c.getChannel(frame.sessionId)
	if err != nil {
		errorFrame := createErrorSession(frame.endpointId, frame.sessionId, err.Error())
		res.Send(&errorFrame)
		return
	}

	forwardReq := Request{
		Context: req.Context,
		Data:    channel,
		Headers: req.Headers,
		Auth:    req.Auth,
	}
	forwardRes := Response{
		sendFunction: func(dataToSend *[]byte, error bool) {
			if error {
				errorFrame := createErrorSession(frame.endpointId, frame.sessionId, string(*dataToSend))
				res.Send(&errorFrame)
				return
			}

			responseFrame := messageFrame{
				endpointId: frame.endpointId,
				sessionId:  frame.sessionId,
				flag:       Data,
				data:       *dataToSend,
			}
			encoded := responseFrame.encode()
			res.Send(&encoded)
		},
	}

	go func() {
		endpoint(&forwardReq, &forwardRes)
		// if there's an error, the session has probably been manually closed elsewhere
		_ = c.endSession(frame.sessionId)
		res.Send(frame.close())
	}()
}
