// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

func Test(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer: 20,
		Persistent:          true,
	}, logger)

	smsClient := &testSMSClient{
		ticker: time.NewTicker(time.Second),
	}

	var expectedMessages []testMessage
	for i := 0; i < 20; i++ {
		phone := fmt.Sprint(1000000 + rand.Intn(1000000))

		event := UserSignedUp{
			Username:    fmt.Sprintf("user-%v", phone),
			PhoneNumber: phone,
			SignedUpAt:  time.Now().UTC().Format(time.RFC3339),
		}
		payload, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}

		msg := message.NewMessage(watermill.NewUUID(), payload)
		err = pubSub.Publish("UserSignedUp", msg)
		if err != nil {
			t.Fatal(err)
		}

		expectedMessages = append(expectedMessages, testMessage{
			Phone:   fmt.Sprintf("%v", phone),
			Message: fmt.Sprintf("Welcome on board, user-%v!", phone),
		})
	}

	err := ProcessMessages(ctx, pubSub, smsClient)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		if len(smsClient.messages) == len(expectedMessages) {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if len(smsClient.messages) != len(expectedMessages) {
		t.Fatalf("expected %v messages, got %v", len(expectedMessages), len(smsClient.messages))
	}

	for _, expected := range expectedMessages {
		found := false
		for _, actual := range smsClient.messages {
			if actual.Phone == expected.Phone && actual.Message == expected.Message {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected message %+v not found", expected)
		}
	}
}

type testMessage struct {
	Phone   string
	Message string
}

type testSMSClient struct {
	messages []testMessage

	ticker                *time.Ticker
	messagesSinceLastTick int

	blocked bool
}

func (c *testSMSClient) SendSMS(phoneNumber string, msg string) error {
	if c.blocked {
		return errors.New("the API is blocked, please contact support")
	}

	select {
	case <-c.ticker.C:
		c.messagesSinceLastTick = 0
	default:
		if c.messagesSinceLastTick > 10 {
			c.blocked = true
			return errors.New("rate limit exceeded")
		}
	}

	c.messages = append(c.messages, testMessage{
		Phone:   phoneNumber,
		Message: msg,
	})
	c.messagesSinceLastTick++

	return nil
}
