package firmwarecheck

import "testing"

func TestPortUsageFromLsofOutputTreatsMissingDeviceAsFree(t *testing.T) {
	usage := portUsageFromLsofOutput("lsof: status error on /dev/cu.usbmodemA21: No such file or directory\n")
	if usage.InUse {
		t.Fatalf("InUse = true, want false: %#v", usage)
	}
}

func TestDetectPortUsageReportsMissingDevice(t *testing.T) {
	usage, err := DetectPortUsage(t.Context(), "/dev/a21-missing-serial-device")
	if err != nil {
		t.Fatal(err)
	}
	if usage.Exists {
		t.Fatalf("Exists = true, want false: %#v", usage)
	}
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

func TestSerialDevicesFromPathsSortsAndClassifiesUSBModem(t *testing.T) {
	devices := serialDevicesFromPaths([]string{
		"/dev/cu.debug-console",
		"/dev/cu.usbmodem1101",
	}, func(port string) (PortUsage, error) {
		return PortUsage{Exists: true, InUse: port == "/dev/cu.usbmodem1101", Detail: "p1 cmonitor"}, nil
	})

	if len(devices) != 2 {
		t.Fatalf("len = %d, want 2", len(devices))
	}
	if devices[0].Path != "/dev/cu.debug-console" {
		t.Fatalf("first path = %q", devices[0].Path)
	}
	if !devices[1].USBModem {
		t.Fatal("expected usbmodem path to be classified")
	}
	if !devices[1].Usage.InUse {
		t.Fatal("expected usage to propagate")
	}
}

func TestValidateUploadPortRejectsNonUSBMacSerialPath(t *testing.T) {
	for _, port := range []string{
		"/dev/cu.Bluetooth-Incoming-Port",
		"/dev/cu.debug-console",
		"/dev/tty.debug-console",
	} {
		if err := validateUploadPort(port); err == nil {
			t.Fatalf("validateUploadPort(%q) returned nil, want error", port)
		}
	}
}

func TestValidateUploadPortAcceptsExplicitUSBSerialPath(t *testing.T) {
	for _, port := range []string{
		"/dev/cu.usbmodem1101",
		"/dev/tty.usbmodem1101",
		"/dev/ttyUSB0",
		"/dev/ttyACM0",
	} {
		if err := validateUploadPort(port); err != nil {
			t.Fatalf("validateUploadPort(%q) returned error: %v", port, err)
		}
	}
}
