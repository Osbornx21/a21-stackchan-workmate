package app

import (
	"context"
	"testing"

	"a21.local/a21/internal/runtimeguard"
)

func allowA21ControlGuardForTest(t *testing.T) {
	t.Helper()
	original := runA21ControlGuard
	runA21ControlGuard = func(ctx context.Context, input runtimeguard.ControlGuardInput) runtimeguard.ControlGuardReport {
		tier := input.Tier
		if spec, ok := runtimeguard.LookupControlCommandSpec(input.Command); ok {
			tier = spec.Tier
		}
		return runtimeguard.ControlGuardReport{
			SchemaVersion: runtimeguard.ControlGuardSchema,
			Command:       input.Command,
			Tier:          tier,
			Result:        runtimeguard.NewResult(nil),
		}
	}
	t.Cleanup(func() {
		runA21ControlGuard = original
	})
}
