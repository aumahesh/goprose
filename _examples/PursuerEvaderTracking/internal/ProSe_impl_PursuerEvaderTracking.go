// +build PursuerEvaderTracking

package internal

import (
	"context"
	"fmt"
	"net"
	"time"

	"math/rand"

	p "aumahesh.com/prose/PursuerEvaderTracking/models"
	"github.com/dmichael/go-multicast/multicast"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
)

const (
	inactivityTimeout = time.Duration(2) * time.Minute
	heartbeatInterval = time.Duration(1) * time.Minute
	maxDatagramSize   = 1024
)

var ()

type NeighborState struct {
	id              string
	state           *p.State
	discoveredAt    time.Time
	updatedAt       time.Time
	lastHeartBeatAt time.Time
	stateChangedAt  time.Time
	active          bool
}

type ProSe_impl_PursuerEvaderTracking struct {
	id             string
	state          *p.State
	mcastAddr      string
	mcastConn      *net.UDPConn
	receiveChannel chan *p.NeighborUpdate
	hbChannel      chan *p.NeighborHeartBeat
	neighborState  map[string]*NeighborState
}

func (this *ProSe_impl_PursuerEvaderTracking) init(id string, mcastAddr string) error {
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

	this.initState()

	return nil
}

func (this *ProSe_impl_PursuerEvaderTracking) initState() {
	this.state.DetectTimestamp = 0
	this.state.Dist2Evader = 0
	this.state.IsEvaderHere = false
	this.state.P = ""

}

func (this *ProSe_impl_PursuerEvaderTracking) EventHandler(ctx context.Context) {
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

				DetectTimestamp: s.State.DetectTimestamp,
				Dist2Evader:     s.State.Dist2Evader,
				IsEvaderHere:    s.State.IsEvaderHere,
				P:               s.State.P,
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

func (this *ProSe_impl_PursuerEvaderTracking) evaluateNeighborStates() {
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

func (this *ProSe_impl_PursuerEvaderTracking) isNeighborUp(id string) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	return nbr.active
}

func (this *ProSe_impl_PursuerEvaderTracking) neighbors() map[string]*NeighborState {
	return this.neighborState
}

func (this *ProSe_impl_PursuerEvaderTracking) setNeighbor(id string, state bool) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	nbr.active = state
	return nbr.active
}

func (this *ProSe_impl_PursuerEvaderTracking) getNeighbor(id string, stateVariable string) (*NeighborState, error) {
	nbr, ok := this.neighborState[id]
	if !ok {
		return nil, fmt.Errorf("%s not found in neighbors", id)
	}
	return nbr, nil
}

func (this *ProSe_impl_PursuerEvaderTracking) doAction0() bool {
	stateChanged := false

	log.Debugf("Executing: doAction0")

	if true {
		temp0 := rand.Int63n(int64(2))
		this.state.IsEvaderHere = (temp0 == int64(1))
		stateChanged = true
	}

	log.Debugf("doAction0: state changed: %v", stateChanged)

	return stateChanged
}

func (this *ProSe_impl_PursuerEvaderTracking) doAction1() bool {
	stateChanged := false

	log.Debugf("Executing: doAction1")

	if this.state.IsEvaderHere {
		this.state.P = this.id
		this.state.Dist2Evader = int64(0)
		temp1 := time.Now().Unix()
		this.state.DetectTimestamp = temp1
		stateChanged = true
	}

	log.Debugf("doAction1: state changed: %v", stateChanged)

	return stateChanged
}

func (this *ProSe_impl_PursuerEvaderTracking) doAction2() bool {
	stateChanged := false

	log.Debugf("Executing: doAction2")

	var found bool
	var neighbor *NeighborState
	for _, neighbor = range this.neighborState {
		if (neighbor.state.DetectTimestamp > this.state.DetectTimestamp) || ((neighbor.state.DetectTimestamp == this.state.DetectTimestamp) && ((neighbor.state.Dist2Evader + int64(1)) < this.state.Dist2Evader)) {
			found = true
			break
		}
	}
	if found {
		this.state.P = neighbor.id
		this.state.Dist2Evader = (neighbor.state.Dist2Evader + int64(1))
		this.state.DetectTimestamp = neighbor.state.DetectTimestamp
		stateChanged = true
	}

	log.Debugf("doAction2: state changed: %v", stateChanged)

	return stateChanged
}

func (this *ProSe_impl_PursuerEvaderTracking) updateLocalState() bool {
	stateChanged := false

	statements := []func() bool{

		this.doAction0,

		this.doAction1,

		this.doAction2,
	}

	for _, stmtFunc := range statements {
		if changed := stmtFunc(); changed {
			stateChanged = true
		}
	}

	return stateChanged
}

func (this *ProSe_impl_PursuerEvaderTracking) broadcastLocalState() (int, error) {
	updMessage := &p.NeighborUpdate{
		Id: this.id,
		State: &p.State{

			DetectTimestamp: this.state.DetectTimestamp,
			Dist2Evader:     this.state.Dist2Evader,
			IsEvaderHere:    this.state.IsEvaderHere,
			P:               this.state.P,
		},
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_StateUpdate,
		Src:  this.id,
		Msg:  &p.BroadcastMessage_Upd{updMessage},
	}

	return this.send(broadcastMessage)
}

func (this *ProSe_impl_PursuerEvaderTracking) sendHeartBeat() (int, error) {
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

func (this *ProSe_impl_PursuerEvaderTracking) send(msg *p.BroadcastMessage) (int, error) {
	log.Debugf("Sending: %+v", msg)
	data, err := proto.Marshal(msg)
	if err != nil {
		return 0, err
	}
	return this.mcastConn.Write(data)
}

func (this *ProSe_impl_PursuerEvaderTracking) msgHandler(src *net.UDPAddr, n int, b []byte) {
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

func (this *ProSe_impl_PursuerEvaderTracking) Listener(ctx context.Context) {
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
