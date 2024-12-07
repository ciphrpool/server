package maintenance

import (
	"fmt"
	"log/slog"
	"sync"
)

type Mode int
type State int
type SubState int

const (
	MODE_INIT Mode = iota
	MODE_OPERATIONAL
)

const (
	STATE_CONFIGURING State = iota
	STATE_RUNNING
	STATE_FAILED
)

const (
	SUBSTATE_CONFIGURING_INIT     SubState = iota // Configure Vault Tokens and retrieve keys
	SUBSTATE_CONFIGURING_SERVICES                 // Configure all services : Db, Cache, Notification, Authentication
	SUBSTATE_CONFIGURING_SECURITY                 // Connect to NexusPool
	SUBSTATE_SAFE
	SUBSTATE_UNSAFE
	SUBSTATE_FAILED
)

type MSS struct {
	mode     Mode
	state    State
	substate SubState
}
type MSSTransition struct {
	From MSS
	To   MSS
}

var transitions = map[MSSTransition]struct{}{
	{MSS{MODE_INIT, STATE_CONFIGURING, SUBSTATE_CONFIGURING_INIT}, MSS{MODE_INIT, STATE_CONFIGURING, SUBSTATE_CONFIGURING_SERVICES}}:     {},
	{MSS{MODE_INIT, STATE_CONFIGURING, SUBSTATE_CONFIGURING_SERVICES}, MSS{MODE_INIT, STATE_CONFIGURING, SUBSTATE_CONFIGURING_SECURITY}}: {},
	{MSS{MODE_INIT, STATE_CONFIGURING, SUBSTATE_CONFIGURING_SECURITY}, MSS{MODE_OPERATIONAL, STATE_RUNNING, SUBSTATE_SAFE}}:              {},
	{MSS{MODE_INIT, STATE_CONFIGURING, SUBSTATE_CONFIGURING_SECURITY}, MSS{MODE_OPERATIONAL, STATE_RUNNING, SUBSTATE_UNSAFE}}:            {},
	{MSS{MODE_OPERATIONAL, STATE_RUNNING, SUBSTATE_SAFE}, MSS{MODE_OPERATIONAL, STATE_RUNNING, SUBSTATE_UNSAFE}}:                         {},
}

func names(mode Mode, state State, substate SubState) (string, string, string) {
	var mode_name string
	switch mode {
	case MODE_INIT:
		mode_name = "INIT"
	case MODE_OPERATIONAL:
		mode_name = "OPERATIONAL"
	}

	var state_name string
	switch state {
	case STATE_CONFIGURING:
		state_name = "CONFIGURING"
	case STATE_FAILED:
		state_name = "FAILED"
	case STATE_RUNNING:
		state_name = "RUNNING"
	}

	var substate_name string
	switch substate {
	case SUBSTATE_CONFIGURING_INIT:
		substate_name = "CONFIGURING_INIT"
	case SUBSTATE_CONFIGURING_SECURITY:
		substate_name = "CONFIGURING_SECURITY"
	case SUBSTATE_CONFIGURING_SERVICES:
		substate_name = "CONFIGURING_SERVICES"
	case SUBSTATE_SAFE:
		substate_name = "SAFE"
	case SUBSTATE_UNSAFE:
		substate_name = "UNSAFE"
	case SUBSTATE_FAILED:
		substate_name = "FAILED"
	}
	return mode_name, state_name, substate_name
}

type StateMachine struct {
	mode     Mode
	state    State
	substate SubState

	signals map[MSS][]func()

	mutex sync.RWMutex
}

func NewStateMachine() StateMachine {
	return StateMachine{
		mode:     MODE_INIT,
		state:    STATE_CONFIGURING,
		substate: SUBSTATE_CONFIGURING_INIT,
		signals:  make(map[MSS][]func()),
	}
}

func (state_machine *StateMachine) Get() (Mode, State, SubState) {
	state_machine.mutex.RLock()
	defer state_machine.mutex.RUnlock()

	return state_machine.mode, state_machine.state, state_machine.substate
}

func (state_machine *StateMachine) To(mode Mode, state State, substate SubState) error {
	current_mode, current_state, current_substate := state_machine.Get()

	if _, ok := transitions[MSSTransition{MSS{current_mode, current_state, current_substate}, MSS{mode, state, substate}}]; ok {
		state_machine.accept(mode, state, substate)
		return nil
	} else if state == STATE_FAILED || substate == SUBSTATE_FAILED {
		state_machine.accept(mode, STATE_FAILED, SUBSTATE_FAILED)
		return nil
	} else {
		mode_name, state_name, substate_name := names(mode, state, substate)
		slog.Warn("MSS : Invalid mode state substate transition", "mode", mode_name, "state", state_name, "substate", substate_name)
		return fmt.Errorf("invalid mode state substate transition")
	}
}

func (state_machine *StateMachine) accept(mode Mode, state State, substate SubState) {
	state_machine.mutex.Lock()
	defer state_machine.mutex.Unlock()

	if signals, ok := state_machine.signals[MSS{mode: mode, state: state, substate: substate}]; ok {
		for _, signal := range signals {
			go signal()
		}
	}

	state_machine.mode = mode
	state_machine.state = state
	state_machine.substate = substate
	mode_name, state_name, substate_name := names(mode, state, substate)
	slog.Info("MSS : transition done", "mode", mode_name, "state", state_name, "substate", substate_name)
}

func (state_machine *StateMachine) When(mode Mode, state State, substate SubState, callback func()) {
	key := MSS{mode: mode, state: state, substate: substate}
	if _, ok := state_machine.signals[key]; !ok {
		state_machine.signals[key] = []func(){}
	}

	state_machine.signals[key] = append(state_machine.signals[key], callback)
}
