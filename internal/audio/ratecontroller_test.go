package audio

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAudioRateControllerPrebuffersThenPacesFrames(t *testing.T) {
	clock := &fakeRateClock{now: time.Unix(0, 0)}
	controller := NewAudioRateController(AudioRateControllerConfig{
		FrameDuration:   60 * time.Millisecond,
		PrebufferFrames: 5,
		Now:             clock.Now,
		Sleep:           clock.Sleep,
	})

	var sent []string
	send := func(_ context.Context, frame []byte) error {
		sent = append(sent, string(frame))
		return nil
	}
	for i := 0; i < 5; i++ {
		ok, err := controller.Send(context.Background(), []byte{byte('a' + i)}, send, nil)
		if err != nil || !ok {
			t.Fatalf("prebuffer send %d = ok:%v err:%v", i, ok, err)
		}
	}
	if len(clock.sleeps) != 0 {
		t.Fatalf("prebuffer sleeps = %v, want none", clock.sleeps)
	}

	ok, err := controller.Send(context.Background(), []byte("f"), send, nil)
	if err != nil || !ok {
		t.Fatalf("paced send = ok:%v err:%v", ok, err)
	}
	if got, want := clock.sleeps, []time.Duration{60 * time.Millisecond}; !equalDurations(got, want) {
		t.Fatalf("sleeps = %v, want %v", got, want)
	}
	if got, want := sent, []string{"a", "b", "c", "d", "e", "f"}; !equalStrings(got, want) {
		t.Fatalf("sent = %v, want %v", got, want)
	}

	ok, err = controller.Send(context.Background(), []byte("g"), send, nil)
	if err != nil || !ok {
		t.Fatalf("second paced send = ok:%v err:%v", ok, err)
	}
	if got, want := clock.sleeps, []time.Duration{60 * time.Millisecond, 60 * time.Millisecond}; !equalDurations(got, want) {
		t.Fatalf("sleeps = %v, want %v", got, want)
	}
}

func TestAudioRateControllerAbortAndReset(t *testing.T) {
	clock := &fakeRateClock{now: time.Unix(0, 0)}
	controller := NewAudioRateController(AudioRateControllerConfig{
		FrameDuration:   60 * time.Millisecond,
		PrebufferFrames: 1,
		Now:             clock.Now,
		Sleep:           clock.Sleep,
	})

	sent := 0
	send := func(_ context.Context, _ []byte) error {
		sent++
		return nil
	}
	ok, err := controller.Send(context.Background(), []byte("a"), send, func() bool { return false })
	if err != nil || !ok || sent != 1 {
		t.Fatalf("initial send = ok:%v err:%v sent:%d", ok, err, sent)
	}

	abort := false
	clock.onSleep = func() { abort = true }
	ok, err = controller.Send(context.Background(), []byte("b"), send, func() bool { return abort })
	if err != nil {
		t.Fatalf("abort send err = %v", err)
	}
	if ok || sent != 1 {
		t.Fatalf("abort send = ok:%v sent:%d, want dropped without send", ok, sent)
	}

	controller.Reset()
	clock.onSleep = nil
	abort = false
	ok, err = controller.Send(context.Background(), []byte("c"), send, func() bool { return abort })
	if err != nil || !ok || sent != 2 {
		t.Fatalf("reset send = ok:%v err:%v sent:%d", ok, err, sent)
	}
	if got, want := len(clock.sleeps), 1; got != want {
		t.Fatalf("sleep count after reset send = %d, want %d", got, want)
	}
}

func TestAudioRateControllerReturnsSendErrors(t *testing.T) {
	controller := NewAudioRateController(AudioRateControllerConfig{})
	want := errors.New("send failed")
	ok, err := controller.Send(context.Background(), []byte("a"), func(context.Context, []byte) error {
		return want
	}, nil)
	if ok || !errors.Is(err, want) {
		t.Fatalf("send result = ok:%v err:%v, want send error", ok, err)
	}
}

type fakeRateClock struct {
	now     time.Time
	sleeps  []time.Duration
	onSleep func()
}

func (c *fakeRateClock) Now() time.Time {
	return c.now
}

func (c *fakeRateClock) Sleep(_ context.Context, duration time.Duration) error {
	c.sleeps = append(c.sleeps, duration)
	c.now = c.now.Add(duration)
	if c.onSleep != nil {
		c.onSleep()
	}
	return nil
}

func equalDurations(a []time.Duration, b []time.Duration) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStrings(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
