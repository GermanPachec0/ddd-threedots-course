// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	repository := &MemoryRepository{users: map[string]User{}}
	newsletterClient := &MemoryNewsletterClient{users: map[string]User{}}
	notificationsClient := &MemoryNotificationsClient{users: map[string]User{}}

	h := NewHandler(repository, newsletterClient, notificationsClient)

	u1 := newUser()
	err := h.SignUp(u1)
	if err != nil {
		t.Fatal(err)
	}

	repository.assertUserFound(t, u1)

	newsletterClient.assertUserFound(t, u1)
	notificationsClient.assertUserFound(t, u1)

	// The newsletter API goes down
	newsletterClient.returnedErr = errors.New("network error")

	u2 := newUser()
	err = h.SignUp(u2)
	if err != nil {
		t.Fatal(err)
	}

	repository.assertUserFound(t, u2)

	newsletterClient.assertUserNotFound(t, u2)
	notificationsClient.assertUserFound(t, u2)

	// The notifications API goes down
	notificationsClient.returnedErr = errors.New("network error")

	u3 := newUser()
	err = h.SignUp(u3)
	if err != nil {
		t.Fatal(err)
	}

	repository.assertUserFound(t, u3)

	newsletterClient.assertUserNotFound(t, u3)
	notificationsClient.assertUserNotFound(t, u3)

	// Both APIs come back up
	newsletterClient.returnedErr = nil
	notificationsClient.returnedErr = nil

	// Allow some time for retries
	time.Sleep(time.Second)

	// Expect the retries to add the missing users
	newsletterClient.assertUserFound(t, u2)
	newsletterClient.assertUserFound(t, u3)
	notificationsClient.assertUserFound(t, u3)
}

func newUser() User {
	return User{Email: uuid.NewString() + "@example.com"}
}

type MemoryRepository struct {
	users map[string]User
}

func (m *MemoryRepository) CreateUserAccount(u User) error {
	if u.Email == "" {
		return errors.New("missing email")
	}

	_, ok := m.users[u.Email]
	if ok {
		return errors.New("user already exists")
	}

	m.users[u.Email] = u
	return nil
}

func (m *MemoryRepository) assertUserFound(t *testing.T, u User) {
	t.Helper()
	_, ok := m.users[u.Email]
	if !ok {
		t.Fatalf("user %s not found", u.Email)
	}
}

type MemoryNewsletterClient struct {
	users       map[string]User
	returnedErr error
}

func (m *MemoryNewsletterClient) AddToNewsletter(u User) error {
	if m.returnedErr != nil {
		return m.returnedErr
	}
	m.users[u.Email] = u
	return nil
}

func (m *MemoryNewsletterClient) assertUserFound(t *testing.T, u User) {
	t.Helper()

	assert.Eventuallyf(t, func() bool {
		_, ok := m.users[u.Email]
		return ok
	}, time.Millisecond*20, time.Millisecond*5, "Expected user %v to be added to the newsletter", u.Email)
}

func (m *MemoryNewsletterClient) assertUserNotFound(t *testing.T, u User) {
	t.Helper()

	time.Sleep(time.Millisecond * 20)
	_, ok := m.users[u.Email]
	assert.False(t, ok, "Expected user %v to not be added to the newsletter", u.Email)
}

type MemoryNotificationsClient struct {
	users       map[string]User
	returnedErr error
}

func (m *MemoryNotificationsClient) SendNotification(u User) error {
	if m.returnedErr != nil {
		return m.returnedErr
	}
	m.users[u.Email] = u
	return nil
}

func (m *MemoryNotificationsClient) assertUserFound(t *testing.T, u User) {
	t.Helper()

	assert.Eventuallyf(t, func() bool {
		_, ok := m.users[u.Email]
		return ok
	}, time.Millisecond*20, time.Millisecond*5, "Expected user %v to have a notification sent", u.Email)
}

func (m *MemoryNotificationsClient) assertUserNotFound(t *testing.T, u User) {
	t.Helper()

	time.Sleep(time.Millisecond * 20)
	_, ok := m.users[u.Email]
	assert.False(t, ok, "Expected user %v to have no notifications sent", u.Email)
}
