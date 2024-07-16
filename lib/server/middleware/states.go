package middleware

import (
	. "backend/lib/maintenance"

	"github.com/gofiber/fiber/v2"
)

func OnMode(required Mode) fiber.Handler {
	return func(c *fiber.Ctx) error {
		state_machine := c.Locals("StateMachine").(*StateMachine)
		mode, _, _ := state_machine.Get()

		if mode != required {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Maintenance server in invalid mode",
			})
		}
		return c.Next()
	}
}

func OnState(required State) fiber.Handler {
	return func(c *fiber.Ctx) error {
		state_machine := c.Locals("StateMachine").(*StateMachine)
		_, state, _ := state_machine.Get()

		if state != required {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Maintenance server in invalid state",
			})
		}
		return c.Next()
	}
}

func OnSubstate(required SubState) fiber.Handler {
	return func(c *fiber.Ctx) error {
		state_machine := c.Locals("StateMachine").(*StateMachine)
		_, _, substate := state_machine.Get()

		if substate != required {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Maintenance server in invalid state",
			})
		}
		return c.Next()
	}
}
