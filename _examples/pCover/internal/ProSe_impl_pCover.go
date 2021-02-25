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
	guardedStatements []func() bool
}

func (this *ProSe_impl_pCover) init(id string, mcastAddr string) error {
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

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guardedStatements = append(this.guardedStatements, this.doAction4)

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guardedStatements = append(this.guardedStatements, this.doAction5)

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guardedStatements = append(this.guardedStatements, this.doAction6)


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


func (this *ProSe_impl_pCover) doAction0() bool {
	stateChanged := false

	this.decrementPriority(0)
	if this.okayToRun(0) {

		log.Debugf("Executing: doAction0")

		
		if ((this.state.St == sleep) && (this.state.Timer >= X)) {
			this.state.St = probe
			this.state.Timer = int64(0)
			stateChanged = true
		}

		log.Debugf("doAction0: state changed: %v", stateChanged)

		this.resetPriority(0)
	}

	return stateChanged
}

func (this *ProSe_impl_pCover) doAction1() bool {
	stateChanged := false

	this.decrementPriority(1)
	if this.okayToRun(1) {

		log.Debugf("Executing: doAction1")

		
		temp0 := rand.Int63n(int64(100))
		if ((this.state.St == probe) && ((this.state.Timer >= Y) && (temp0 > OnThreshold))) {
			this.state.St = sleep
			this.state.Timer = int64(0)
			stateChanged = true
		}

		log.Debugf("doAction1: state changed: %v", stateChanged)

		this.resetPriority(1)
	}

	return stateChanged
}

func (this *ProSe_impl_pCover) doAction2() bool {
	stateChanged := false

	this.decrementPriority(2)
	if this.okayToRun(2) {

		log.Debugf("Executing: doAction2")

		
		temp1 := rand.Int63n(int64(100))
		if ((this.state.St == probe) && ((this.state.Timer >= Y) && (temp1 <= OffThreshold))) {
			this.state.St = awake
			temp2 := rand.Int63n(S)
			this.state.Timer = temp2
			stateChanged = true
		}

		log.Debugf("doAction2: state changed: %v", stateChanged)

		this.resetPriority(2)
	}

	return stateChanged
}

func (this *ProSe_impl_pCover) doAction3() bool {
	stateChanged := false

	this.decrementPriority(3)
	if this.okayToRun(3) {

		log.Debugf("Executing: doAction3")

		
		if ((this.state.St == awake) && (this.state.Timer >= Z)) {
			this.state.St = readyoff
			this.state.Timer = int64(0)
			stateChanged = true
		}

		log.Debugf("doAction3: state changed: %v", stateChanged)

		this.resetPriority(3)
	}

	return stateChanged
}

func (this *ProSe_impl_pCover) doAction4() bool {
	stateChanged := false

	this.decrementPriority(4)
	if this.okayToRun(4) {

		log.Debugf("Executing: doAction4")

		
		if ((this.state.St == readyoff) && (this.state.Timer >= W)) {
			this.state.St = awake
			temp3 := rand.Int63n(S)
			this.state.Timer = temp3
			stateChanged = true
		}

		log.Debugf("doAction4: state changed: %v", stateChanged)

		this.resetPriority(4)
	}

	return stateChanged
}

func (this *ProSe_impl_pCover) doAction5() bool {
	stateChanged := false

	this.decrementPriority(5)
	if this.okayToRun(5) {

		log.Debugf("Executing: doAction5")

		
		temp4 := rand.Int63n(int64(100))
		if ((this.state.St == readyoff) && (temp4 > OffThreshold)) {
			this.state.St = sleep
			this.state.Timer = int64(0)
			stateChanged = true
		}

		log.Debugf("doAction5: state changed: %v", stateChanged)

		this.resetPriority(5)
	}

	return stateChanged
}

func (this *ProSe_impl_pCover) doAction6() bool {
	stateChanged := false

	this.decrementPriority(6)
	if this.okayToRun(6) {

		log.Debugf("Executing: doAction6")

		
		if (((this.state.St == sleep) && (this.state.Timer <= X)) || (((this.state.St == probe) && (this.state.Timer <= Y)) || (((this.state.St == awake) && (this.state.Timer <= Z)) || ((this.state.St == readyoff) && (this.state.Timer <= W))))) {
			this.state.Timer = (this.state.Timer + int64(1))
			stateChanged = true
		}

		log.Debugf("doAction6: state changed: %v", stateChanged)

		this.resetPriority(6)
	}

	return stateChanged
}


func (this *ProSe_impl_pCover) updateLocalState() bool {
	stateChanged := false

	for _, stmtFunc := range this.guardedStatements {
		if changed := stmtFunc(); changed {
			stateChanged = true
		}
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