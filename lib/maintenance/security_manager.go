package maintenance

type SecurityManager struct {
	enginesTokenApplication     bool
	ChanEnginesTokenApplication chan bool

	apiTokenApplication     bool
	ChanApiTokenApplication chan bool

	servicesTokenApplication     bool
	ChanServicesTokenApplication chan bool
}

func NewSecurityManager() (SecurityManager, error) {

	manager := SecurityManager{
		enginesTokenApplication:     false,
		ChanEnginesTokenApplication: make(chan bool),

		apiTokenApplication:     false,
		ChanApiTokenApplication: make(chan bool),

		servicesTokenApplication:     false,
		ChanServicesTokenApplication: make(chan bool),
	}

	return manager, nil
}

func (manager *SecurityManager) Start(state_machine *StateMachine) {

	go func() {
		for {
			select {
			case is_applied := <-manager.ChanServicesTokenApplication:
				if is_applied {
					manager.servicesTokenApplication = true
				}
			case is_applied := <-manager.ChanApiTokenApplication:
				if is_applied {
					manager.apiTokenApplication = true
				}
			}

			if manager.servicesTokenApplication && manager.apiTokenApplication {
				if err := state_machine.To(MODE_INIT, STATE_CONFIGURING, SUBSTATE_CONFIGURING_SERVICES); err != nil {
					// TODO : log error
					break
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case is_applied := <-manager.ChanEnginesTokenApplication:
				if is_applied {
					manager.enginesTokenApplication = true
				}

			}

			if manager.enginesTokenApplication {
				if err := state_machine.To(MODE_OPERATIONAL, STATE_RUNNING, SUBSTATE_SAFE); err != nil {
					// TODO : log error
					break
				}
			}
		}
	}()
}
