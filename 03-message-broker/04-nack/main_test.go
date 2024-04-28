// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/assert"
)

const topic = "smoke_sensor"

func Test(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	pubSub := gochannel.NewGoChannel(gochannel.Config{}, logger)

	alarm := &Alarm{}

	go ConsumeMessages(pubSub, alarm)

	time.Sleep(1 * time.Second)

	publishOn := func() {
		messageOn := message.NewMessage(watermill.NewUUID(), []byte("1"))
		err := pubSub.Publish(topic, messageOn)
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	publishOff := func() {
		messageOff := message.NewMessage(watermill.NewUUID(), []byte("0"))
		err := pubSub.Publish(topic, messageOff)
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	publishOn()
	assert.True(t, alarm.enabled, "alarm should be enabled")

	publishOff()
	assert.False(t, alarm.enabled, "alarm should be disabled")

	alarm.returnedErr = errors.New("error")

	publishOn()
	assert.False(t, alarm.enabled, "alarm should not be enabled")

	alarm.returnedErr = nil
	time.Sleep(100 * time.Millisecond)
	assert.True(t, alarm.enabled, "alarm should be enabled")

	publishOn()
	assert.True(t, alarm.enabled, "alarm should be enabled")

	alarm.returnedErr = errors.New("error")
	publishOff()
	assert.True(t, alarm.enabled, "alarm should be enabled")

	alarm.returnedErr = nil
	time.Sleep(100 * time.Millisecond)
	assert.False(t, alarm.enabled, "alarm should not be enabled")
}

type Alarm struct {
	enabled     bool
	returnedErr error
}

func (a *Alarm) StartAlarm() error {
	if a.returnedErr == nil {
		a.enabled = true
		return nil
	}

	return a.returnedErr
}

func (a *Alarm) StopAlarm() error {
	if a.returnedErr == nil {
		a.enabled = false
		return nil
	}

	return a.returnedErr
}
