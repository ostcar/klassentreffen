package mail

import (
	"fmt"

	"github.com/ostcar/klassentreffen/config"
	"github.com/wneessen/go-mail"
)

func Send(cfg config.Config, to string, text string) error {
	if cfg.Debug {
		return sendMailDebug(to, text)
	}
	// First we create a mail message
	m := mail.NewMsg()
	if err := m.From(cfg.MailFrom); err != nil {
		return fmt.Errorf("set from header: %w", err)
	}

	if err := m.To(to); err != nil {
		return fmt.Errorf("set to header: %w", err)
	}

	m.Subject("Klassentreffen")
	m.SetBodyString(mail.TypeTextPlain, text)

	// Secondly the mail client
	c, err := mail.NewClient("localhost",
		mail.WithPort(25),
		mail.WithSMTPAuth(mail.SMTPAuthNoAuth),
		mail.WithTLSPolicy(mail.NoTLS),
	)
	if err != nil {
		return fmt.Errorf("create mail client: %w", err)
	}

	// Finally let's send out the mail
	if err := c.DialAndSend(m); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}

	return nil
}

func sendMailDebug(to string, text string) error {
	fmt.Printf("To: %s\n", to)
	fmt.Printf("Subject: %s\n", "Klassentreffen")
	fmt.Printf("Body:\n%s\n", text)
	return nil
}
