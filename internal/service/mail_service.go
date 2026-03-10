// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"log"
	"sessiondb/internal/config"
)

// MailService sends transactional email (e.g. credentials on user create). No-op when config.Mail.Enabled is false.
type MailService struct {
	Config *config.Config
}

// NewMailService returns a MailService. When Mail.Enabled is false, SendCredentialsEmail does nothing.
func NewMailService(cfg *config.Config) *MailService {
	return &MailService{Config: cfg}
}

// SendCredentialsEmail sends login details to the user. When mail is disabled, logs and returns nil.
// Do not log tempPassword. When enabled, implement SMTP send here (e.g. net/smtp or a third-party client).
func (s *MailService) SendCredentialsEmail(to, userName, tempPassword, loginURL string) error {
	if s.Config == nil || !s.Config.Mail.Enabled {
		log.Printf("Mail disabled: would send credentials to %q for user %q (login URL: %s)", to, userName, loginURL)
		return nil
	}
	// TODO: send via SMTP using s.Config.Mail (From, SMTPHost, SMTPPort, SMTPUser, SMTPPass)
	// For now log only; do not log tempPassword.
	log.Printf("Mail enabled: sending credentials to %q for user %q", to, userName)
	return nil
}
