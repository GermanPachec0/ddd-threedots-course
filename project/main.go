package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/receipts"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/spreadsheets"
	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lithammer/shortuuid/v3"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Task int

const (
	TaskIssueReceipt Task = iota
	TaskAppendToTracker
)

type Price struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}
type IssueReceiptPayload struct {
	TicketID string `json:"ticket_id"`
	Price    Price  `json:"price"`
}
type AppendToTrackerPayload struct {
	TicketID      string `json:"ticket_id"`
	CustomerEmail string `json:"customer_email"`
	Price         Price  `json:"price"`
}

type Header struct {
	ID          string `json:"id"`
	PublishedAt string `json:"published_at"`
}

type Ticket struct {
	Header        Header `json:"header"`
	ID            string `json:"ticket_id"`
	Status        string `json:"status"`
	CustomerEmail string `json:"customer_email"`
	Price         Price  `json:"price"`
}

type TicketsConfirmationRequest struct {
	Tickets []Ticket `json:"tickets"`
}
type Worker struct {
	queue chan Message
}

type Message struct {
	Task     Task
	TicketID string
}

type Publisher struct {
	pub message.Publisher
}

func main() {
	log.Init(logrus.InfoLevel)
	logger := watermill.NewStdLogger(false, false)
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	clients, err := clients.NewClients(os.Getenv("GATEWAY_ADDR"), func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Correlation-ID", log.CorrelationIDFromContext(ctx))
		return nil
	},
	)
	if err != nil {
		panic(err)
	}
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}
	router.AddMiddleware(PropagateCorrelationID)
	router.AddMiddleware(LogMessage)
	watermillLogger := log.NewWatermill(logrus.NewEntry(logrus.StandardLogger()))
	m := middleware.Retry{
		MaxRetries:      10,
		InitialInterval: time.Millisecond * 100,
		MaxInterval:     time.Second,
		Multiplier:      2,
		Logger:          watermillLogger,
	}
	router.AddMiddleware(m.Middleware)
	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: rdb,
	}, logger)
	if err != nil {
		panic(err)
	}
	issueReceiptSub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: "issue-receipt",
	}, logger)
	if err != nil {
		panic(err)
	}

	spreedSheetsSub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: "append-to-tracker",
	}, logger)
	if err != nil {
		panic(err)
	}

	e := commonHTTP.NewEcho()
	worker := NewWorker()

	pub := NewPublisher(publisher)
	worker.SubIssueReceipt(issueReceiptSub, clients, "TicketBookingConfirmed", router)
	worker.SubSpreedSheetReceipt(spreedSheetsSub, clients, "TicketBookingConfirmed", router, "tickets-to-print")
	worker.SubSpreedSheetReceipt(spreedSheetsSub, clients, "TicketBookingCanceled", router, "tickets-to-refund")

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	g, ctx := errgroup.WithContext(ctx)

	defer cancel()

	g.Go(func() error {
		return router.Run(ctx)
	})

	e.POST("/tickets-status", func(c echo.Context) error {
		var request TicketsConfirmationRequest
		err := c.Bind(&request)
		if err != nil {
			return err
		}
		corrId := c.Request().Header.Get("Correlation-ID")
		for _, ticket := range request.Tickets {
			if ticket.Status == "canceled" {
				err = pub.PublishBookingCanceled(ticket, corrId)
				if err != nil {
					return err
				}
			} else if ticket.Status == "confirmed" {
				err = pub.PublishBookingConfirmed(ticket, corrId)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("unknown ticket status: %s", ticket.Status)
			}

		}

		return c.NoContent(http.StatusOK)
	})

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	logrus.Info("Server starting...")

	g.Go(func() error {
		err := e.Start(":8080")
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		return e.Shutdown(ctx)
	})

	err = g.Wait()
	if err != nil {
		panic(err)
	}

}

type ReceiptsClient struct {
	clients *clients.Clients
}

func NewReceiptsClient(clients *clients.Clients) ReceiptsClient {
	return ReceiptsClient{
		clients: clients,
	}
}

func (c ReceiptsClient) IssueReceipt(ctx context.Context, ticketRequest IssueReceiptPayload) error {

	body := receipts.PutReceiptsJSONRequestBody{
		TicketId: ticketRequest.TicketID,
		Price: receipts.Money{
			MoneyAmount:   ticketRequest.Price.Amount,
			MoneyCurrency: ticketRequest.Price.Currency,
		},
	}
	receiptsResp, err := c.clients.Receipts.PutReceiptsWithResponse(ctx, body)
	if err != nil {
		return err
	}
	if receiptsResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %v", receiptsResp.StatusCode())
	}

	return nil
}

type SpreadsheetsClient struct {
	clients *clients.Clients
}

func NewSpreadsheetsClient(clients *clients.Clients) SpreadsheetsClient {
	return SpreadsheetsClient{
		clients: clients,
	}
}

func (c SpreadsheetsClient) AppendRow(ctx context.Context, spreadsheetName string, row []string) error {
	request := spreadsheets.PostSheetsSheetRowsJSONRequestBody{
		Columns: row,
	}

	sheetsResp, err := c.clients.Spreadsheets.PostSheetsSheetRowsWithResponse(ctx, spreadsheetName, request)
	if err != nil {
		return err
	}
	if sheetsResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %v", sheetsResp.StatusCode())
	}

	return nil
}

func NewWorker() *Worker {
	return &Worker{
		queue: make(chan Message, 100),
	}
}

func (w *Worker) SubIssueReceipt(sub message.Subscriber, client *clients.Clients, queue string, r *message.Router) {
	receiptsClient := NewReceiptsClient(client)
	r.AddNoPublisherHandler(
		"issue-queue",
		queue,
		sub,
		func(msg *message.Message) error {
			var ticket Ticket
			err := json.Unmarshal(msg.Payload, &ticket)
			if err != nil {
				return err
			}

			issueReceiptsPayload := IssueReceiptPayload{
				TicketID: ticket.ID,
				Price:    ticket.Price,
			}
			ctx := msg.Context()
			err = receiptsClient.IssueReceipt(ctx, issueReceiptsPayload)
			if err != nil {
				return err
			}
			return nil
		},
	)
}

func (w *Worker) SubSpreedSheetReceipt(sub message.Subscriber, clients *clients.Clients, queue string, r *message.Router, sheet string) {
	spreadsheetsClient := NewSpreadsheetsClient(clients)

	r.AddNoPublisherHandler(
		queue,
		queue,
		sub,
		func(msg *message.Message) error {
			var ticket Ticket

			err := json.Unmarshal(msg.Payload, &ticket)
			if err != nil {
				return err
			}

			sheetPayload := AppendToTrackerPayload{
				CustomerEmail: ticket.CustomerEmail,
				TicketID:      ticket.ID,
				Price:         ticket.Price,
			}

			ctx := msg.Context()

			err = spreadsheetsClient.AppendRow(ctx, sheet, []string{sheetPayload.TicketID, sheetPayload.CustomerEmail, sheetPayload.Price.Amount, sheetPayload.Price.Currency})
			if err != nil {
				return err
			}

			return nil
		})
}

func NewPublisher(pub message.Publisher) Publisher {
	return Publisher{
		pub: pub,
	}
}

func (p *Publisher) PublishBookingConfirmed(payload Ticket, correlationID string) error {
	h := NewHeader()
	payload.Header = h

	msg, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msgSend := message.NewMessage(watermill.NewUUID(), msg)
	msgSend.Metadata.Set("correlation_id", correlationID)

	return p.pub.Publish("TicketBookingConfirmed", msgSend)
}

func (p *Publisher) PublishBookingCanceled(payload Ticket, correlationID string) error {
	h := NewHeader()
	payload.Header = h

	msg, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msgSend := message.NewMessage(watermill.NewUUID(), msg)
	msgSend.Metadata.Set("correlation_id", correlationID)
	return p.pub.Publish("TicketBookingCanceled", msgSend)
}

func NewHeader() Header {
	return Header{
		ID:          uuid.NewString(),
		PublishedAt: time.Now().Format(time.RFC3339),
	}
}

func LogMessage(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		logger := log.FromContext(msg.Context())
		correlationID := log.CorrelationIDFromContext(msg.Context())
		logger.Info("Handling a message")
		logger = logger.WithField("message_uuid", correlationID)
		msgs, err := next(msg)
		if err != nil {
			logger.WithField("error", err.Error()).Error("Message handling error")
		}

		return msgs, nil
	}
}

func PropagateCorrelationID(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		correlationID := msg.Metadata.Get("correlation_id")
		if correlationID == "" {
			correlationID = shortuuid.New()
		}

		ctx := log.ToContext(msg.Context(), logrus.WithFields(logrus.Fields{"correlation_id": correlationID}))
		ctx = log.ContextWithCorrelationID(ctx, correlationID)

		msg.SetContext(ctx)
		return next(msg)
	}
}
