// +build pCover

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
	p "aumahesh.com/prose/pCover/models"
)

const (
	inactivityTimeout = time.Duration(2) * time.Minute
	heartbeatInterval = time.Duration(1) * time.Minute
	updateLocalStateInterval = time.Duration(10) * time.Second
	maxDatagramSize = 1024
)

var (
	
	
	OffThreshold int64 = 0
	
	
	OnThreshold int64 = 0
	
	
	S int64 = 0
	
	
	W int64 = 0
	
	
	X int64 = 0
	
	
	Y int64 = 0
	
	
	Z int64 = 0
	
	
	awake int64 = 0
	
	
	probe int64 = 0
	
	
	readyoff int64 = 0
	
	
	sleep int64 = 0
	
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

type ProSe_impl_pCover struct {
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

func (this *ProSe_impl_pCover) init(id string, mcastAddr string) error {
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

	this.neighborState = map[string]*NeighborState{}

	OffThreshold = this.initConstantOffThreshold()
	OnThreshold = this.initConstantOnThreshold()
	S = this.initConstantS()
	W = this.initConstantW()
	X = this.initConstantX()
	Y = this.initConstantY()
	Z = this.initConstantZ()
	awake = this.initConstantawake()
	probe = this.initConstantprobe()
	readyoff = this.initConstantreadyoff()
	sleep = this.initConstantsleep()
	

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

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guards = append(this.guards, this.evaluateGuard5)
	this.actions = append(this.actions, this.executeAction5)

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guards = append(this.guards, this.evaluateGuard6)
	this.actions = append(this.actions, this.executeAction6)


	return nil
}

func (this *ProSe_impl_pCover) initState() {
	this.state.St = 0
        this.state.Timer = 0
        

	
}

func (this *ProSe_impl_pCover) initConstantOffThreshold() int64 {
	
	return int64(200)
}

func (this *ProSe_impl_pCover) initConstantOnThreshold() int64 {
	
	return int64(100)
}

func (this *ProSe_impl_pCover) initConstantS() int64 {
	
	return int64(40)
}

func (this *ProSe_impl_pCover) initConstantW() int64 {
	
	return int64(50)
}

func (this *ProSe_impl_pCover) initConstantX() int64 {
	
	return int64(10)
}

func (this *ProSe_impl_pCover) initConstantY() int64 {
	
	return int64(20)
}

func (this *ProSe_impl_pCover) initConstantZ() int64 {
	
	return int64(30)
}

func (this *ProSe_impl_pCover) initConstantawake() int64 {
	
	return int64(3)
}

func (this *ProSe_impl_pCover) initConstantprobe() int64 {
	
	return int64(2)
}

func (this *ProSe_impl_pCover) initConstantreadyoff() int64 {
	
	return int64(4)
}

func (this *ProSe_impl_pCover) initConstantsleep() int64 {
	
	return int64(1)
}

func (this *ProSe_impl_pCover) EventHandler(ctx context.Context) {
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	updTicker := time.NewTicker(updateLocalStateInterval)
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
				
					St: s.State.St,
					Timer: s.State.Timer,
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
		case <-updTicker.C:
			if stateChanged := this.updateLocalState(); stateChanged {
				n, err := this.broadcastLocalState()
				if err != nil {
					log.Errorf("Error broadcasting local state to neighbors")
				} else {
					log.Debugf("%s: sent heartbeat: %d bytes", this.id, n)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (this *ProSe_impl_pCover) evaluateNeighborStates() {
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

func (this *ProSe_impl_pCover) isNeighborUp(id string) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	return nbr.active
}

func (this *ProSe_impl_pCover) neighbors() map[string]*NeighborState {
	return this.neighborState
}

func (this *ProSe_impl_pCover) setNeighbor(id string, state bool) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	nbr.active = state
	return nbr.active
}

func (this *ProSe_impl_pCover) getNeighbor(id string) (*NeighborState, error) {
	if id == this.id {
		return &NeighborState{
					id: this.id,
					state: this.state,
					discoveredAt: time.Now(),
					updatedAt: time.Now(),
					lastHeartBeatAt: time.Now(),
					stateChangedAt: time.Now(),
					active: true,
				}, nil
	}
	nbr, ok := this.neighborState[id]
	if !ok {
		return nil, fmt.Errorf("%s not found in neighbors", id)
	}
	return nbr, nil
}

func (this *ProSe_impl_pCover) decrementPriority(actionIndex int) {
	p := this.runningPriority[actionIndex]
	this.runningPriority[actionIndex] = p-1
}

func (this *ProSe_impl_pCover) resetPriority(actionIndex int) {
	this.runningPriority[actionIndex] = this.configuredPriority[actionIndex]
}

func (this *ProSe_impl_pCover) okayToRun(actionIndex int) bool {
	if this.runningPriority[actionIndex] == 0 {
		return true
	}
	return false
}


func (this *ProSe_impl_pCover) evaluateGuard0() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(0)
	if this.okayToRun(0) {
		log.Debugf("Evaluating Guard 0")

		
		if ((this.state.St == sleep) && (this.state.Timer >= X)) {
			takeAction = true
		}

		log.Debugf("Guard 0 evaluated to %v", takeAction)
		this.resetPriority(0)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_pCover) executeAction0(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 0")

	
	this.state.St = probe
	this.state.Timer = int64(0)

	log.Debugf("Action 0 executed")

	return true, neighbor
}

func (this *ProSe_impl_pCover) evaluateGuard1() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(1)
	if this.okayToRun(1) {
		log.Debugf("Evaluating Guard 1")

		
		temp0 := rand.Int63n(int64(100))
		if ((this.state.St == probe) && ((this.state.Timer >= Y) && (temp0 > OnThreshold))) {
			takeAction = true
		}

		log.Debugf("Guard 1 evaluated to %v", takeAction)
		this.resetPriority(1)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_pCover) executeAction1(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 1")

	
	this.state.St = sleep
	this.state.Timer = int64(0)

	log.Debugf("Action 1 executed")

	return true, neighbor
}

func (this *ProSe_impl_pCover) evaluateGuard2() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(2)
	if this.okayToRun(2) {
		log.Debugf("Evaluating Guard 2")

		
		temp1 := rand.Int63n(int64(100))
		if ((this.state.St == probe) && ((this.state.Timer >= Y) && (temp1 <= OffThreshold))) {
			takeAction = true
		}

		log.Debugf("Guard 2 evaluated to %v", takeAction)
		this.resetPriority(2)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_pCover) executeAction2(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 2")

	
	this.state.St = awake
	temp2 := rand.Int63n(S)
	this.state.Timer = temp2

	log.Debugf("Action 2 executed")

	return true, neighbor
}

func (this *ProSe_impl_pCover) evaluateGuard3() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(3)
	if this.okayToRun(3) {
		log.Debugf("Evaluating Guard 3")

		
		if ((this.state.St == awake) && (this.state.Timer >= Z)) {
			takeAction = true
		}

		log.Debugf("Guard 3 evaluated to %v", takeAction)
		this.resetPriority(3)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_pCover) executeAction3(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 3")

	
	this.state.St = readyoff
	this.state.Timer = int64(0)

	log.Debugf("Action 3 executed")

	return true, neighbor
}

func (this *ProSe_impl_pCover) evaluateGuard4() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(4)
	if this.okayToRun(4) {
		log.Debugf("Evaluating Guard 4")

		
		if ((this.state.St == readyoff) && (this.state.Timer >= W)) {
			takeAction = true
		}

		log.Debugf("Guard 4 evaluated to %v", takeAction)
		this.resetPriority(4)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_pCover) executeAction4(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 4")

	
	this.state.St = awake
	temp3 := rand.Int63n(S)
	this.state.Timer = temp3

	log.Debugf("Action 4 executed")

	return true, neighbor
}

func (this *ProSe_impl_pCover) evaluateGuard5() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(5)
	if this.okayToRun(5) {
		log.Debugf("Evaluating Guard 5")

		
		temp4 := rand.Int63n(int64(100))
		if ((this.state.St == readyoff) && (temp4 > OffThreshold)) {
			takeAction = true
		}

		log.Debugf("Guard 5 evaluated to %v", takeAction)
		this.resetPriority(5)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_pCover) executeAction5(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 5")

	
	this.state.St = sleep
	this.state.Timer = int64(0)

	log.Debugf("Action 5 executed")

	return true, neighbor
}

func (this *ProSe_impl_pCover) evaluateGuard6() (bool, *NeighborState) {
	var takeAction bool
	var neighbor *NeighborState

	takeAction = false
	neighbor = nil

	this.decrementPriority(6)
	if this.okayToRun(6) {
		log.Debugf("Evaluating Guard 6")

		
		if (((this.state.St == sleep) && (this.state.Timer <= X)) || (((this.state.St == probe) && (this.state.Timer <= Y)) || (((this.state.St == awake) && (this.state.Timer <= Z)) || ((this.state.St == readyoff) && (this.state.Timer <= W))))) {
			takeAction = true
		}

		log.Debugf("Guard 6 evaluated to %v", takeAction)
		this.resetPriority(6)
	}

	return takeAction, neighbor
}

func (this *ProSe_impl_pCover) executeAction6(neighbor *NeighborState) (bool, *NeighborState) {
	log.Debugf("Executing Action 6")

	
	this.state.Timer = (this.state.Timer + int64(1))

	log.Debugf("Action 6 executed")

	return true, neighbor
}


func (this *ProSe_impl_pCover) updateLocalState() bool {
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

func (this *ProSe_impl_pCover) broadcastLocalState() (int, error) {
	updMessage := &p.NeighborUpdate{
		Id: this.id,
		State: &p.State{
			
                St: this.state.St,
                Timer: this.state.Timer,
		},
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_StateUpdate,
		Src: this.id,
		Msg:  &p.BroadcastMessage_Upd{updMessage},
	}

	return this.send(broadcastMessage)
}

func (this *ProSe_impl_pCover) sendHeartBeat() (int, error) {
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

func (this *ProSe_impl_pCover) send(msg *p.BroadcastMessage) (int, error) {
	log.Debugf("Sending: %+v", msg)
	data, err := proto.Marshal(msg)
	if err != nil {
		return 0, err
	}
	return this.mcastConn.Write(data)
}

func (this *ProSe_impl_pCover) msgHandler(src *net.UDPAddr, n int, b []byte) {
	log.Debugf("received message (%d bytes) from %s", n, src.String())
	broadcastMessage := &p.BroadcastMessage{}
	err := proto.Unmarshal(b[:n], broadcastMessage)
	if err != nil {
		log.Errorf("error unmarshalling proto from src %s: %s", src.String(), err)
		return
	}
	if broadcastMessage.Src == this.id {
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

func (this *ProSe_impl_pCover) Listener(ctx context.Context) {
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