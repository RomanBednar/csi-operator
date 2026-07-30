package main

import (
	"context"
	"os"
	"time"

	smb "github.com/openshift/csi-operator/pkg/driver/samba"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/spf13/cobra"
	"k8s.io/component-base/cli"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"

	"github.com/openshift/csi-operator/pkg/operator"
	"github.com/openshift/csi-operator/pkg/version"
)

func main() {
	command := NewOperatorCommand()
	code := cli.Run(command)
	os.Exit(code)
}

func NewOperatorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "smb-csi-driver-operator",
		Short: "OpenShift CIFS/SMB CSI Driver Operator",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
	}

	ctrlCmd := controllercmd.NewControllerCommandConfig(
		"smb-csi-driver-operator",
		version.Get(),
		runCSIDriverOperator,
		clock.RealClock{},
	).NewCommand()

	ctrlCmd.Use = "start"
	ctrlCmd.Short = "Start the CIFS/SMB CSI Driver Operator"

	// Inject cluster TLS profile into the operator's own HTTPS endpoint before
	// controllercmd starts the server.
	// See: openshift/enhancements#1910 § "OLM-managed operators"
	originalPreRunE := ctrlCmd.PersistentPreRunE
	ctrlCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if originalPreRunE != nil {
			if err := originalPreRunE(cmd, args); err != nil {
				return err
			}
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()
		configPath, err := operator.WriteOperatorTLSConfig(ctx, "smb-csi-driver-operator")
		if err != nil {
			klog.Warningf("Failed to write TLS config, continuing with defaults: %v", err)
			return nil
		}
		if configPath != "" {
			if err := cmd.Flags().Set("config", configPath); err != nil {
				klog.Warningf("Failed to set --config flag: %v", err)
			}
		}
		return nil
	}

	cmd.AddCommand(ctrlCmd)

	return cmd
}

func runCSIDriverOperator(ctx context.Context, controllerConfig *controllercmd.ControllerContext) error {
	opConfig := smb.GetSambaOperatorConfig()
	return operator.RunOperator(ctx, controllerConfig, "", opConfig)
}
