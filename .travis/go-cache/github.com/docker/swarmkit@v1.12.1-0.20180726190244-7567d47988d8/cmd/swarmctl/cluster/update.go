package cluster

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cli"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/spf13/cobra"
)

var (
	externalCAOpt cli.ExternalCAOpt

	updateCmd = &cobra.Command{
		Use:   "update <cluster name>",
		Short: "Update a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("cluster name missing")
			}

			if len(args) > 1 {
				return errors.New("update command takes exactly 1 argument")
			}

			c, err := common.Dial(cmd)
			if err != nil {
				return err
			}

			cluster, err := getCluster(common.Context(cmd), c, args[0])
			if err != nil {
				return err
			}

			flags := cmd.Flags()
			spec := &cluster.Spec
			var rotation api.KeyRotation

			if flags.Changed("certexpiry") {
				cePeriod, err := flags.GetDuration("certexpiry")
				if err != nil {
					return err
				}
				ceProtoPeriod := gogotypes.DurationProto(cePeriod)
				spec.CAConfig.NodeCertExpiry = ceProtoPeriod
			}
			if flags.Changed("external-ca") {
				spec.CAConfig.ExternalCAs = externalCAOpt.Value()
			}
			if flags.Changed("taskhistory") {
				taskHistory, err := flags.GetInt64("taskhistory")
				if err != nil {
					return err
				}
				spec.Orchestration.TaskHistoryRetentionLimit = taskHistory
			}
			if flags.Changed("heartbeatperiod") {
				hbPeriod, err := flags.GetDuration("heartbeatperiod")
				if err != nil {
					return err
				}
				spec.Dispatcher.HeartbeatPeriod = gogotypes.DurationProto(hbPeriod)
			}
			if flags.Changed("rotate-join-token") {
				rotateJoinToken, err := flags.GetString("rotate-join-token")
				if err != nil {
					return err
				}
				rotateJoinToken = strings.ToLower(rotateJoinToken)

				switch rotateJoinToken {
				case "worker":
					rotation.WorkerJoinToken = true
				case "manager":
					rotation.ManagerJoinToken = true
				default:
					return errors.New("--rotate-join-token flag must be followed by 'worker' or 'manager'")
				}
			}
			if flags.Changed("autolock") {
				spec.EncryptionConfig.AutoLockManagers, err = flags.GetBool("autolock")
				if err != nil {
					return err
				}
			}
			rotateUnlockKey, err := flags.GetBool("rotate-unlock-key")
			if err != nil {
				return err
			}
			rotation.ManagerUnlockKey = rotateUnlockKey

			driver, err := common.ParseLogDriverFlags(flags)
			if err != nil {
				return err
			}
			spec.TaskDefaults.LogDriver = driver

			r, err := c.UpdateCluster(common.Context(cmd), &api.UpdateClusterRequest{
				ClusterID:      cluster.ID,
				ClusterVersion: &cluster.Meta.Version,
				Spec:           spec,
				Rotation:       rotation,
			})
			if err != nil {
				return err
			}
			fmt.Println(r.Cluster.ID)

			if rotation.ManagerUnlockKey {
				return displayUnlockKey(cmd)
			}
			return nil
		},
	}
)

func init() {
	updateCmd.Flags().Int64("taskhistory", 0, "Number of historic task entries to retain per slot or node")
	updateCmd.Flags().Duration("certexpiry", 24*30*3*time.Hour, "Duration node certificates will be valid for")
	updateCmd.Flags().Var(&externalCAOpt, "external-ca", "Specifications of one or more certificate signing endpoints")
	updateCmd.Flags().Duration("heartbeatperiod", 0, "Period when heartbeat is expected to receive from agent")

	updateCmd.Flags().String("log-driver", "", "Set default log driver for cluster")
	updateCmd.Flags().StringSlice("log-opt", nil, "Set options for default log driver")
	updateCmd.Flags().String("rotate-join-token", "", "Rotate join token for worker or manager")
	updateCmd.Flags().Bool("rotate-unlock-key", false, "Rotate manager unlock key")
	updateCmd.Flags().Bool("autolock", false, "Enable or disable manager autolocking (requiring an unlock key to start a stopped manager)")
}
