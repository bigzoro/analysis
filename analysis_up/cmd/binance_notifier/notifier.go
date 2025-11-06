package main

import (
	"analysis/internal/db"
	"context"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"github.com/keighl/postmark"
	"gorm.io/gorm"
)

// Watch defines a single Binance catalog to monitor.
type Watch struct {
	Name      string
	CatalogID int
	Keywords  []string
}

// PostmarkConfig stores email delivery configuration.
type PostmarkConfig struct {
	ServerToken  string
	AccountToken string
	Stream       string
	From         string
	To           []string
}

// Announcement represents a normalized Binance announcement entry.
type Announcement struct {
	Code       string
	Title      string
	URL        string
	ReleasedAt time.Time
	CatalogID  string
}

// Notifier polls Binance announcements and sends notifications for new entries.
type Notifier struct {
	db       *gorm.DB
	fetcher  *AnnouncementFetcher
	watches  []Watch
	postmark PostmarkConfig
	loc      *time.Location
}

// NewNotifier constructs a Notifier instance.
func NewNotifier(gdb *gorm.DB, fetcher *AnnouncementFetcher, watches []Watch, pm PostmarkConfig, loc *time.Location) *Notifier {
	if loc == nil {
		loc = time.Local
	}
	return &Notifier{
		db:       gdb,
		fetcher:  fetcher,
		watches:  watches,
		postmark: pm,
		loc:      loc,
	}
}

// Prime performs an initial sync without emitting notifications.
func (n *Notifier) Prime(ctx context.Context) error {
	return n.process(ctx, true)
}

// Tick performs one polling cycle.
func (n *Notifier) Tick(ctx context.Context) error {
	return n.process(ctx, false)
}

func (n *Notifier) process(ctx context.Context, skipEmail bool) error {
	if n.fetcher == nil || n.db == nil {
		return fmt.Errorf("notifier not initialized")
	}

	newItems := make(map[string][]Announcement)
	for _, watch := range n.watches {
		articles, err := n.fetcher.Fetch(ctx, watch.CatalogID)
		if err != nil {
			log.Printf("[binance_notifier] fetch catalog=%d error: %v", watch.CatalogID, err)
			continue
		}

		for _, art := range articles {
			if len(watch.Keywords) > 0 && !containsKeyword(art.Title, watch.Keywords) {
				continue
			}

			record := &db.BinanceAnnouncement{
				Type:       watch.Name,
				Code:       art.Code,
				Title:      art.Title,
				URL:        art.URL,
				ReleasedAt: art.ReleasedAt.UTC(),
			}

			inserted, err := db.InsertBinanceAnnouncement(n.db, record)
			if err != nil {
				log.Printf("[binance_notifier] insert announcement failed: %v", err)
				continue
			}
			if inserted {
				log.Printf("[binance_notifier] new announcement captured type=%s code=%s title=%s", watch.Name, art.Code, art.Title)
				newItems[watch.Name] = append(newItems[watch.Name], art)
			}
		}
	}

	if skipEmail || len(newItems) == 0 {
		return nil
	}

	return n.sendEmail(newItems)
}

func (n *Notifier) sendEmail(items map[string][]Announcement) error {
	if n.postmark.ServerToken == "" || n.postmark.From == "" || len(n.postmark.To) == 0 {
		return fmt.Errorf("postmark configuration incomplete")
	}

	order := make([]string, 0, len(n.watches))
	seen := make(map[string]struct{})
	for _, w := range n.watches {
		if _, ok := seen[w.Name]; ok {
			continue
		}
		order = append(order, w.Name)
		seen[w.Name] = struct{}{}
	}

	total := 0
	for _, list := range items {
		total += len(list)
	}
	subject := fmt.Sprintf("[Binance Notify] %d new announcement(s)", total)

	var textBody strings.Builder
	var htmlBody strings.Builder

	textBody.WriteString("New Binance announcements detected:\n\n")
	htmlBody.WriteString("<p>New Binance announcements detected:</p>")

	for _, name := range order {
		list := items[name]
		if len(list) == 0 {
			continue
		}

		upper := strings.ToUpper(name)
		textBody.WriteString(fmt.Sprintf("[%s]\n", upper))
		htmlBody.WriteString(fmt.Sprintf("<h3>%s</h3><ul>", html.EscapeString(upper)))

		for _, art := range list {
			when := art.ReleasedAt.In(n.loc).Format("2006-01-02 15:04 MST")
			textBody.WriteString(fmt.Sprintf("- %s (%s)\n  %s\n", art.Title, when, art.URL))

			link := html.EscapeString(art.URL)
			title := html.EscapeString(art.Title)
			htmlBody.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a> <em>(%s)</em></li>", link, title, html.EscapeString(when)))
		}
		textBody.WriteString("\n")
		htmlBody.WriteString("</ul>")
	}

	email := postmark.Email{
		From:          n.postmark.From,
		To:            strings.Join(n.postmark.To, ","),
		Subject:       subject,
		TextBody:      textBody.String(),
		HtmlBody:      htmlBody.String(),
		MessageStream: n.postmark.Stream,
	}

	client := postmark.NewClient(n.postmark.ServerToken, n.postmark.AccountToken)
	if _, err := client.SendEmail(email); err != nil {
		return fmt.Errorf("send postmark email: %w", err)
	}

	log.Printf("[binance_notifier] notification sent: total=%d", total)
	return nil
}

func containsKeyword(title string, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}
	lower := strings.ToLower(title)
	for _, kw := range keywords {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw == "" {
			continue
		}
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
