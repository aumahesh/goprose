// +build example

package internal

import (
	"context"
	"fmt"
	"net"
	"time"

	p "aumahesh.com/prose/example/models"
	"github.com/dmichael/go-multicast/multicast"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
)

const (
	inactivityTimeout = time.Duration(2) * time.Minute
	heartbeatInterval = time.Duration(1) * time.Minute
	maxDatagramSize   = 1024
)

var (
	a int64 = 0
)

type NeighborState struct {
	id              string
	state           *p.State
	discoveredAt    time.Time
	updatedAt       time.Time
	lastHeartBeatAt time.Time
	stateChangedAt  time.Time
	active          bool
}

type ProSe_impl_example struct {
	id             string
	state          *p.State
	mcastAddr      string
	mcastConn      *net.UDPConn
	receiveChannel chan *p.NeighborUpdate
	hbChannel      chan *p.NeighborHeartBeat
	neighborState  map[string]*NeighborState
}

func (this *ProSe_impl_example) init(id string, mcastAddr string) error {
	this.id = id
	this.state = &p.State{}
	this.mcastAddr = mcastAddr

	conn, err := multicast.NewBroadcaster(this.mcastAddr)
	if err != nil {
		return err
	}
	this.mcastConn = conn

	this.receiveChannel = make(chan *p.NeighborUpdate, 10)
	this.hbChannel = make(chan *p.NeighborHeartBeat, 10)

	this.neighborState = map[string]*NeighborState{
		this.id: &NeighborState{
			id:              this.id,
			state:           this.state,
			discoveredAt:    time.Now(),
			updatedAt:       time.Now(),
			lastHeartBeatAt: time.Now(),
			stateChangedAt:  time.Now(),
			active:          true,
		},
	}

	a = this.initConstanta()

	this.initState()

	return nil
}

func (this *ProSe_impl_example) initState() {
	this.state.Msg = ""
	this.state.St = 0
	this.state.Timer = 0
	this.state.Tmp = false
	this.state.X = 0

	this.state.Msg = this.initVaribleMsg()
	this.state.Tmp = this.initVaribleTmp()
	this.state.X = this.initVaribleX()

}

func (this *ProSe_impl_example) initConstanta() int64 {

	return int64(20)
}

func (this *ProSe_impl_example) initVaribleMsg() string {

	return "Hello, World!"
}

func (this *ProSe_impl_example) initVaribleTmp() bool {

	return true
}

func (this *ProSe_impl_example) initVaribleX() int64 {

	return ((-int64(10)) + int64(500))
}

func (this *ProSe_impl_example) EventHandler(ctx context.Context) {
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	for {
		select {
		case s := <-this.receiveChannel:
			_, ok := this.neighborState[s.Id]
			if !ok {
				this.neighborState[s.Id] = &NeighborState{
					id:             s.Id,
					discoveredAt:   time.Now(),
					active:         true,
					stateChangedAt: time.Now(),
				}
			}
			this.neighborState[s.Id].state = &p.State{

				Msg:   s.State.Msg,
				St:    s.State.St,
				Timer: s.State.Timer,
				Tmp:   s.State.Tmp,
				X:     s.State.X,
			}
			this.neighborState[s.Id].updatedAt = time.Now()
			this.evaluateNeighborStates()
			this.updateLocalState()
			if stateChanged := this.updateLocalState(); stateChanged {
				n, err := this.broadcastLocalState()
				if err != nil {
					log.Errorf("Error broadcasting local state to neighbors")
				} else {
					log.Debugf("%s: sent state update: %d bytes", this.id, n)
				}
			}
		case s := <-this.hbChannel:
			_, ok := this.neighborState[s.Id]
			if !ok {
				this.neighborState[s.Id] = &NeighborState{
					id:             s.Id,
					state:          &p.State{},
					discoveredAt:   time.Now(),
					active:         true,
					stateChangedAt: time.Now(),
					updatedAt:      time.Now(),
				}
			}
			this.neighborState[s.Id].lastHeartBeatAt = time.Now()
			this.evaluateNeighborStates()
			if stateChanged := this.updateLocalState(); stateChanged {
				n, err := this.broadcastLocalState()
				if err != nil {
					log.Errorf("Error broadcasting local state to neighbors")
				} else {
					log.Debugf("%s: sent heartbeat: %d bytes", this.id, n)
				}
			}
		case <-heartbeatTicker.C:
			this.sendHeartBeat()
		case <-ctx.Done():
			return
		}
	}
}

func (this *ProSe_impl_example) evaluateNeighborStates() {
	for id, nbr := range this.neighborState {
		if nbr.updatedAt.Add(inactivityTimeout).Before(time.Now()) {
			nbr.active = false
			nbr.stateChangedAt = time.Now()
			log.Warnf("neighbor %s became inactive at %s", id, time.Now())
		} else {
			if !nbr.active {
				log.Infof("neighbor %s became active at %s", id, time.Now())
			}
			nbr.active = true
			nbr.stateChangedAt = time.Now()
		}
		log.Debugf("state: %s: (%v) -> %+v", id, nbr.active, nbr.state)
	}
}

func (this *ProSe_impl_example) isNeighborUp(id string) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	return nbr.active
}

func (this *ProSe_impl_example) neighbors() map[string]*NeighborState {
	return this.neighborState
}

func (this *ProSe_impl_example) setNeighbor(id string, state bool) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	nbr.active = state
	return nbr.active
}

func (this *ProSe_impl_example) getNeighbor(id string, stateVariable string) (*NeighborState, error) {
	nbr, ok := this.neighborState[id]
	if !ok {
		return nil, fmt.Errorf("%s not found in neighbors", id)
	}
	return nbr, nil
}

func (this *ProSe_impl_example) doAction0() bool {
	stateChanged := false

	log.Debugf("Executing: doAction0")

	if (!(this.state.St == int64(1))) && (this.state.Timer >= a) {
		this.state.St = int64(2)
		this.state.Timer = int64(10)
		stateChanged = true
	}

	log.Debugf("doAction0: state changed: %v", stateChanged)

	return stateChanged
}

func (this *ProSe_impl_example) updateLocalState() bool {
	stateChanged := false

	statements := []func() bool{

		this.doAction0,
	}

	for _, stmtFunc := range statements {
		if changed := stmtFunc(); changed {
			stateChanged = true
		}
	}

	return stateChanged
}

func (this *ProSe_impl_example) broadcastLocalState() (int, error) {
	updMessage := &p.NeighborUpdate{
		Id: this.id,
		State: &p.State{

			Msg:   this.state.Msg,
			St:    this.state.St,
			Timer: this.state.Timer,
			Tmp:   this.state.Tmp,
			X:     this.state.X,
		},
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_StateUpdate,
		Src:  this.id,
		Msg:  &p.BroadcastMessage_Upd{updMessage},
	}

	return this.send(broadcastMessage)
}

func (this *ProSe_impl_example) sendHeartBeat() (int, error) {
	hbMessage := &p.NeighborHeartBeat{
		Id:     this.id,
		SentAt: time.Now().Unix(),
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_Heartbeat,
		Src:  this.id,
		Msg:  &p.BroadcastMessage_Hb{hbMessage},
	}

	return this.send(broadcastMessage)
}

func (this *ProSe_impl_example) send(msg *p.BroadcastMessage) (int, error) {
	log.Debugf("Sending: %+v", msg)
	data, err := proto.Marshal(msg)
	if err != nil {
		return 0, err
	}
	return this.mcastConn.Write(data)
}

func (this *ProSe_impl_example) msgHandler(src *net.UDPAddr, n int, b []byte) {
	log.Debugf("received message (%d bytes) from %s", n, src.String())
	broadcastMessage := &p.BroadcastMessage{}
	err := proto.Unmarshal(b[:n], broadcastMessage)
	if err != nil {
		log.Errorf("error unmarshalling proto from src %s: %s", src.String(), err)
		return
	}
	log.Debugf("received: %+v", broadcastMessage)
	switch broadcastMessage.Type {
	case p.MessageType_Heartbeat:
		this.hbChannel <- broadcastMessage.GetHb()
	case p.MessageType_StateUpdate:
		this.receiveChannel <- broadcastMessage.GetUpd()
	default:
		log.Errorf("invalid message type")
	}
}

func (this *ProSe_impl_example) Listener(ctx context.Context) {
	addr, err := net.ResolveUDPAddr("udp4", this.mcastAddr)
	if err != nil {
		log.Errorf("Error resolving mcast address: %s", err)
		return
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		log.Errorf("Error connecting to mcast address: %s", err)
	}

	conn.SetReadBuffer(maxDatagramSize)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			buffer := make([]byte, maxDatagramSize)
			numBytes, src, err := conn.ReadFromUDP(buffer)
			if err != nil {
				log.Fatal("ReadFromUDP failed:", err)
			}
			this.msgHandler(src, numBytes, buffer)
		}
	}
}
