package flagparser

import (
	"errors"
	"time"

	"github.com/docker/swarmkit/api"
	"github.com/spf13/pflag"
)

func parseUpdate(flags *pflag.FlagSet, spec *api.ServiceSpec) error {
	if flags.Changed("update-parallelism") {
		parallelism, err := flags.GetUint64("update-parallelism")
		if err != nil {
			return err
		}
		if spec.Update == nil {
			spec.Update = &api.UpdateConfig{}
		}
		spec.Update.Parallelism = parallelism
	}

	if flags.Changed("update-delay") {
		delay, err := flags.GetString("update-delay")
		if err != nil {
			return err
		}

		delayDuration, err := time.ParseDuration(delay)
		if err != nil {
			return err
		}

		if spec.Update == nil {
			spec.Update = &api.UpdateConfig{}
		}
		spec.Update.Delay = delayDuration
	}

	if flags.Changed("update-on-failure") {
		if spec.Update == nil {
			spec.Update = &api.UpdateConfig{}
		}

		action, err := flags.GetString("update-on-failure")
		if err != nil {
			return err
		}
		switch action {
		case "pause":
			spec.Update.FailureAction = api.UpdateConfig_PAUSE
		case "continue":
			spec.Update.FailureAction = api.UpdateConfig_CONTINUE
		case "rollback":
			spec.Update.FailureAction = api.UpdateConfig_ROLLBACK
		default:
			return errors.New("--update-on-failure value must be pause or continue")
		}
	}

	if flags.Changed("update-order") {
		if spec.Update == nil {
			spec.Update = &api.UpdateConfig{}
		}

		order, err := flags.GetString("update-order")
		if err != nil {
			return err
		}

		switch order {
		case "stop-first":
			spec.Update.Order = api.UpdateConfig_STOP_FIRST
		case "start-first":
			spec.Update.Order = api.UpdateConfig_START_FIRST
		default:
			return errors.New("--update-order value must be stop-first or start-first")
		}
	}

	if flags.Changed("rollback-parallelism") {
		parallelism, err := flags.GetUint64("rollback-parallelism")
		if err != nil {
			return err
		}
		if spec.Rollback == nil {
			spec.Rollback = &api.UpdateConfig{}
		}
		spec.Rollback.Parallelism = parallelism
	}

	if flags.Changed("rollback-delay") {
		delay, err := flags.GetString("rollback-delay")
		if err != nil {
			return err
		}

		delayDuration, err := time.ParseDuration(delay)
		if err != nil {
			return err
		}

		if spec.Rollback == nil {
			spec.Rollback = &api.UpdateConfig{}
		}
		spec.Rollback.Delay = delayDuration
	}

	if flags.Changed("rollback-on-failure") {
		if spec.Rollback == nil {
			spec.Rollback = &api.UpdateConfig{}
		}

		action, err := flags.GetString("rollback-on-failure")
		if err != nil {
			return err
		}
		switch action {
		case "pause":
			spec.Rollback.FailureAction = api.UpdateConfig_PAUSE
		case "continue":
			spec.Rollback.FailureAction = api.UpdateConfig_CONTINUE
		default:
			return errors.New("--rollback-on-failure value must be pause or continue")
		}
	}

	if flags.Changed("rollback-order") {
		if spec.Rollback == nil {
			spec.Rollback = &api.UpdateConfig{}
		}

		order, err := flags.GetString("rollback-order")
		if err != nil {
			return err
		}

		switch order {
		case "stop-first":
			spec.Rollback.Order = api.UpdateConfig_STOP_FIRST
		case "start-first":
			spec.Rollback.Order = api.UpdateConfig_START_FIRST
		default:
			return errors.New("--rollback-order value must be stop-first or start-first")
		}
	}

	return nil
}
