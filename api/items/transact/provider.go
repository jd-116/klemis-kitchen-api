package transact

import (
	"context"
	"log"
	"time"

	"github.com/hako/durafmt"

	"github.com/jd-116/klemis-kitchen-api/util"
)

// Bundles together a stateful item provider via the Transact API,
// including an active session with the API
// and a cache in front of it
type Provider struct {
	Scraper             *Scraper
	stopFetch           chan struct{}
	stopReloadSession   chan struct{}
	fetchPeriod         time.Duration
	reloadSessionPeriod time.Duration
}

// Loads values from the environment
// and creates the provider
// (doesn't involve authentication or start goroutines)
func NewProvider() (*Provider, error) {
	baseUrl, err := util.GetEnv("Transact base URL", "TRANSACT_BASE_URL")
	if err != nil {
		return nil, err
	}

	tenant, err := util.GetEnv("Transact tenant", "TRANSACT_TENANT")
	if err != nil {
		return nil, err
	}

	username, err := util.GetEnv("Transact username", "TRANSACT_USERNAME")
	if err != nil {
		return nil, err
	}

	password, err := util.GetEnv("Transact password", "TRANSACT_PASSWORD")
	if err != nil {
		return nil, err
	}

	fetchPeriod, err := util.GetDurationEnv("Transact API fetch period", "TRANSACT_FETCH_PERIOD")
	if err != nil {
		return nil, err
	}

	reloadSessionPeriod, err := util.GetDurationEnv("Transact API reload session period", "TRANSACT_RELOAD_SESSION_PERIOD")
	if err != nil {
		return nil, err
	}

	// Create the scraper
	scraper, err := NewScraper(baseUrl, tenant, username, password)
	if err != nil {
		return nil, err
	}

	return &Provider{
		Scraper:             scraper,
		stopFetch:           make(chan struct{}),
		stopReloadSession:   make(chan struct{}),
		fetchPeriod:         fetchPeriod,
		reloadSessionPeriod: reloadSessionPeriod,
	}, nil
}

// Initializes the authentication
// and starts goroutines to periodically re-authenticate/fetch
func (p *Provider) Connect(ctx context.Context) error {
	// Load the session
	err := p.Scraper.ReloadSession()
	if err != nil {
		return err
	}

	// Start the periodic goroutines
	go p.periodFetch()
	go p.periodReloadSession()

	return nil
}

// Periodically fetches from the API
// and stores the data into the cache
func (p *Provider) periodFetch() {
	for {
		select {
		case <-p.stopFetch:
			return
		case <-time.After(p.fetchPeriod):
			// TODO fetch
		}
	}
}

// Periodically reloads the session
func (p *Provider) periodReloadSession() {
	humanDuration := durafmt.Parse(p.reloadSessionPeriod).LimitFirstN(2).String()
	log.Printf("reloading Transact API session in %s", humanDuration)
	for {
		select {
		case <-p.stopReloadSession:
			return
		case <-time.After(p.reloadSessionPeriod):
			err := p.Scraper.ReloadSession()
			if err != nil {
				// Report error,
				// but continue the goroutine
				log.Println("an error occurred while reloading Transact API session:")
				log.Println(err)
			} else {
				log.Printf("reloaded Transact API session for version %s; reloading again in %s\n",
					p.Scraper.ClientVersion, humanDuration)
			}
		}
	}
}

// Stops all periodic goroutines
// (for re-authentication and fetching)
func (p *Provider) Disconnect(ctx context.Context) error {
	p.stopFetch <- struct{}{}
	p.stopReloadSession <- struct{}{}

	return nil
}
