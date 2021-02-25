// +build distanceVector

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
	p "aumahesh.com/prose/distanceVector/models"
)

const (
	inactivityTimeout = time.Duration(2) * time.Minute
	heartbeatInterval = time.Duration(1) * time.Minute
	maxDatagramSize = 1024
)

var (
	
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

type ProSe_impl_distanceVector struct {
	id string
	state *p.State
	mcastAddr string
	mcastConn *net.UDPConn
	receiveChannel chan *p.NeighborUpdate
	hbChannel chan *p.NeighborHeartBeat
	neighborState map[string]*NeighborState
	configuredPriority []int
	runningPriority []int
	guards []func() (bool, *NeighborState)
	actions []func(*NeighborState) (bool, *NeighborState)
}

func (this *ProSe_impl_distanceVector) init(id string, mcastAddr string) error {
	this.id = id
	this.state = &p.State{}
	this.mcastAddr = mcastAddr
	this.configuredPriority = []int{}
	this.runningPriority = []int{}
	this.guards = []func() (bool, *NeighborState){}
	this.actions = []func(state *NeighborState) (bool, *NeighborState){}

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

	

	this.initState()

	// set priorities for actions

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guards = append(this.guards, this.evaluateGuard0)
	this.actions = append(this.actions, this.executeAction0)

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guards = append(this.guards, this.evaluateGuard1)
	this.actions = append(this.actions, this.executeAction1)

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guards = append(this.guards, this.evaluateGuard2)
	this.actions = append(this.actions, this.executeAction2)

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guards = append(this.guards, this.evaluateGuard3)
	this.actions = append(this.actions, this.executeAction3)


	return nil
}

func (this *ProSe_impl_distanceVector) initState() {
	this.state.Diameter = 0
        this.state.Dis = 0
        this.state.P = ""
        this.state.Up = false
        

	
}

func (this *ProSe_impl_distanceVector) EventHandler(ctx context.Context) {
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
				
					Diameter: s.State.Diameter,
					Dis: s.State.Dis,
					P: s.State.P,
					Up: s.State.Up,
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

func (this *ProSe_impl_distanceVector) evaluateNeighborStates() {
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

func (this *ProSe_impl_distanceVector) isNeighborUp(id string) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	return nbr.active
}

func (this *ProSe_impl_distanceVector) neighbors() map[string]*NeighborState {
	return this.neighborState
}

func (this *ProSe_impl_distanceVector) setNeighbor(id string, state bool) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	nbr.active = state
	return nbr.active
}

func (this *ProSe_impl_distanceVector) getNeighbor(id string) (*NeighborState, error) {
	nbr, ok := this.neighborState[id]
	if !ok {
		return nil, fmt.Errorf("%s not found in neighbors", id)
	}
	return nbr, nil
}

func (this *ProSe_impl_distanceVector) decrementPriority(actionIndex int) {
	p := this.runningPriority[actionIndex]
	this.runningPriority[actionIndex] = p-1
}

func (this *ProSe_impl_distanceVector) resetPriority(actionIndex int) {
	this.runningPriority[actionIndex] = this.configuredPriority[actionIndex]
}

func (this *ProSe_impl_distanceVector) okayToRun(actionIndex int) bool {
	if this.runningPriority[actionIndex] == 0 {
		return true
	}
	return false
}


func (this *ProSe_impl_distanceVector) evaluateGuard0() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(0)
	if this.okayToRun(0) {
		log.Debugf("Evaluating Guard 0")

		
		for _, neighbor = range this.neighborState {
			temp0 := this.isNeighborUp(this.state.P)
			if neighbor.id != this.state.P {
				continue
			}
			if ((this.state.P != "") && (temp0 && ((this.state.Dis != (neighbor.state.Dis + int64(1))) && (this.state.Diameter > (neighbor.state.Dis + int64(1)))))) {
				takeAction = true
				break
			}
		}

		log.Debugf("Guard 0 evaluated to %v", takeAction)
		this.resetPriority(0)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_distanceVector) executeAction0(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 0")

	
	if neighbor == nil {
		log.Errorf("invalid neighbor, nil received")
		return false, nil
	}
	this.state.Dis = (neighbor.state.Dis + int64(1))

	log.Debugf("Action 0 executed")

	return true, neighbor
}

func (this *ProSe_impl_distanceVector) evaluateGuard1() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(1)
	if this.okayToRun(1) {
		log.Debugf("Evaluating Guard 1")

		
		for _, neighbor = range this.neighborState {
			if neighbor.id != this.state.P {
				continue
			}
			if ((this.state.Dis != (neighbor.state.Dis + int64(1))) && (this.state.Diameter <= (neighbor.state.Dis + int64(1)))) {
				takeAction = true
				break
			}
		}

		log.Debugf("Guard 1 evaluated to %v", takeAction)
		this.resetPriority(1)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_distanceVector) executeAction1(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 1")

	
	if neighbor == nil {
		log.Errorf("invalid neighbor, nil received")
		return false, nil
	}
	this.state.Dis = this.state.Diameter

	log.Debugf("Action 1 executed")

	return true, neighbor
}

func (this *ProSe_impl_distanceVector) evaluateGuard2() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(2)
	if this.okayToRun(2) {
		log.Debugf("Evaluating Guard 2")

		
		temp1 := this.isNeighborUp(this.state.P)
		if ((this.state.P != "") && (! temp1)) {
			takeAction = true
		}

		log.Debugf("Guard 2 evaluated to %v", takeAction)
		this.resetPriority(2)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_distanceVector) executeAction2(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 2")

	
	this.state.P = ""
	this.state.Dis = this.state.Diameter

	log.Debugf("Action 2 executed")

	return true, neighbor
}

func (this *ProSe_impl_distanceVector) evaluateGuard3() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(3)
	if this.okayToRun(3) {
		log.Debugf("Evaluating Guard 3")

		
		for _, neighbor = range this.neighborState {
			temp2 := this.isNeighborUp(neighbor.id)
			if (temp2 && (this.state.Dis > neighbor.state.Dis)) {
				takeAction = true
				break
			}
		}

		log.Debugf("Guard 3 evaluated to %v", takeAction)
		this.resetPriority(3)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_distanceVector) executeAction3(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 3")

	
	if neighbor == nil {
		log.Errorf("invalid neighbor, nil received")
		return false, nil
	}
	this.state.P = neighbor.id
	this.state.Dis = (neighbor.state.Dis + int64(1))

	log.Debugf("Action 3 executed")

	return true, neighbor
}


func (this *ProSe_impl_distanceVector) updateLocalState() bool {
	stateChanged := false

	couldExecute := []int{}
	returnNeighbors := []*NeighborState{}

	// Find all guards that light up
	for index, stmtFunc := range this.guards {
		if okayToExecute, nbr := stmtFunc(); okayToExecute {
			couldExecute = append(couldExecute, index)
			returnNeighbors = append(returnNeighbors, nbr)
		}
	}

	// of all the guards that lighted out, pick a random guard and execute its action
	if len(couldExecute) > 0 {
		actionIndex := rand.Intn(len(couldExecute))
		this.actions[couldExecute[actionIndex]](returnNeighbors[actionIndex])
		stateChanged = true
	}

	return stateChanged
}

func (this *ProSe_impl_distanceVector) broadcastLocalState() (int, error) {
	updMessage := &p.NeighborUpdate{
		Id: this.id,
		State: &p.State{
			
                Diameter: this.state.Diameter,
                Dis: this.state.Dis,
                P: this.state.P,
                Up: this.state.Up,
		},
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_StateUpdate,
		Src: this.id,
		Msg:  &p.BroadcastMessage_Upd{updMessage},
	}

	return this.send(broadcastMessage)
}

func (this *ProSe_impl_distanceVector) sendHeartBeat() (int, error) {
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

func (this *ProSe_impl_distanceVector) send(msg *p.BroadcastMessage) (int, error) {
	log.Debugf("Sending: %+v", msg)
	data, err := proto.Marshal(msg)
	if err != nil {
		return 0, err
	}
	return this.mcastConn.Write(data)
}

func (this *ProSe_impl_distanceVector) msgHandler(src *net.UDPAddr, n int, b []byte) {
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

func (this *ProSe_impl_distanceVector) Listener(ctx context.Context) {
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