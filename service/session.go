package service

import (
	"errors"
	"fmt"
)

func newSessionsStruct() connectionSessions {
	return connectionSessions{
		sessions: map[uint32]chan *[]byte{},
	}
}

type connectionSessions struct {
	sessions map[uint32]chan *[]byte
}

func (c *connectionSessions) newSessionId() (uint32, error) {
	sessionId := uint32(0)
	for {
		if _, ok := c.sessions[sessionId]; ok {
			sessionId += 1
		} else {
			return sessionId, nil
		}

		if sessionId > 0xffffffff {
			break
		}
	}

	return 0, errors.New("no more session IDs available")
}

func (c *connectionSessions) createChannel(sessionId uint32) error {
	_, found := c.sessions[sessionId]
	if found {
		return errors.New(fmt.Sprintf("session id %d already in use", sessionId))
	}

	c.sessions[sessionId] = make(chan *[]byte)
	return nil
}

func (c *connectionSessions) getChannel(sessionId uint32) (chan *[]byte, error) {
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

	// these must happen together to avoid duplicate channel closure (a fatal error)
	close(channel)
	delete(c.sessions, sessionId)
	return nil
}

func (c *connectionSessions) initiateNewSession(req *Request, res *Response, frame messageFrame, endpoint func(*Request, *Response)) {
	channel, err := c.getChannel(frame.sessionId)
	if err != nil {
		res.SendError(err)
		return
	}

	forwardReq := Request{
		Context: req.Context,
		Data:    channel,
		Headers: req.Headers,
	}
	forwardRes := Response{
		sendFunction: func(dataToSend *[]byte, error bool) {
			responseFrame := messageFrame{
				flag: Data,
				data: *dataToSend,
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
