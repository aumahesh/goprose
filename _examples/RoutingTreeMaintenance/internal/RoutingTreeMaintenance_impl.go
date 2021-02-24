// +build RoutingTreeMaintenance

package internal

import (
	"context"
	"net"
	"time"

	"math/rand"

	p "aumahesh.com/prose/RoutingTreeMaintenance/models"
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
	CMAX int64 = 0

	NULL int64 = 0
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

type RoutingTreeMaintenance_impl struct {
	id             string
	state          *p.State
	mcastAddr      string
	mcastConn      *net.UDPConn
	receiveChannel chan *p.NeighborUpdate
	hbChannel      chan *p.NeighborHeartBeat
	neighborState  map[string]*NeighborState
}

func (this *RoutingTreeMaintenance_impl) init(id string, mcastAddr string) error {
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

	CMAX = this.initConstantCMAX()
	NULL = this.initConstantNULL()

	this.initState()

	return nil
}

func (this *RoutingTreeMaintenance_impl) initState() {
	this.state.Dist = 0
	this.state.Inv = 0
	this.state.P = ""

	this.state.Dist = this.initVaribleDist()
	this.state.Inv = this.initVaribleInv()

}

func (this *RoutingTreeMaintenance_impl) initConstantCMAX() int64 {

	temp0 := int64(100)

	return temp0
}

func (this *RoutingTreeMaintenance_impl) initConstantNULL() int64 {

	temp1 := int64(0)

	return temp1
}

func (this *RoutingTreeMaintenance_impl) initVaribleDist() int64 {

	temp3 := int64(10)

	temp4 := rand.Int63n(temp3)

	return temp4
}

func (this *RoutingTreeMaintenance_impl) initVaribleInv() int64 {

	temp2 := int64(0)

	return temp2
}

func (this *RoutingTreeMaintenance_impl) EventHandler(ctx context.Context) {
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

				Dist: s.State.Dist,
				Inv:  s.State.Inv,
				P:    s.State.P,
			}
			this.neighborState[s.Id].updatedAt = time.Now()
			this.evaluateNeighborStates()
			this.updateLocalState()
			if stateChanged := this.updateLocalState(); stateChanged {
				this.broadcastLocalState()
				if err != nil {
					log.Errorf("Error broadcasting local state to neighbors")
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
				this.broadcastLocalState()
				if err != nil {
					log.Errorf("Error broadcasting local state to neighbors")
				}
			}
		case <-heartbeatTicker.C:
			this.sendHeartBeat()
		case <-ctx.Done():
			return
		}
	}
}

func (this *RoutingTreeMaintenance_impl) evaluateNeighborStates() {
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

func (this *RoutingTreeMaintenance_impl) isNeighborUp(id string) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	return nbr.active
}

func (this *RoutingTreeMaintenance_impl) neighbors() map[string]*NeighborState {
	return this.neighborState
}

func (this *RoutingTreeMaintenance_impl) setNeighbor(id string, state bool) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	nbr.active = state
	return nbr.active
}

func (this *RoutingTreeMaintenance_impl) doAction0() bool {
	stateChanged := false

	log.Debugf("Executing: doAction0")

	var found bool
	var neighbor *NeighborState
	for _, neighbor = range this.neighborState {
		temp5 := neighbor.state.Dist
		temp6 := this.state.Dist
		temp7 := temp5 < temp6
		temp8 := neighbor.id
		temp9 := this.isNeighborUp(temp8)
		temp10 := neighbor.state.Inv
		temp11 := CMAX
		temp12 := temp10 < temp11
		temp13 := neighbor.state.Inv
		temp14 := this.state.Inv
		temp15 := temp13 < temp14
		temp16 := temp12 && temp15
		temp17 := temp9 && temp16
		temp18 := temp7 && temp17
		if temp18 {
			found = true
			break
		}
	}
	if found {
		temp19 := neighbor.id
		this.state.P = temp19
		temp20 := neighbor.state.Inv
		this.state.Inv = temp20
		stateChanged = true
	}

	log.Debugf("doAction0: state changed: %v", stateChanged)

	return stateChanged
}

func (this *RoutingTreeMaintenance_impl) doAction1() bool {
	stateChanged := false

	log.Debugf("Executing: doAction1")

	var found bool
	var neighbor *NeighborState
	for _, neighbor = range this.neighborState {
		temp21 := neighbor.state.Dist
		temp22 := this.state.Dist
		temp23 := temp21 < temp22
		temp24 := neighbor.id
		temp25 := this.isNeighborUp(temp24)
		temp26 := neighbor.state.Inv
		temp27 := int64(1)
		temp28 := temp26 + temp27
		temp29 := CMAX
		temp30 := temp28 < temp29
		temp31 := neighbor.state.Inv
		temp32 := int64(1)
		temp33 := temp31 + temp32
		temp34 := this.state.Inv
		temp35 := temp33 < temp34
		temp36 := temp30 && temp35
		temp37 := temp25 && temp36
		temp38 := temp23 && temp37
		if temp38 {
			found = true
			break
		}
	}
	if found {
		temp39 := neighbor.id
		this.state.P = temp39
		temp40 := neighbor.state.Inv
		temp41 := int64(1)
		temp42 := temp40 + temp41
		this.state.Inv = temp42
		stateChanged = true
	}

	log.Debugf("doAction1: state changed: %v", stateChanged)

	return stateChanged
}

func (this *RoutingTreeMaintenance_impl) doAction2() bool {
	stateChanged := false

	log.Debugf("Executing: doAction2")

	var found bool
	var neighbor *NeighborState
	for _, neighbor = range this.neighborState {
		temp43 := this.state.P
		temp44 := ""
		temp45 := temp43 != temp44
		temp46 := this.state.P
		temp47 := this.isNeighborUp(temp46)
		temp48 := false
		temp49 := temp47 == temp48
		var temp50 int64
		if neighbor.id == this.state.P {
			temp50 = this.state.Inv
		} else {
			continue
		}
		temp51 := CMAX
		temp52 := temp50 >= temp51
		var temp53 int64
		if neighbor.id == this.state.P {
			temp53 = this.state.Dist
		} else {
			continue
		}
		temp54 := this.state.Dist
		temp55 := temp53 < temp54
		temp56 := this.state.Inv
		var temp57 int64
		if neighbor.id == this.state.P {
			temp57 = this.state.Inv
		} else {
			continue
		}
		temp58 := temp56 != temp57
		temp59 := temp55 && temp58
		var temp60 int64
		if neighbor.id == this.state.P {
			temp60 = this.state.Dist
		} else {
			continue
		}
		temp61 := this.state.Dist
		temp62 := temp60 > temp61
		temp63 := this.state.Inv
		var temp64 int64
		if neighbor.id == this.state.P {
			temp64 = this.state.Inv
		} else {
			continue
		}
		temp65 := int64(1)
		temp66 := temp64 + temp65
		temp67 := temp63 != temp66
		temp68 := temp62 && temp67
		temp69 := temp59 || temp68
		temp70 := temp52 || temp69
		temp71 := temp49 || temp70
		temp72 := temp45 && temp71
		if temp72 {
			found = true
			break
		}
	}
	if found {
		temp73 := ""
		this.state.P = temp73
		temp74 := CMAX
		this.state.Inv = temp74
		stateChanged = true
	}

	log.Debugf("doAction2: state changed: %v", stateChanged)

	return stateChanged
}

func (this *RoutingTreeMaintenance_impl) doAction3() bool {
	stateChanged := false

	log.Debugf("Executing: doAction3")

	temp75 := this.state.P
	temp76 := ""
	temp77 := temp75 == temp76
	temp78 := this.state.Inv
	temp79 := CMAX
	temp80 := temp78 < temp79
	temp81 := temp77 && temp80
	if temp81 {
		temp82 := CMAX
		this.state.Inv = temp82
		stateChanged = true
	}

	log.Debugf("doAction3: state changed: %v", stateChanged)

	return stateChanged
}

func (this *RoutingTreeMaintenance_impl) updateLocalState() bool {
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

func (this *RoutingTreeMaintenance_impl) broadcastLocalState() (int, error) {
	updMessage := &p.NeighborUpdate{
		Id: this.id,
		State: &p.State{

			Dist: this.state.Dist,
			Inv:  this.state.Inv,
			P:    this.state.P,
		},
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_StateUpdate,
		Src:  this.id,
		Msg:  &p.BroadcastMessage_Upd{updMessage},
	}

	return this.send(broadcastMessage)
}

func (this *RoutingTreeMaintenance_impl) sendHeartBeat() (int, error) {
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

func (this *RoutingTreeMaintenance_impl) send(msg *p.BroadcastMessage) (int, error) {
	log.Debugf("Sending: %+v", msg)
	data, err := proto.Marshal(msg)
	if err != nil {
		return 0, err
	}
	return this.mcastConn.Write(data)
}

func (this *RoutingTreeMaintenance_impl) msgHandler(src *net.UDPAddr, n int, b []byte) {
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

func (this *RoutingTreeMaintenance_impl) Listener(ctx context.Context) {
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
