package mail

import "strings"

// GuessedHosts is IMAP/SMTP guessed from an email domain.
type GuessedHosts struct {
	IMAP     string `json:"imap"`
	SMTP     string `json:"smtp"`
	Provider string `json:"provider,omitempty"` // google, microsoft, yahoo, icloud, fastmail, proton, ""
	AuthHint string `json:"authHint,omitempty"`
}

// GuessMailHosts fills IMAP/SMTP from the address domain (first-run polish).
func GuessMailHosts(address string) GuessedHosts {
	addr := canonAddr(address)
	_, domain, ok := strings.Cut(addr, "@")
	if !ok || domain == "" {
		return GuessedHosts{}
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	switch domain {
	case "gmail.com", "googlemail.com":
		return GuessedHosts{
			IMAP: "imap.gmail.com:993", SMTP: "smtp.gmail.com:465",
			Provider: "google", AuthHint: "Google OAuth or an App Password",
		}
	case "outlook.com", "hotmail.com", "live.com", "msn.com", "office365.com":
		return GuessedHosts{
			IMAP: "outlook.office365.com:993", SMTP: "smtp.office365.com:587",
			Provider: "microsoft", AuthHint: "Microsoft OAuth or an App Password",
		}
	case "yahoo.com", "ymail.com":
		return GuessedHosts{
			IMAP: "imap.mail.yahoo.com:993", SMTP: "smtp.mail.yahoo.com:587",
			Provider: "yahoo", AuthHint: "Yahoo App Password",
		}
	case "icloud.com", "me.com", "mac.com":
		return GuessedHosts{
			IMAP: "imap.mail.me.com:993", SMTP: "smtp.mail.me.com:587",
			Provider: "icloud", AuthHint: "Apple App-Specific Password",
		}
	case "fastmail.com", "fastmail.fm":
		return GuessedHosts{
			IMAP: "imap.fastmail.com:993", SMTP: "smtp.fastmail.com:465",
			Provider: "fastmail", AuthHint: "Fastmail App Password",
		}
	case "proton.me", "protonmail.com":
		return GuessedHosts{
			IMAP: "imap.proton.me:993", SMTP: "smtp.proton.me:587",
			Provider: "proton", AuthHint: "Proton Bridge / app password (IMAP Bridge)",
		}
	}
	// Microsoft 365 custom domains often use outlook.office365.com; we cannot
	// know without MX. Fall back to imap./smtp. of the domain.
	host := domain
	return GuessedHosts{
		IMAP: "imap." + host + ":993",
		SMTP: "smtp." + host + ":587",
		AuthHint: "Type an app password, or OAuth if the host is Google/Microsoft",
	}
}

func providerForAddress(address string) string {
	return GuessMailHosts(address).Provider
}
