package main

import (
	"log"
)

type User struct {
	Email string
}

type UserRepository interface {
	CreateUserAccount(u User) error
}

type NotificationsClient interface {
	SendNotification(u User) error
}

type NewsletterClient interface {
	AddToNewsletter(u User) error
}

type Handler struct {
	repository          UserRepository
	newsletterClient    NewsletterClient
	notificationsClient NotificationsClient
}

func NewHandler(
	repository UserRepository,
	newsletterClient NewsletterClient,
	notificationsClient NotificationsClient,
) Handler {
	return Handler{
		repository:          repository,
		newsletterClient:    newsletterClient,
		notificationsClient: notificationsClient,
	}
}

func (h Handler) SignUp(u User) error {
	if err := h.repository.CreateUserAccount(u); err != nil {
		return err
	}
	go func() {
		for {
			if err := h.newsletterClient.AddToNewsletter(u); err != nil {
				log.Printf("failed to add user to the newsletter: %v", err)
				continue
			}
			break
		}

	}()
	go func() {
		for {

			if err := h.notificationsClient.SendNotification(u); err != nil {
				log.Printf("failed to send user notification: %v", err)
				continue
			}
			break
		}
	}()

	return nil
}

// ANOTHER SOLUTION
// func (h Handler) SignUp(u User) error {
// 	if err := h.repository.CreateUserAccount(u); err != nil {
// 		return err
// 	}

// 	go retry(func() error {
// 		return h.newsletterClient.AddToNewsletter(u)
// 	})

// 	go retry(func() error {
// 		return h.notificationsClient.SendNotification(u)
// 	})

// 	return nil
// }

// func retry(f func() error) {
// 	for {
// 		if err := f(); err != nil {
// 			time.Sleep(10 * time.Millisecond)
// 			continue
// 		}
// 		return
// 	}
// }
