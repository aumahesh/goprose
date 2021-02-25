// +build TrackingPriority

package internal

import (
	"context"
	"net"
	"time"
	"fmt"



	log "github.com/sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/dmichael/go-multicast/multicast"
	p "aumahesh.com/prose/TrackingPriority/models"
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

type ProSe_impl_TrackingPriority struct {
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

func (this *ProSe_impl_TrackingPriority) init(id string, mcastAddr string) error {
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

	

	this.initState()

	// set priorities for actions

	this.configuredPriority = append(this.configuredPriority, 2)
	this.runningPriority = append(this.runningPriority, 2)
	this.guardedStatements = append(this.guardedStatements, this.doAction0)

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guardedStatements = append(this.guardedStatements, this.doAction1)

	this.configuredPriority = append(this.configuredPriority, 1)
	this.runningPriority = append(this.runningPriority, 1)
	this.guardedStatements = append(this.guardedStatements, this.doAction2)


	return nil
}

func (this *ProSe_impl_TrackingPriority) initState() {
	this.state.Dist2Evader = 0
        this.state.IsEvaderHere = false
        this.state.P = ""
        this.state.TimeStampOfDetection = 0
        

	
}

func (this *ProSe_impl_TrackingPriority) EventHandler(ctx context.Context) {
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
				
					Dist2Evader: s.State.Dist2Evader,
					IsEvaderHere: s.State.IsEvaderHere,
					P: s.State.P,
					TimeStampOfDetection: s.State.TimeStampOfDetection,
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

func (this *ProSe_impl_TrackingPriority) evaluateNeighborStates() {
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

func (this *ProSe_impl_TrackingPriority) isNeighborUp(id string) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	return nbr.active
}

func (this *ProSe_impl_TrackingPriority) neighbors() map[string]*NeighborState {
	return this.neighborState
}

func (this *ProSe_impl_TrackingPriority) setNeighbor(id string, state bool) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	nbr.active = state
	return nbr.active
}

func (this *ProSe_impl_TrackingPriority) getNeighbor(id string) (*NeighborState, error) {
	nbr, ok := this.neighborState[id]
	if !ok {
		return nil, fmt.Errorf("%s not found in neighbors", id)
	}
	return nbr, nil
}

func (this *ProSe_impl_TrackingPriority) decrementPriority(actionIndex int) {
	p := this.runningPriority[actionIndex]
	this.runningPriority[actionIndex] = p-1
}

func (this *ProSe_impl_TrackingPriority) resetPriority(actionIndex int) {
	this.runningPriority[actionIndex] = this.configuredPriority[actionIndex]
}

func (this *ProSe_impl_TrackingPriority) okayToRun(actionIndex int) bool {
	if this.runningPriority[actionIndex] == 0 {
		return true
	}
	return false
}


func (this *ProSe_impl_TrackingPriority) doAction0() bool {
	stateChanged := false

	this.decrementPriority(0)
	if this.okayToRun(0) {

		log.Debugf("Executing: doAction0")

		
		if this.state.IsEvaderHere {
			this.state.P = this.id
			this.state.Dist2Evader = int64(0)
			temp0 := time.Now().Unix()
			this.state.TimeStampOfDetection = temp0
			stateChanged = true
		}

		log.Debugf("doAction0: state changed: %v", stateChanged)

		this.resetPriority(0)
	}

	return stateChanged
}

func (this *ProSe_impl_TrackingPriority) doAction1() bool {
	stateChanged := false

	this.decrementPriority(1)
	if this.okayToRun(1) {

		log.Debugf("Executing: doAction1")

		
		var found bool
		var neighbor *NeighborState
		for _, neighbor = range this.neighborState {
			if (neighbor.state.TimeStampOfDetection > this.state.TimeStampOfDetection) {
				found = true
				break
			}
		}
		if found {
			this.state.P = neighbor.id
			this.state.TimeStampOfDetection = neighbor.state.TimeStampOfDetection
			this.state.Dist2Evader = (neighbor.state.Dist2Evader + int64(1))
			stateChanged = true
		}

		log.Debugf("doAction1: state changed: %v", stateChanged)

		this.resetPriority(1)
	}

	return stateChanged
}

func (this *ProSe_impl_TrackingPriority) doAction2() bool {
	stateChanged := false

	this.decrementPriority(2)
	if this.okayToRun(2) {

		log.Debugf("Executing: doAction2")

		
		var found bool
		var neighbor *NeighborState
		for _, neighbor = range this.neighborState {
			if ((neighbor.state.TimeStampOfDetection == this.state.TimeStampOfDetection) && ((neighbor.state.Dist2Evader + int64(1)) < this.state.Dist2Evader)) {
				found = true
				break
			}
		}
		if found {
			this.state.P = neighbor.id
			this.state.TimeStampOfDetection = neighbor.state.TimeStampOfDetection
			this.state.Dist2Evader = (neighbor.state.Dist2Evader + int64(1))
			stateChanged = true
		}

		log.Debugf("doAction2: state changed: %v", stateChanged)

		this.resetPriority(2)
	}

	return stateChanged
}


func (this *ProSe_impl_TrackingPriority) updateLocalState() bool {
	stateChanged := false

	for _, stmtFunc := range this.guardedStatements {
		if changed := stmtFunc(); changed {
			stateChanged = true
		}
	}

	return stateChanged
}

func (this *ProSe_impl_TrackingPriority) broadcastLocalState() (int, error) {
	updMessage := &p.NeighborUpdate{
		Id: this.id,
		State: &p.State{
			
                Dist2Evader: this.state.Dist2Evader,
                IsEvaderHere: this.state.IsEvaderHere,
                P: this.state.P,
                TimeStampOfDetection: this.state.TimeStampOfDetection,
		},
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_StateUpdate,
		Src: this.id,
		Msg:  &p.BroadcastMessage_Upd{updMessage},
	}

	return this.send(broadcastMessage)
}

func (this *ProSe_impl_TrackingPriority) sendHeartBeat() (int, error) {
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

func (this *ProSe_impl_TrackingPriority) send(msg *p.BroadcastMessage) (int, error) {
	log.Debugf("Sending: %+v", msg)
	data, err := proto.Marshal(msg)
	if err != nil {
		return 0, err
	}
	return this.mcastConn.Write(data)
}

func (this *ProSe_impl_TrackingPriority) msgHandler(src *net.UDPAddr, n int, b []byte) {
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

func (this *ProSe_impl_TrackingPriority) Listener(ctx context.Context) {
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