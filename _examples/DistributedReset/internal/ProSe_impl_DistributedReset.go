// +build DistributedReset

package internal

import (
	"context"
	"net"
	"time"
	"fmt"



	log "github.com/sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/dmichael/go-multicast/multicast"
	p "aumahesh.com/prose/DistributedReset/models"
)

const (
	inactivityTimeout = time.Duration(2) * time.Minute
	heartbeatInterval = time.Duration(1) * time.Minute
	maxDatagramSize = 1024
)

var (
	
	
	initiate int64 = 0
	
	
	normal int64 = 0
	
	
	reset int64 = 0
	
)

type NeighborState struct {
	id string
	state *p.State
	discoveredAt time.Time
	updatedAt time.Time
	lastHeartBeatAt time.Time
	stateChangedAt time.Time
	active bool
}

type ProSe_impl_DistributedReset struct {
	id string
	state *p.State
	mcastAddr string
	mcastConn *net.UDPConn
	receiveChannel chan *p.NeighborUpdate
	hbChannel chan *p.NeighborHeartBeat
	neighborState map[string]*NeighborState
}

func (this *ProSe_impl_DistributedReset) init(id string, mcastAddr string) error {
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
					id: this.id,
					state: this.state,
					discoveredAt: time.Now(),
					updatedAt: time.Now(),
					lastHeartBeatAt: time.Now(),
					stateChangedAt: time.Now(),
					active: true,
				},
	}

	initiate = this.initConstantinitiate()
	normal = this.initConstantnormal()
	reset = this.initConstantreset()
	

	this.initState()

	return nil
}

func (this *ProSe_impl_DistributedReset) initState() {
	this.state.P = ""
        this.state.Sn = 0
        this.state.St = 0
        

	this.state.P = this.initVaribleP()
	this.state.Sn = this.initVaribleSn()
	this.state.St = this.initVaribleSt()
	
}

func (this *ProSe_impl_DistributedReset) initConstantinitiate() int64 {
	
	return int64(1)
}

func (this *ProSe_impl_DistributedReset) initConstantnormal() int64 {
	
	return int64(3)
}

func (this *ProSe_impl_DistributedReset) initConstantreset() int64 {
	
	return int64(2)
}

func (this *ProSe_impl_DistributedReset) initVaribleP() string {
    
	return this.id
}

func (this *ProSe_impl_DistributedReset) initVaribleSn() int64 {
    
	return int64(1)
}

func (this *ProSe_impl_DistributedReset) initVaribleSt() int64 {
    
	return initiate
}

func (this *ProSe_impl_DistributedReset) EventHandler(ctx context.Context) {
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	for {
		select {
		case s := <-this.receiveChannel:
			_, ok := this.neighborState[s.Id]
			if !ok {
				this.neighborState[s.Id] = &NeighborState{
					id: s.Id,
					discoveredAt: time.Now(),
					active: true,
					stateChangedAt: time.Now(),
				}
			}
			this.neighborState[s.Id].state = &p.State{
				
					P: s.State.P,
					Sn: s.State.Sn,
					St: s.State.St,
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
					id: s.Id,
					state: &p.State{},
					discoveredAt: time.Now(),
					active: true,
					stateChangedAt: time.Now(),
					updatedAt: time.Now(),
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

func (this *ProSe_impl_DistributedReset) evaluateNeighborStates() {
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

func (this *ProSe_impl_DistributedReset) isNeighborUp(id string) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	return nbr.active
}

func (this *ProSe_impl_DistributedReset) neighbors() map[string]*NeighborState {
	return this.neighborState
}

func (this *ProSe_impl_DistributedReset) setNeighbor(id string, state bool) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	nbr.active = state
	return nbr.active
}

func (this *ProSe_impl_DistributedReset) getNeighbor(id string, stateVariable string) (*NeighborState, error) {
	nbr, ok := this.neighborState[id]
	if !ok {
		return nil, fmt.Errorf("%s not found in neighbors", id)
	}
	return nbr, nil
}


func (this *ProSe_impl_DistributedReset) doAction0() bool {
	stateChanged := false

	log.Debugf("Executing: doAction0")

	
	if ((this.state.St == initiate) && (this.state.P == this.id)) {
		this.state.St = reset
		this.state.Sn = (this.state.Sn + int64(1))
		stateChanged = true
	}

	log.Debugf("doAction0: state changed: %v", stateChanged)

	return stateChanged
}

func (this *ProSe_impl_DistributedReset) doAction1() bool {
	stateChanged := false

	log.Debugf("Executing: doAction1")

	
	var found bool
	var neighbor *NeighborState
	for _, neighbor = range this.neighborState {
		if ((this.state.St != reset) && ((this.state.P == neighbor.id) && ((neighbor.state.St == reset) && ((this.state.Sn + int64(1)) == neighbor.state.Sn)))) {
			found = true
			break
		}
	}
	if found {
		this.state.St = reset
		this.state.Sn = neighbor.state.Sn
		stateChanged = true
	}

	log.Debugf("doAction1: state changed: %v", stateChanged)

	return stateChanged
}

func (this *ProSe_impl_DistributedReset) doAction2() bool {
	stateChanged := false

	log.Debugf("Executing: doAction2")

	
	temp0 := this.neighbors()
	temp2 := true
	for _, neighbor := range temp0 {
		temp1 := !(neighbor.state.P == this.id)
		if temp2 && !(temp1 || ((neighbor.state.St != reset) && (this.state.Sn == neighbor.state.Sn))) {
			temp2 = false
			break
		}
	}
	if ((this.state.St == reset) && temp2) {
		this.state.St = normal
		stateChanged = true
	}

	log.Debugf("doAction2: state changed: %v", stateChanged)

	return stateChanged
}

func (this *ProSe_impl_DistributedReset) doAction3() bool {
	stateChanged := false

	log.Debugf("Executing: doAction3")

	
	var found bool
	var neighbor *NeighborState
	for _, neighbor = range this.neighborState {
		temp3 := !((this.state.P == neighbor.id) && (neighbor.state.St != reset))
		temp4 := !((this.state.P == neighbor.id) && (neighbor.state.St == reset))
		if (! ((temp3 || ((this.state.St != reset) && (neighbor.state.Sn == this.state.Sn))) && (temp4 || (((this.state.St != reset) && (neighbor.state.Sn == (this.state.Sn + int64(1)))) || (neighbor.state.Sn == this.state.Sn))))) {
			found = true
			break
		}
	}
	if found {
		this.state.St = neighbor.state.St
		this.state.Sn = neighbor.state.Sn
		stateChanged = true
	}

	log.Debugf("doAction3: state changed: %v", stateChanged)

	return stateChanged
}


func (this *ProSe_impl_DistributedReset) updateLocalState() bool {
	stateChanged := false

	statements := []func() bool{

		this.doAction0,

		this.doAction1,

		this.doAction2,

		this.doAction3,

	}

	for _, stmtFunc := range statements {
		if changed := stmtFunc(); changed {
			stateChanged = true
		}
	}

	return stateChanged
}

func (this *ProSe_impl_DistributedReset) broadcastLocalState() (int, error) {
	updMessage := &p.NeighborUpdate{
		Id: this.id,
		State: &p.State{
			
                P: this.state.P,
                Sn: this.state.Sn,
                St: this.state.St,
		},
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_StateUpdate,
		Src: this.id,
		Msg:  &p.BroadcastMessage_Upd{updMessage},
	}

	return this.send(broadcastMessage)
}

func (this *ProSe_impl_DistributedReset) sendHeartBeat() (int, error) {
	hbMessage := &p.NeighborHeartBeat{
		Id: this.id,
		SentAt: time.Now().Unix(),
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_Heartbeat,
		Src: this.id,
		Msg:  &p.BroadcastMessage_Hb{hbMessage},
	}

	return this.send(broadcastMessage)
}

func (this *ProSe_impl_DistributedReset) send(msg *p.BroadcastMessage) (int, error) {
	log.Debugf("Sending: %+v", msg)
	data, err := proto.Marshal(msg)
	if err != nil {
		return 0, err
	}
	return this.mcastConn.Write(data)
}

func (this *ProSe_impl_DistributedReset) msgHandler(src *net.UDPAddr, n int, b []byte) {
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

func (this *ProSe_impl_DistributedReset) Listener(ctx context.Context) {
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