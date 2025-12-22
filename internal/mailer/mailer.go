package mailer

import "embed"

const (
	FromName            = "VladimirSocial"
	maxRetries          = 3
	UserWelcomeTemplate = "user_invitation.tmpl"
)

//go:embed "templates"
var FS embed.FS

// ClientManager он регулятор, похуй, главное чтобы частные случаи (SendGrid, MailDropMailer) реализовывали интерфейс Send
type Client interface {
	Send(templateFile string, username, email string, data interface{}, isSandbox bool) error // isSandbox служит флагом чтобы не отправлять письма на тестовом окружении
}
