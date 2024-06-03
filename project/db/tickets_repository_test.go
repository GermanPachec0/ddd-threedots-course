package db

import (
	"context"
	"os"
	"sync"
	"testing"
	"tickets/entities"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

var db *sqlx.DB
var getDbOnce sync.Once

func getDb() *sqlx.DB {
	getDbOnce.Do(func() {
		var err error
		db, err = sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
		if err != nil {
			panic(err)
		}
	})
	return db
}

func TestAddTicket(t *testing.T) {
	dbconn := getDb()
	db := DB{Conn: dbconn}
	db.MigrateSchema()
	ticketRepo := NewTicketRepo(&db)
	ctx := context.Background()

	ticket1 := entities.Ticket{
		TicketID: "2741c2ee-a9a4-4435-9a80-4d8181d3d0fb",
		Price: entities.Money{
			Amount:   "123",
			Currency: "USD",
		},
		CustomerEmail: "pepe@gmail.com",
	}
	err := ticketRepo.Create(ctx, ticket1)
	assert.NoError(t, err)

	err = ticketRepo.Create(ctx, ticket1)
	assert.NoError(t, err)

	tickets, err := ticketRepo.Get(ctx)
	assert.NoError(t, err)

	assert.Equal(t, len(tickets), 1)

}
