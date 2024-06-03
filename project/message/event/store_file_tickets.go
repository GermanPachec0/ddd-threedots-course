package event

import (
	"context"
	"fmt"
	"tickets/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

func (h Handler) StoreTicketsInFile(ctx context.Context, event *entities.TicketBookingConfirmed) error {
	log.FromContext(ctx).Info("Printing ticket")

	ticketHTML := `
		<html>
			<head>
				<title>Ticket</title>
			</head>
			<body>
				<h1>Ticket ` + event.TicketID + `</h1>
				<p>Price: ` + event.Price.Amount + ` ` + event.Price.Currency + `</p>	
			</body>
		</html>
`

	ticketFile := event.TicketID + "-ticket.html"

	err := h.fileService.StoreFile(ctx, ticketFile, ticketHTML)
	if err != nil {
		return fmt.Errorf("failed to upload ticket file: %w", err)
	}

	ticketPrintedEvent := entities.TicketPrinted{
		Header:   entities.NewEventHeader(),
		TicketID: event.TicketID,
		FileName: ticketFile,
	}

	return h.eventBus.Publish(ctx, ticketPrintedEvent)
}
