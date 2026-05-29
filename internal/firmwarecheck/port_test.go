package firmwarecheck

import "testing"

func TestPortUsageFromLsofOutputTreatsMissingDeviceAsFree(t *testing.T) {
	usage := portUsageFromLsofOutput("lsof: status error on /dev/cu.usbmodemA21: No such file or directory\n")
	if usage.InUse {
		t.Fatalf("InUse = true, want false: %#v", usage)
	}
}

func TestPortUsageFromLsofOutputDetectsProcessOwner(t *testing.T) {
	usage := portUsageFromLsofOutput("p17499\ncPython\nf3\n")
	if !usage.InUse {
		t.Fatal("InUse = false, want true")
	}
	if usage.Detail == "" {
		t.Fatal("expected detail")
	}
}

func TestLsofPortUsageArgsUseMacCompatibleForm(t *testing.T) {
	args := lsofPortUsageArgs("/dev/cu.usbmodem1101")
	for _, arg := range args {
		if arg == "--" {
			t.Fatal("macOS lsof rejects -- in this environment")
		}
	}
	if args[len(args)-1] != "/dev/cu.usbmodem1101" {
		t.Fatalf("last arg = %q, want port path", args[len(args)-1])
	}
}
