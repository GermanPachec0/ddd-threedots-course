package main

type MissingInvoiceNumber struct{}

func (m MissingInvoiceNumber) Error() string {
	return "missing the invoice number - can't continue"
}

func (m MissingInvoiceNumber) IsPermanent() bool {
	return true
}
