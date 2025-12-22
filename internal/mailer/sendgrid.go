package mailer

import (
	"bytes"
	"fmt"
	"log"
	"text/template"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type SendGridMailer struct {
	fromEmail string // откуда отправляем письма
	apiKey    string
	client    *sendgrid.Client
}

func New(fromEmail string, apiKey string) *SendGridMailer {
	return &SendGridMailer{
		fromEmail: fromEmail,
		apiKey:    apiKey,
		client:    sendgrid.NewSendClient(apiKey),
	}
}

// Отправляет на почту email исходя из нашего template файла
func (s *SendGridMailer) Send(templateFile string, username, email string, data interface{}, isSandbox bool) error {
	from := mail.NewEmail(FromName, s.fromEmail)
	to := mail.NewEmail(username, email)

	// template parsing
	tmlp, err := template.ParseFS(FS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	err = tmlp.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return nil
	}

	body := new(bytes.Buffer)
	err = tmlp.ExecuteTemplate(body, "body", data)
	if err != nil {
		return nil
	}

	message := mail.NewSingleEmail(from, subject.String(), to, "", body.String())

	// check env, if sandbox == "dev" => true
	message.SetMailSettings(&mail.MailSettings{
		SandboxMode: &mail.Setting{
			Enable: &isSandbox,
		},
	})

	// попытки отправить письмо
	for i := 0; i < maxRetries; i++ {
		res, err := s.client.Send(message)
		if err != nil {
			log.Printf("failed to send email to %v, attempt %d of %d", email, i+1, maxRetries)
			log.Printf("Error: %v", err.Error())

			// exponential backoff
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		log.Printf("Email to %v was sent with status code %v", email, res.StatusCode)
		return nil
	}

	return fmt.Errorf("failed to send email after %d attempts", maxRetries)
}
