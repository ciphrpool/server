package maintenance

import "fmt"

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
		ChanEnginesTokenApplication: make(chan bool, 8),

		apiTokenApplication:     false,
		ChanApiTokenApplication: make(chan bool, 8),

		servicesTokenApplication:     false,
		ChanServicesTokenApplication: make(chan bool, 8),
	}

	return manager, nil
}

func (manager *SecurityManager) Start(state_machine *StateMachine) {

	fmt.Println("INFO : Starting the SecurityManager")
	go func() {
		for {
			select {
			case is_applied := <-manager.ChanServicesTokenApplication:
				if is_applied {
					Debug("SecurityManager : The services token is applied")
					manager.servicesTokenApplication = true
				}
			case is_applied := <-manager.ChanApiTokenApplication:
				if is_applied {
					Debug("SecurityManager : The api token is applied")
					manager.apiTokenApplication = true
				}
			}

			if manager.servicesTokenApplication && manager.apiTokenApplication {
				if err := state_machine.To(MODE_INIT, STATE_CONFIGURING, SUBSTATE_CONFIGURING_SERVICES); err != nil {
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
					Debug("SecurityManager : The engines token is applied")
					manager.enginesTokenApplication = true
				}

			}

			if manager.enginesTokenApplication {
				if err := state_machine.To(MODE_OPERATIONAL, STATE_RUNNING, SUBSTATE_SAFE); err != nil {
					break
				}
			}
		}
	}()
}
