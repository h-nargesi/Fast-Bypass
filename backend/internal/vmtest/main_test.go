//go:build vm

package vmtest

import (
	"fmt"
	"os"
	"testing"
)

var sharedCfg VMConfig

func TestMain(m *testing.M) {
	sharedCfg = LoadConfig()
	if reason := sharedCfg.SkipReason(); reason != "" {
		fmt.Fprintf(os.Stderr, "vm tests skipped: %s\n", reason)
		os.Exit(0)
	}
	if sharedCfg.ManageVM {
		if err := PrepareVM(sharedCfg); err != nil {
			fmt.Fprintf(os.Stderr, "vm prepare failed: %v\n", err)
			os.Exit(1)
		}
		defer StopVM(sharedCfg.VMName)
	}
	if err := WaitRouter(sharedCfg.WaitHost, sharedCfg.WaitPort, sharedCfg.WaitTimeout); err != nil {
		fmt.Fprintf(os.Stderr, "vm router wait: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
