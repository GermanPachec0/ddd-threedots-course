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
)

func Test(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := testSubscriber{}
	smsClient := &testSMSClient{}

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
		expectedMessages = append(expectedMessages, testMessage{
			Phone:   fmt.Sprintf("%v", phone),
			Message: fmt.Sprintf("Welcome on board, user-%v!", phone),
		})
		sub.messages = append(sub.messages, msg)
	}

	err := ProcessMessages(ctx, sub, smsClient)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
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

	resolveTime  time.Time
	messageCount int

	failing bool
}

func (c *testSMSClient) SendSMS(phoneNumber string, msg string) error {
	c.messageCount++

	if c.messageCount == 1 {
		c.failing = true
		c.resolveTime = time.Now().UTC().Add(500 * time.Millisecond)
	}

	if c.failing {
		if c.resolveTime.Before(time.Now().UTC()) {
			c.failing = false
		} else {
			c.resolveTime = c.resolveTime.Add(100 * time.Millisecond)
			return errors.New("internal server error")
		}
	}

	c.messages = append(c.messages, testMessage{
		Phone:   phoneNumber,
		Message: msg,
	})

	return nil
}

type testSubscriber struct {
	messages []*message.Message
}

func (t testSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	ch := make(chan *message.Message)

	go func() {
		for i := 0; i < len(t.messages); i++ {
			msg := t.messages[i]
			select {
			case ch <- msg:
			case <-ctx.Done():
				return
			}

			select {
			case <-msg.Acked():
			case <-msg.Nacked():
				t.messages[i] = msg.Copy()
				i--
				time.Sleep(10 * time.Millisecond)
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (t testSubscriber) Close() error {
	return nil
}
