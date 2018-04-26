package mailer_test

import (
	"fmt"
	"net/mail"
	"testing"

	"github.com/qor/mailer"
)

func equalCheck(email1, email2 mailer.Email) error {
	for i, e := range email1.TO {
		e2 := email2.TO[i]
		if e.String() != e2.String() {
			return fmt.Errorf("TO email address should be same, but got %v, %v", e.String(), e2.String())
		}
	}

	for i, e := range email1.CC {
		e2 := email2.CC[i]
		if e.String() != e2.String() {
			return fmt.Errorf("CC email address should be same, but got %v, %v", e.String(), e2.String())
		}
	}

	for i, e := range email1.BCC {
		e2 := email2.BCC[i]
		if e.String() != e2.String() {
			return fmt.Errorf("BCC email address should be same, but got %v, %v", e.String(), e2.String())
		}
	}

	if email1.From != email2.From {
		if email1.From == nil || email2.From == nil {
			return fmt.Errorf("From should be same, but got %+v, %+v", email1.From, email2.From)
		}

		if email1.From.String() != email2.From.String() {
			return fmt.Errorf("From should be same, but got %+v, %+v", email1.From, email2.From)
		}
	}

	if email1.ReplyTo != email2.ReplyTo {
		if email1.ReplyTo == nil || email2.ReplyTo == nil {
			return fmt.Errorf("ReplyTo should be same, but got %+v, %+v", email1.ReplyTo, email2.ReplyTo)
		}

		if email1.ReplyTo.String() != email2.ReplyTo.String() {
			return fmt.Errorf("ReplyTo should be same, but got %+v, %+v", email1.ReplyTo, email2.ReplyTo)
		}
	}

	if email1.Subject != email2.Subject {
		return fmt.Errorf("Email's Subject should be same, but got %v, %v", email1.Subject, email2.Subject)
	}

	if len(email1.Headers) != len(email2.Headers) {
		return fmt.Errorf("Email's Header should be same, but got %v, %v", email1.Headers, email2.Headers)
	}

	for k, v := range email1.Headers {
		if fmt.Sprint(v) != fmt.Sprint(email2.Headers[k]) {
			return fmt.Errorf("Email's Header should be same, but got %v, %v", email1.Headers, email2.Headers)
		}
	}

	if len(email1.Attachments) != len(email2.Attachments) {
		return fmt.Errorf("Email's Attachments should be same, but got %v, %v", email1.Attachments, email2.Attachments)
	}

	for i, attachment := range email1.Attachments {
		if fmt.Sprint(attachment) != fmt.Sprint(email2.Attachments[i]) {
			return fmt.Errorf("Email's attachment should be same, but got %v, %v", email1.Attachments, email2.Attachments)
		}
	}

	// if email1.Template != email2.Template {
	// 	return fmt.Errorf("Email's Template should be same, but got %v, %v", email1.Template, email2.Template)
	// }

	// if email1.Layout != email2.Layout {
	// 	return fmt.Errorf("Email's Layout should be same, but got %v, %v", email1.Layout, email2.Layout)
	// }

	if email1.Text != email2.Text {
		return fmt.Errorf("Email's Text should be same, but got %v, %v", email1.Text, email2.Text)
	}

	if email1.HTML != email2.HTML {
		return fmt.Errorf("Email's HTML should be same, but got %v, %v", email1.HTML, email2.HTML)
	}
	return nil
}

func TestEmailMerge(t *testing.T) {
	email1 := mailer.Email{
		TO:   []mail.Address{{Address: "to1@example.org"}},
		CC:   []mail.Address{},
		BCC:  []mail.Address{{Address: "bcc1@example.org"}},
		From: &mail.Address{Address: "from1@example.org"},
		// ReplyTo:     &mail.Address{},
		Subject: "subject",
		Headers: mail.Header{"Key1": []string{"Value1"}},
		Attachments: []mailer.Attachment{
			{
				FileName: "logo.png",
				Inline:   true,
			},
		},
		// Template: "template1",
		// Layout:      "layout",
		Text: "text1",
	}
	email1Clone := email1

	email2 := mailer.Email{
		// TO:  []mail.Address{{Address: "to2@example.org"}},
		CC: []mail.Address{{Address: "cc2@example.org"}},
		// BCC: []mail.Address{{Address: "bcc2@example.org"}},
		// From: &mail.Address{Address: "from2@example.org"},
		ReplyTo: &mail.Address{Address: "reply2@example.org"},
		// Subject: "subject",
		Headers: mail.Header{"Key2": []string{"Value2"}},
		Attachments: []mailer.Attachment{
			{
				FileName: "logo2.png",
				Inline:   true,
			},
		},
		// Template: "template2",
		// Layout: "layout2",
		HTML: "html2",
	}
	email2Clone := email2

	email3 := email1.Merge(email2)

	if err := equalCheck(email1, email1Clone); err != nil {
		t.Errorf("Email should not be changed when use Merge, got %v", err)
	}

	if err := equalCheck(email2, email2Clone); err != nil {
		t.Errorf("Email should not be changed when use Mergei, got %v", err)
	}

	generatedEmail := mailer.Email{
		TO:      []mail.Address{{Address: "to1@example.org"}},
		CC:      []mail.Address{{Address: "cc2@example.org"}},
		BCC:     []mail.Address{{Address: "bcc1@example.org"}},
		From:    &mail.Address{Address: "from1@example.org"},
		ReplyTo: &mail.Address{Address: "reply2@example.org"},
		Subject: "subject",
		Headers: mail.Header{"Key1": []string{"Value1"}, "Key2": []string{"Value2"}},
		Attachments: []mailer.Attachment{
			{
				FileName: "logo2.png",
				Inline:   true,
			},
		},
		// Template: "template1",
		// Layout:   "layout2",
		Text: "text1",
		HTML: "html2",
	}
	if err := equalCheck(email3, generatedEmail); err != nil {
		t.Errorf("Generated email with Merge should be correct, but got %v", err)
	}
}
