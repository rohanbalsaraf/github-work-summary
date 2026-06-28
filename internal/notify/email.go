package notify

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/RDX463/github-work-summary/internal/summary"
)

// EmailNotifier sends a summary via SMTP.
type EmailNotifier struct {
	Host     string
	Port     string
	User     string
	Password string
	To       string
}

func (n *EmailNotifier) Send(ctx context.Context, report summary.Report) error {
	subject := fmt.Sprintf("Subject: 🚀 GitHub Work Summary (%s)\n", report.WindowEnd.Format("Jan 02, 2006"))
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	html, err := report.ToHTML()
	if err != nil {
		return err
	}

	msg := []byte(subject + mime + html)
	auth := smtp.PlainAuth("", n.User, n.Password, n.Host)

	addr := fmt.Sprintf("%s:%s", n.Host, n.Port)
	return smtp.SendMail(addr, auth, n.User, []string{n.To}, msg)
}
