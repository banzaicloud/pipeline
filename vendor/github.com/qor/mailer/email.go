package mailer

import "net/mail"

// Email email struct
type Email struct {
	TO, CC, BCC   []mail.Address
	From, ReplyTo *mail.Address
	Subject       string
	Headers       mail.Header
	Attachments   []Attachment
	Text          string // text email content
	HTML          string // html email content
}

// Attachment attachment struct
type Attachment struct {
	FileName string
	Inline   bool
}

// Merge merge email struct and create a new one
func (email Email) Merge(e Email) Email {
	if len(e.TO) > 0 {
		email.TO = e.TO
	}

	if len(e.CC) > 0 {
		email.CC = e.CC
	}

	if len(e.BCC) > 0 {
		email.BCC = e.BCC
	}

	if e.From != nil {
		email.From = e.From
	}

	if e.ReplyTo != nil {
		email.ReplyTo = e.ReplyTo
	}

	if e.Subject != "" {
		email.Subject = e.Subject
	}

	if e.Headers != nil {
		newHeaders := mail.Header{}

		for k, v := range email.Headers {
			newHeaders[k] = v
		}

		for k, v := range e.Headers {
			newHeaders[k] = v
		}

		email.Headers = newHeaders
	}

	if len(e.Attachments) > 0 {
		email.Attachments = e.Attachments
	}

	if e.Text != "" {
		email.Text = e.Text
	}

	if e.HTML != "" {
		email.HTML = e.HTML
	}

	return email
}
