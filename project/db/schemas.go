package db

var schema = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS tickets (
	ticket_id UUID PRIMARY KEY,
	price_amount NUMERIC(10, 2) NOT NULL,
	price_currency CHAR(3) NOT NULL,
	customer_email VARCHAR(255) NOT NULL,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE shows (
    show_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    dead_nation_id UUID NOT NULL,
    number_of_tickets INT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    title VARCHAR(255) NOT NULL,
    venue VARCHAR(255) NOT NULL
);
CREATE TABLE bookings (
	booking_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    show_id UUID,
    number_of_tickets INT NOT NULL,
    customer_email VARCHAR(255) NOT NULL
);
CREATE TABLE IF NOT EXISTS read_model_ops_bookings (
    booking_id UUID PRIMARY KEY,
    payload JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
    event_id UUID PRIMARY KEY,
    published_at TIMESTAMP NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    event_payload JSONB NOT NULL
);

`
