// +build TreeColoring

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
	p "aumahesh.com/prose/TreeColoring/models"
)

const (
	inactivityTimeout = time.Duration(2) * time.Minute
	heartbeatInterval = time.Duration(1) * time.Minute
	maxDatagramSize = 1024
)

var (
	
	
	green int64 = 0
	
	
	red int64 = 0
	
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

type ProSe_impl_TreeColoring struct {
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

func (this *ProSe_impl_TreeColoring) init(id string, mcastAddr string) error {
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

	green = this.initConstantgreen()
	red = this.initConstantred()
	

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

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guards = append(this.guards, this.evaluateGuard4)
	this.actions = append(this.actions, this.executeAction4)


	return nil
}

func (this *ProSe_impl_TreeColoring) initState() {
	this.state.P = ""
        this.state.Color = 0
        this.state.Root = ""
        this.state.Tmp = false
        

	this.state.P = this.initVaribleP()
	this.state.Color = this.initVaribleColor()
	this.state.Root = this.initVaribleRoot()
	
}

func (this *ProSe_impl_TreeColoring) initConstantgreen() int64 {
	
	return int64(1)
}

func (this *ProSe_impl_TreeColoring) initConstantred() int64 {
	
	return int64(0)
}

func (this *ProSe_impl_TreeColoring) initVaribleP() string {
    
	return this.id
}

func (this *ProSe_impl_TreeColoring) initVaribleColor() int64 {
    
	return green
}

func (this *ProSe_impl_TreeColoring) initVaribleRoot() string {
    
	return this.id
}

func (this *ProSe_impl_TreeColoring) EventHandler(ctx context.Context) {
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
					Color: s.State.Color,
					Root: s.State.Root,
					Tmp: s.State.Tmp,
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

func (this *ProSe_impl_TreeColoring) evaluateNeighborStates() {
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

func (this *ProSe_impl_TreeColoring) isNeighborUp(id string) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	return nbr.active
}

func (this *ProSe_impl_TreeColoring) neighbors() map[string]*NeighborState {
	return this.neighborState
}

func (this *ProSe_impl_TreeColoring) setNeighbor(id string, state bool) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	nbr.active = state
	return nbr.active
}

func (this *ProSe_impl_TreeColoring) getNeighbor(id string) (*NeighborState, error) {
	nbr, ok := this.neighborState[id]
	if !ok {
		return nil, fmt.Errorf("%s not found in neighbors", id)
	}
	return nbr, nil
}

func (this *ProSe_impl_TreeColoring) decrementPriority(actionIndex int) {
	p := this.runningPriority[actionIndex]
	this.runningPriority[actionIndex] = p-1
}

func (this *ProSe_impl_TreeColoring) resetPriority(actionIndex int) {
	this.runningPriority[actionIndex] = this.configuredPriority[actionIndex]
}

func (this *ProSe_impl_TreeColoring) okayToRun(actionIndex int) bool {
	if this.runningPriority[actionIndex] == 0 {
		return true
	}
	return false
}


func (this *ProSe_impl_TreeColoring) evaluateGuard0() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(0)
	if this.okayToRun(0) {
		log.Debugf("Evaluating Guard 0")

		
		var found bool
		for _, neighbor = range this.neighborState {
			temp0 := this.isNeighborUp(this.state.P)
			if neighbor.id != this.state.P {
				continue
			}
			if ((this.state.Color == green) && ((temp0 == false) || (neighbor.state.Color == red))) {
				found = true
				break
			}
		}
		if found {
			takeAction = true
		}

		log.Debugf("Guard 0 evaluated to %v", takeAction)
		this.resetPriority(0)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_TreeColoring) executeAction0(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 0")

	
	if neighbor == nil {
		log.Errorf("invalid neighbor, nil received")
		return false, nil
	}
	this.state.Color = red

	log.Debugf("Action 0 executed")

	return true, neighbor
}

func (this *ProSe_impl_TreeColoring) evaluateGuard1() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(1)
	if this.okayToRun(1) {
		log.Debugf("Evaluating Guard 1")

		
		temp1 := this.neighbors()
		temp2 := true
		for _, neighbor := range temp1 {
			if temp2 && !(neighbor.state.P != this.id) {
				temp2 = false
				break
			}
		}
		if ((this.state.Color == red) && temp2) {
			takeAction = true
		}

		log.Debugf("Guard 1 evaluated to %v", takeAction)
		this.resetPriority(1)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_TreeColoring) executeAction1(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 1")

	
	this.state.Color = green
	this.state.P = this.id
	this.state.Root = this.id

	log.Debugf("Action 1 executed")

	return true, neighbor
}

func (this *ProSe_impl_TreeColoring) evaluateGuard2() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(2)
	if this.okayToRun(2) {
		log.Debugf("Evaluating Guard 2")

		
		var found bool
		for _, neighbor = range this.neighborState {
			if ((this.state.Root < neighbor.state.Root) && ((this.state.Color == green) && (neighbor.state.Color == green))) {
				found = true
				break
			}
		}
		if found {
			takeAction = true
		}

		log.Debugf("Guard 2 evaluated to %v", takeAction)
		this.resetPriority(2)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_TreeColoring) executeAction2(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 2")

	
	if neighbor == nil {
		log.Errorf("invalid neighbor, nil received")
		return false, nil
	}
	this.state.P = neighbor.id
	this.state.Root = neighbor.state.Root

	log.Debugf("Action 2 executed")

	return true, neighbor
}

func (this *ProSe_impl_TreeColoring) evaluateGuard3() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(3)
	if this.okayToRun(3) {
		log.Debugf("Evaluating Guard 3")

		
		temp3 := this.isNeighborUp(this.id)
		if temp3 {
			takeAction = true
		}

		log.Debugf("Guard 3 evaluated to %v", takeAction)
		this.resetPriority(3)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_TreeColoring) executeAction3(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 3")

	
	temp4 := this.setNeighbor(this.id, false)
	this.state.Tmp = temp4

	log.Debugf("Action 3 executed")

	return true, neighbor
}

func (this *ProSe_impl_TreeColoring) evaluateGuard4() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(4)
	if this.okayToRun(4) {
		log.Debugf("Evaluating Guard 4")

		
		temp5 := this.isNeighborUp(this.id)
		if (temp5 == false) {
			takeAction = true
		}

		log.Debugf("Guard 4 evaluated to %v", takeAction)
		this.resetPriority(4)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_TreeColoring) executeAction4(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 4")

	
	temp6 := this.setNeighbor(this.id, true)
	this.state.Tmp = temp6
	this.state.P = this.id
	this.state.Color = red

	log.Debugf("Action 4 executed")

	return true, neighbor
}


func (this *ProSe_impl_TreeColoring) updateLocalState() bool {
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

func (this *ProSe_impl_TreeColoring) broadcastLocalState() (int, error) {
	updMessage := &p.NeighborUpdate{
		Id: this.id,
		State: &p.State{
			
                P: this.state.P,
                Color: this.state.Color,
                Root: this.state.Root,
                Tmp: this.state.Tmp,
		},
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_StateUpdate,
		Src: this.id,
		Msg:  &p.BroadcastMessage_Upd{updMessage},
	}

	return this.send(broadcastMessage)
}

func (this *ProSe_impl_TreeColoring) sendHeartBeat() (int, error) {
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

func (this *ProSe_impl_TreeColoring) send(msg *p.BroadcastMessage) (int, error) {
	log.Debugf("Sending: %+v", msg)
	data, err := proto.Marshal(msg)
	if err != nil {
		return 0, err
	}
	return this.mcastConn.Write(data)
}

func (this *ProSe_impl_TreeColoring) msgHandler(src *net.UDPAddr, n int, b []byte) {
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

func (this *ProSe_impl_TreeColoring) Listener(ctx context.Context) {
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