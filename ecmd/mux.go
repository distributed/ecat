package ecmd

import (
	"errors"
	"launchpad.net/tomb"
)

type Multiplexer struct {
	muxlinked bool
	c         Commander
	mux       *Multiplexer

	reqchan chan interface{}
	tomb    tomb.Tomb

	chans []*muxChanControlBlock
	//cyclingChans []cyclingChan

	cyclepending  bool
	cycleRespChan chan error
}

func NewMultiplexer(c Commander) (m *Multiplexer, err error) {
	m = &Multiplexer{
		c:       c,
		reqchan: make(chan interface{}),
	}

	go m.loop()

	return
}

func (m *Multiplexer) loop() {
	defer m.tomb.Done()

down:
	for {
		if m.cyclepending {
			allcycling := true
			for _, cb := range m.chans {
				if cb.commandsOpen && !cb.cycling {
					allcycling = false
					break
				}
			}
			//log.Printf("allcycling %v\n", allcycling)

			if allcycling {
				//log.Printf("mux calling underlying Cycle\n")
				err := m.c.Cycle()
				//log.Printf("underlying Cycle err %v\n", err)

				nnot := 0
				for _, cb := range m.chans {
					if cb.cycling {
						cb.cyclingChan.responseChan <- err
						nnot++
					}
					cb.cycling = false
					cb.commandsOpen = false
				}
				//log.Printf("notified %d channels\n", nnot)

				m.cyclepending = false
				m.cycleRespChan <- err
				m.cycleRespChan = nil
			}
		}

		select {
		case req := <-m.reqchan:
			switch req := req.(type) {
			case muxChanNew:
				ec, err := m.c.New(req.datalen)
				req.responseChan <- muxChanNewResponse{ec, err}
				m.getCB(req.muxChannel).commandsOpen = true

			case muxChanCycle:
				// wait for mux controlled cycle
				//m.cyclingChans = append(m.cyclingChans, cyclingChan{req.muxChannel, req.responseChan})
				//log.Printf("mux chan is cycling")
				cb := m.getCB(req.muxChannel)
				if cb.cycling {
					req.responseChan <- errors.New("there already is a concurrent Cycle() pending on this mux channel")
				}

				cb.cycling = true
				cb.cyclingChan = cyclingChan{req.muxChannel, req.responseChan}

			case muxCycle:
				// mux controlled cycle
				//log.Printf("mux make cycle pending\n")
				if m.cycleRespChan != nil {
					req.responseChan <- errors.New("there already is a concurrent Cycle() on this multiplexer")
				}
				m.cyclepending = true
				m.cycleRespChan = req.responseChan

			case openCommander:
				c := &muxChannel{
					mux:             m,
					newResponseChan: make(chan muxChanNewResponse),
					errResponseChan: make(chan error),
				}

				m.chans = append(m.chans, &muxChanControlBlock{muxChannel: c})

				req.responseChan <- openCommanderResponse{c, nil}
			}
		case <-m.tomb.Dying():
			break down
		}
	}
}

func (m *Multiplexer) getCB(mc *muxChannel) *muxChanControlBlock {
	for _, cb := range m.chans {
		if cb.muxChannel == mc {
			return cb
		}
	}
	panic("missing mux chan control block")
}

func (m *Multiplexer) OpenCommander() (Commander, error) {
	req := openCommander{make(chan openCommanderResponse)}
	m.reqchan <- req
	resp := <-req.responseChan
	return resp.Commander, resp.error
}

func (c *Multiplexer) Cycle() error {
	req := muxCycle{make(chan error)}
	c.reqchan <- req
	return <-req.responseChan
}

type muxChanControlBlock struct {
	*muxChannel
	cyclingChan  cyclingChan
	commandsOpen bool
	cycling      bool
}

// cycle bound channel
type muxChannel struct {
	mux             *Multiplexer
	newResponseChan chan muxChanNewResponse
	errResponseChan chan error
}

func (mc *muxChannel) New(datalen int) (*ExecutingCommand, error) {
	mc.mux.reqchan <- muxChanNew{mc, datalen, mc.newResponseChan}
	resp := <-mc.newResponseChan
	return resp.ExecutingCommand, resp.error
}

func (mc *muxChannel) Cycle() error {
	mc.mux.reqchan <- muxChanCycle{mc, mc.errResponseChan}
	return <-mc.errResponseChan
}

func (mc *muxChannel) Close() error {
	return errors.New("nimpl")
}

type muxChanNew struct {
	*muxChannel
	datalen      int
	responseChan chan muxChanNewResponse
}

type muxChanNewResponse struct {
	*ExecutingCommand
	error
}

type muxChanCycle struct {
	*muxChannel
	responseChan chan error
}

type muxCycle struct {
	responseChan chan error
}

type openCommander struct {
	responseChan chan openCommanderResponse
}

type openCommanderResponse struct {
	Commander
	error
}

type cyclingChan struct {
	*muxChannel
	responseChan chan error
}
