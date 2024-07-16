package maintenance

import (
	"errors"
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
	SUBSTATE_CONFIGURING_INIT SubState = iota
	SUBSTATE_CONFIGURING_SERVICES
	SUBSTATE_CONFIGURING_SECURITY
	SUBSTATE_SAFE
	SUBSTATE_UNSAFE
	SUBSTATE_FAILED
)

type MSS struct {
	mode     Mode
	state    State
	substate SubState
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

	var new_mode Mode
	new_mode = current_mode

	var new_state State
	new_state = current_state

	var new_substate SubState
	new_substate = current_substate

	switch mode {
	case MODE_INIT:
		if current_mode != MODE_INIT {
			return errors.New("State Machine : cannot change mode to INIT")
		}
	case MODE_OPERATIONAL:
		if current_mode == MODE_INIT && current_state != STATE_CONFIGURING && current_substate != SUBSTATE_CONFIGURING_SECURITY {
			return errors.New("State Machine : cannot change mode to OPERATIONAL")
		}
	default:
		return errors.New("State Machine : Invalid mode")
	}
	new_mode = mode

	switch state {
	case STATE_CONFIGURING:
		if new_mode != MODE_INIT || current_state != STATE_CONFIGURING {
			return errors.New("State Machine : cannot change state to CONFIGURING")
		}
	case STATE_RUNNING:
		if new_mode == MODE_INIT || (current_state == STATE_CONFIGURING && current_substate != SUBSTATE_CONFIGURING_SECURITY) {
			return errors.New("State Machine : cannot change state to RUNNING")
		}
	case STATE_FAILED:
		break
	default:
		return errors.New("State Machine : Invalid state")
	}
	new_state = state

	switch substate {
	case SUBSTATE_CONFIGURING_INIT:
		if new_mode != MODE_INIT || new_state != STATE_CONFIGURING {
			return errors.New("State Machine : cannot change substate to CONFIGURING_SERVICES")
		}
	case SUBSTATE_CONFIGURING_SERVICES:
		if new_mode != MODE_INIT || new_state != STATE_CONFIGURING ||
			(new_state == STATE_CONFIGURING && current_substate != SUBSTATE_CONFIGURING_INIT) {
			return errors.New("State Machine : cannot change substate to CONFIGURING_SERVICES")
		}
	case SUBSTATE_CONFIGURING_SECURITY:
		if new_mode != MODE_INIT || new_state != STATE_CONFIGURING ||
			(new_state == STATE_CONFIGURING && current_substate != SUBSTATE_CONFIGURING_SERVICES) {
			return errors.New("State Machine : cannot change substate to CONFIGURING_SECURITY")
		}
	case SUBSTATE_SAFE:
		if new_mode != MODE_OPERATIONAL || new_state != STATE_RUNNING {
			return errors.New("State Machine : cannot change substate to CONFIGURING_SECURITY")
		}
	case SUBSTATE_UNSAFE:
		if new_mode != MODE_OPERATIONAL || new_state != STATE_RUNNING {
			return errors.New("State Machine : cannot change substate to CONFIGURING_SECURITY")
		}
	case SUBSTATE_FAILED:
		break
	default:
		return errors.New("State Machine : Invalid substate")
	}
	new_substate = substate

	state_machine.accept(new_mode, new_state, new_substate)

	return nil
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
}

func (state_machine *StateMachine) When(mode Mode, state State, substate SubState, callback func()) {
	key := MSS{mode: mode, state: state, substate: substate}
	if _, ok := state_machine.signals[key]; !ok {
		state_machine.signals[key] = []func(){}
	}

	state_machine.signals[key] = append(state_machine.signals[key], callback)
}
