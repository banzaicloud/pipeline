package logger

import (
	"bytes"
	"fmt"
	"net/mail"
	"strings"

	"github.com/qor/mailer"
	gomail "gopkg.in/gomail.v2"
)

// Sender gomail struct
type Sender struct {
	*Config
}

// Config gomail config
type Config struct {
	Sender gomail.Sender
}

// New initalize gomail sender with gomail.Dailer
func New(config *Config) *Sender {
	if config == nil {
		config = &Config{}
	}

	return &Sender{Config: config}
}

// Send send email with GoMail
func (sender *Sender) Send(email mailer.Email) error {
	var result bytes.Buffer

	formatAddress := func(key string, addresses []mail.Address) {
		var emails []string

		if len(addresses) > 0 {
			result.WriteString(fmt.Sprintf("%v: ", key))

			for _, address := range addresses {
				emails = append(emails, address.String())
			}

			result.WriteString(strings.Join(emails, ", ") + "\n")
		}
	}

	formatAddress("TO", email.TO)
	formatAddress("CC", email.CC)
	formatAddress("BCC", email.BCC)

	if email.From != nil {
		formatAddress("From", []mail.Address{*email.From})
	}

	if email.ReplyTo != nil {
		formatAddress("ReplyTO", []mail.Address{*email.ReplyTo})
	}

	if email.Subject != "" {
		result.WriteString(fmt.Sprintf("Subject: %v\n", email.Subject))
	}

	if email.Headers != nil {
		for key, value := range email.Headers {
			result.WriteString(fmt.Sprintf("%v: %v\n", key, value))
		}
	}

	for _, attachment := range email.Attachments {
		if attachment.Inline {
			result.WriteString(fmt.Sprintf("\nContent-Disposition: inline; filename=\"%v\"\n\n", attachment.FileName))
		} else {
			result.WriteString(fmt.Sprintf("\nContent-Disposition: attachment; filename=\"%v\"\n\n", attachment.FileName))
		}
	}

	if email.Text != "" {
		result.WriteString(fmt.Sprintf("\nContent-Type: text/plain; charset=UTF-8\n%v\n", email.Text))
	}

	if email.HTML != "" {
		result.WriteString(fmt.Sprintf("\nContent-Type: text/html; charset=UTF-8\n%v\n", email.HTML))
	}

	fmt.Println(result.String())
	return nil
}
