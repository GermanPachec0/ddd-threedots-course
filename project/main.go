package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/receipts"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/spreadsheets"
	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type Task int

const (
	TaskIssueReceipt Task = iota
	TaskAppendToTracker
)

type TicketsConfirmationRequest struct {
	Tickets []string `json:"tickets"`
}
type Worker struct {
	queue chan Message
}

type Message struct {
	Task     Task
	TicketID string
}

func main() {
	log.Init(logrus.InfoLevel)

	e := commonHTTP.NewEcho()
	worker := NewWorker()
	go worker.Run()

	e.POST("/tickets-confirmation", func(c echo.Context) error {
		var request TicketsConfirmationRequest
		err := c.Bind(&request)
		if err != nil {
			return err
		}

		for _, ticket := range request.Tickets {
			worker.Send(Message{
				Task:     TaskIssueReceipt,
				TicketID: ticket,
			})
			worker.Send(Message{
				Task:     TaskAppendToTracker,
				TicketID: ticket,
			})
		}

		return c.NoContent(http.StatusOK)
	})

	logrus.Info("Server starting...")

	err := e.Start(":8080")
	if err != nil && err != http.ErrServerClosed {
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

func (c ReceiptsClient) IssueReceipt(ctx context.Context, ticketID string) error {
	body := receipts.PutReceiptsJSONRequestBody{
		TicketId: ticketID,
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

func (w *Worker) Run() {
	clients, err := clients.NewClients(os.Getenv("GATEWAY_ADDR"), nil)
	if err != nil {
		panic(err)
	}

	receiptsClient := NewReceiptsClient(clients)
	spreadsheetsClient := NewSpreadsheetsClient(clients)
	for msg := range w.queue {
		switch msg.Task {
		//Send receipt to user
		case TaskIssueReceipt:
			//Send receipt to sheet
			err = receiptsClient.IssueReceipt(context.Background(), msg.TicketID)
			if err != nil {
				w.Send(msg)
			}
		case TaskAppendToTracker:
			err = spreadsheetsClient.AppendRow(context.Background(), "tickets-to-print", []string{msg.TicketID})
			if err != nil {
				w.Send(msg)
			}
		}
	}
}

func (w *Worker) Send(msg ...Message) {
	for _, m := range msg {
		w.queue <- m
	}
}
