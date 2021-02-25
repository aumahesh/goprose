// +build RoutingTreeMaintenance

package internal

import (
	"context"
	"net"
	"time"
	"fmt"


	"math/rand"


	log "github.com/sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/dmichael/go-multicast/multicast"
	p "aumahesh.com/prose/RoutingTreeMaintenance/models"
)

const (
	inactivityTimeout = time.Duration(2) * time.Minute
	heartbeatInterval = time.Duration(1) * time.Minute
	maxDatagramSize = 1024
)

var (
	
	
	CMAX int64 = 0
	
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

type ProSe_impl_RoutingTreeMaintenance struct {
	id string
	state *p.State
	mcastAddr string
	mcastConn *net.UDPConn
	receiveChannel chan *p.NeighborUpdate
	hbChannel chan *p.NeighborHeartBeat
	neighborState map[string]*NeighborState
	configuredPriority []int
	runningPriority []int
	guardedStatements []func() bool
}

func (this *ProSe_impl_RoutingTreeMaintenance) init(id string, mcastAddr string) error {
	this.id = id
	this.state = &p.State{}
	this.mcastAddr = mcastAddr
	this.configuredPriority = []int{}
	this.runningPriority = []int{}
	this.guardedStatements = []func() bool{}

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

	CMAX = this.initConstantCMAX()
	

	this.initState()

	// set priorities for actions

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guardedStatements = append(this.guardedStatements, this.doAction0)

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guardedStatements = append(this.guardedStatements, this.doAction1)

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guardedStatements = append(this.guardedStatements, this.doAction2)

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guardedStatements = append(this.guardedStatements, this.doAction3)


	return nil
}

func (this *ProSe_impl_RoutingTreeMaintenance) initState() {
	this.state.Dist = 0
        this.state.Inv = 0
        this.state.P = ""
        

	this.state.Dist = this.initVaribleDist()
	this.state.Inv = this.initVaribleInv()
	
}

func (this *ProSe_impl_RoutingTreeMaintenance) initConstantCMAX() int64 {
	
	return int64(100)
}

func (this *ProSe_impl_RoutingTreeMaintenance) initVaribleDist() int64 {
    
	temp0 := rand.Int63n(int64(10))
    
	return temp0
}

func (this *ProSe_impl_RoutingTreeMaintenance) initVaribleInv() int64 {
    
	return int64(0)
}

func (this *ProSe_impl_RoutingTreeMaintenance) EventHandler(ctx context.Context) {
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
				
					Dist: s.State.Dist,
					Inv: s.State.Inv,
					P: s.State.P,
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

func (this *ProSe_impl_RoutingTreeMaintenance) evaluateNeighborStates() {
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

func (this *ProSe_impl_RoutingTreeMaintenance) isNeighborUp(id string) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	return nbr.active
}

func (this *ProSe_impl_RoutingTreeMaintenance) neighbors() map[string]*NeighborState {
	return this.neighborState
}

func (this *ProSe_impl_RoutingTreeMaintenance) setNeighbor(id string, state bool) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	nbr.active = state
	return nbr.active
}

func (this *ProSe_impl_RoutingTreeMaintenance) getNeighbor(id string) (*NeighborState, error) {
	nbr, ok := this.neighborState[id]
	if !ok {
		return nil, fmt.Errorf("%s not found in neighbors", id)
	}
	return nbr, nil
}

func (this *ProSe_impl_RoutingTreeMaintenance) decrementPriority(actionIndex int) {
	p := this.runningPriority[actionIndex]
	this.runningPriority[actionIndex] = p-1
}

func (this *ProSe_impl_RoutingTreeMaintenance) resetPriority(actionIndex int) {
	this.runningPriority[actionIndex] = this.configuredPriority[actionIndex]
}

func (this *ProSe_impl_RoutingTreeMaintenance) okayToRun(actionIndex int) bool {
	if this.runningPriority[actionIndex] == 0 {
		return true
	}
	return false
}


func (this *ProSe_impl_RoutingTreeMaintenance) doAction0() bool {
	stateChanged := false

	this.decrementPriority(0)
	if this.okayToRun(0) {

		log.Debugf("Executing: doAction0")

		
		var found bool
		var neighbor *NeighborState
		for _, neighbor = range this.neighborState {
			temp1 := this.isNeighborUp(neighbor.id)
			if ((neighbor.state.Dist < this.state.Dist) && (temp1 && ((neighbor.state.Inv < CMAX) && (neighbor.state.Inv < this.state.Inv)))) {
				found = true
				break
			}
		}
		if found {
			this.state.P = neighbor.id
			this.state.Inv = neighbor.state.Inv
			stateChanged = true
		}

		log.Debugf("doAction0: state changed: %v", stateChanged)

		this.resetPriority(0)
	}

	return stateChanged
}

func (this *ProSe_impl_RoutingTreeMaintenance) doAction1() bool {
	stateChanged := false

	this.decrementPriority(1)
	if this.okayToRun(1) {

		log.Debugf("Executing: doAction1")

		
		var found bool
		var neighbor *NeighborState
		for _, neighbor = range this.neighborState {
			temp2 := this.isNeighborUp(neighbor.id)
			if ((neighbor.state.Dist < this.state.Dist) && (temp2 && (((neighbor.state.Inv + int64(1)) < CMAX) && ((neighbor.state.Inv + int64(1)) < this.state.Inv)))) {
				found = true
				break
			}
		}
		if found {
			this.state.P = neighbor.id
			this.state.Inv = (neighbor.state.Inv + int64(1))
			stateChanged = true
		}

		log.Debugf("doAction1: state changed: %v", stateChanged)

		this.resetPriority(1)
	}

	return stateChanged
}

func (this *ProSe_impl_RoutingTreeMaintenance) doAction2() bool {
	stateChanged := false

	this.decrementPriority(2)
	if this.okayToRun(2) {

		log.Debugf("Executing: doAction2")

		
		var found bool
		var neighbor *NeighborState
		for _, neighbor = range this.neighborState {
			temp3 := this.isNeighborUp(this.state.P)
			if neighbor.id != this.state.P {
				continue
			}
			if ((this.state.P != "") && ((temp3 == false) || ((neighbor.state.Inv >= CMAX) || (((neighbor.state.Dist < this.state.Dist) && (this.state.Inv != neighbor.state.Inv)) || ((neighbor.state.Dist > this.state.Dist) && (this.state.Inv != (neighbor.state.Inv + int64(1)))))))) {
				found = true
				break
			}
		}
		if found {
			this.state.P = ""
			this.state.Inv = CMAX
			stateChanged = true
		}

		log.Debugf("doAction2: state changed: %v", stateChanged)

		this.resetPriority(2)
	}

	return stateChanged
}

func (this *ProSe_impl_RoutingTreeMaintenance) doAction3() bool {
	stateChanged := false

	this.decrementPriority(3)
	if this.okayToRun(3) {

		log.Debugf("Executing: doAction3")

		
		if ((this.state.P == "") && (this.state.Inv < CMAX)) {
			this.state.Inv = CMAX
			stateChanged = true
		}

		log.Debugf("doAction3: state changed: %v", stateChanged)

		this.resetPriority(3)
	}

	return stateChanged
}


func (this *ProSe_impl_RoutingTreeMaintenance) updateLocalState() bool {
	stateChanged := false

	for _, stmtFunc := range this.guardedStatements {
		if changed := stmtFunc(); changed {
			stateChanged = true
		}
	}

	return stateChanged
}

func (this *ProSe_impl_RoutingTreeMaintenance) broadcastLocalState() (int, error) {
	updMessage := &p.NeighborUpdate{
		Id: this.id,
		State: &p.State{
			
                Dist: this.state.Dist,
                Inv: this.state.Inv,
                P: this.state.P,
		},
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_StateUpdate,
		Src: this.id,
		Msg:  &p.BroadcastMessage_Upd{updMessage},
	}

	return this.send(broadcastMessage)
}

func (this *ProSe_impl_RoutingTreeMaintenance) sendHeartBeat() (int, error) {
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

func (this *ProSe_impl_RoutingTreeMaintenance) send(msg *p.BroadcastMessage) (int, error) {
	log.Debugf("Sending: %+v", msg)
	data, err := proto.Marshal(msg)
	if err != nil {
		return 0, err
	}
	return this.mcastConn.Write(data)
}

func (this *ProSe_impl_RoutingTreeMaintenance) msgHandler(src *net.UDPAddr, n int, b []byte) {
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

func (this *ProSe_impl_RoutingTreeMaintenance) Listener(ctx context.Context) {
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