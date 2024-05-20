package main

import "errors"

type MissingInvoiceNumber struct{}

func (m MissingInvoiceNumber) Error() error {
	return errors.New("Ticket Not found")
}

func (m MissingInvoiceNumber) IsPermanent() bool {
	return true
}
