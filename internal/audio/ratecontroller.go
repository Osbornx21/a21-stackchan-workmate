package audio

import (
	"context"
	"errors"
	"time"
)

var ErrAudioRateControllerSendNil = errors.New("audio rate controller send function is nil")

type AudioRateControllerConfig struct {
	FrameDuration   time.Duration
	PrebufferFrames int
	Now             func() time.Time
	Sleep           func(context.Context, time.Duration) error
}

type AudioRateController struct {
	frameDuration   time.Duration
	prebufferFrames int
	now             func() time.Time
	sleep           func(context.Context, time.Duration) error
	sentFrames      int
	startedAt       time.Time
}

func NewAudioRateController(config AudioRateControllerConfig) *AudioRateController {
	frameDuration := config.FrameDuration
	if frameDuration <= 0 {
		frameDuration = 60 * time.Millisecond
	}
	prebufferFrames := config.PrebufferFrames
	if prebufferFrames < 0 {
		prebufferFrames = 0
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	sleep := config.Sleep
	if sleep == nil {
		sleep = sleepContext
	}
	return &AudioRateController{
		frameDuration:   frameDuration,
		prebufferFrames: prebufferFrames,
		now:             now,
		sleep:           sleep,
	}
}

func (c *AudioRateController) Reset() {
	c.sentFrames = 0
	c.startedAt = time.Time{}
}

func (c *AudioRateController) Send(ctx context.Context, frame []byte, send func(context.Context, []byte) error, shouldAbort func() bool) (bool, error) {
	if send == nil {
		return false, ErrAudioRateControllerSendNil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	aborted := func() bool {
		return shouldAbort != nil && shouldAbort()
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if aborted() {
		return false, nil
	}
	if c.sentFrames >= c.prebufferFrames {
		if err := c.waitForSlot(ctx); err != nil {
			return false, err
		}
		if aborted() {
			return false, nil
		}
	}
	if err := send(ctx, frame); err != nil {
		return false, err
	}
	c.sentFrames++
	return true, nil
}

func (c *AudioRateController) waitForSlot(ctx context.Context) error {
	if c.startedAt.IsZero() {
		c.startedAt = c.now()
	}
	frameIndex := c.sentFrames - c.prebufferFrames + 1
	targetAt := c.startedAt.Add(time.Duration(frameIndex) * c.frameDuration)
	if delay := targetAt.Sub(c.now()); delay > 0 {
		return c.sleep(ctx, delay)
	}
	return nil
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
