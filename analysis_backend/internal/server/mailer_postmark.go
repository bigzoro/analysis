package server

import (
	"github.com/keighl/postmark"
)

type Mailer interface {
	Send(subject, htmlBody, textBody string) error
}

type PostmarkMailer struct {
	client *postmark.Client
	from   string
	to     []string
	stream string // e.g. "outbound"
}

func NewPostmarkMailer(serverToken, from string, to []string, stream string) *PostmarkMailer {
	return &PostmarkMailer{
		client: postmark.NewClient(serverToken, ""),
		from:   from,
		to:     to,
		stream: stream,
	}
}

func (m *PostmarkMailer) Send(subject, htmlBody, textBody string) error {
	for _, addr := range m.to {
		_, err := m.client.SendEmail(postmark.Email{
			From:     m.from,
			To:       addr,
			Subject:  subject,
			HtmlBody: htmlBody,
			TextBody: textBody,
			//MessageStream: m.stream,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
