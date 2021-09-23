package transact

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/rs/zerolog"

	"github.com/jd-116/klemis-kitchen-api/env"
	"github.com/jd-116/klemis-kitchen-api/products"
)

// Provider bundles together a stateful item provider via the Transact API,
// including an active session with the API
// and a cache in front of it
//
// Safe to copy and keep multiple references
type Provider struct {
	stopFetch         chan struct{}
	stopReloadSession chan struct{}

	// Config values
	fetchPeriod                   time.Duration
	reloadSessionPeriod           time.Duration
	csvReportName                 string
	reportPollPeriod              time.Duration
	reportPollTimeout             time.Duration
	csvReportIDColumnOffset       int
	csvReportNameColumnOffset     int
	csvReportQuantityColumnOffset int
	profitCenterPrefix            string
	reportType                    string

	*Scraper
	*products.Cache
	logger zerolog.Logger
}

// NewProvider loads values from the environment
// and creates the provider
// (doesn't involve authentication or start goroutines)
func NewProvider(logger zerolog.Logger) (*Provider, error) {
	baseURL, err := env.GetEnv("Transact base URL", "TRANSACT_BASE_URL")
	if err != nil {
		return nil, err
	}

	tenant, err := env.GetEnv("Transact tenant", "TRANSACT_TENANT")
	if err != nil {
		return nil, err
	}

	username, err := env.GetEnv("Transact username", "TRANSACT_USERNAME")
	if err != nil {
		return nil, err
	}

	password, err := env.GetEnv("Transact password", "TRANSACT_PASSWORD")
	if err != nil {
		return nil, err
	}

	fetchPeriod, err := env.GetDurationEnv("Transact API fetch period", "TRANSACT_FETCH_PERIOD")
	if err != nil {
		return nil, err
	}

	reloadSessionPeriod, err := env.GetDurationEnv("Transact API reload session period", "TRANSACT_RELOAD_SESSION_PERIOD")
	if err != nil {
		return nil, err
	}

	csvReportName, err := env.GetEnv("Transact CSV favorite report name", "TRANSACT_CSV_FAVORITE_REPORT_NAME")
	if err != nil {
		return nil, err
	}

	reportPollPeriod, err := env.GetDurationEnv("Transact API report poll period", "TRANSACT_REPORT_POLL_PERIOD")
	if err != nil {
		return nil, err
	}

	reportPollTimeout, err := env.GetDurationEnv("Transact API report poll timeout", "TRANSACT_REPORT_POLL_TIMEOUT")
	if err != nil {
		return nil, err
	}

	csvReportIDColumnOffset, err := env.GetIntEnv("Transact CSV report ID column offset from 'Profit Center - '", "TRANSACT_CSV_REPORT_ID_COLUMN_OFFSET")
	if err != nil {
		return nil, err
	}

	csvReportNameColumnOffset, err := env.GetIntEnv("Transact CSV report name column offset from 'Profit Center - '", "TRANSACT_CSV_REPORT_NAME_COLUMN_OFFSET")
	if err != nil {
		return nil, err
	}

	csvReportQuantityColumnOffset, err := env.GetIntEnv("Transact CSV report quantity column offset from 'Profit Center - '", "TRANSACT_CSV_REPORT_QTY_COLUMN_OFFSET")
	if err != nil {
		return nil, err
	}

	profitCenterPrefix, err := env.GetEnv("Transact CSV report profit center prefix", "TRANSACT_PROFIT_CENTER_PREFIX")
	if err != nil {
		return nil, err
	}

	reportType, err := env.GetEnv("Transact CSV report __type value", "TRANSACT_CSV_REPORT_TYPE")
	if err != nil {
		return nil, err
	}

	// Create the scraper
	scraper, err := NewScraper(baseURL, tenant, username, password, logger)
	if err != nil {
		return nil, err
	}

	return &Provider{
		stopFetch:         make(chan struct{}),
		stopReloadSession: make(chan struct{}),

		fetchPeriod:                   fetchPeriod,
		reloadSessionPeriod:           reloadSessionPeriod,
		csvReportName:                 csvReportName,
		reportPollPeriod:              reportPollPeriod,
		reportPollTimeout:             reportPollTimeout,
		csvReportIDColumnOffset:       csvReportIDColumnOffset,
		csvReportNameColumnOffset:     csvReportNameColumnOffset,
		csvReportQuantityColumnOffset: csvReportQuantityColumnOffset,
		profitCenterPrefix:            profitCenterPrefix,
		reportType:                    reportType,

		Scraper: scraper,
		Cache:   &products.Cache{},
		logger:  logger,
	}, nil
}

// Connect initializes the authentication
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
	humanDuration := durafmt.Parse(p.fetchPeriod).LimitFirstN(2).String()
	p.tryFetch(humanDuration)
	for {
		select {
		case <-p.stopFetch:
			return
		case <-time.After(p.fetchPeriod):
			p.tryFetch(humanDuration)
		}
	}
}

// Attempts to fetch and reload the cache,
// printing out an error if it occurs
func (p *Provider) tryFetch(delayUntilNext string) {
	p.logger.
		Info().
		Msg("started to fetch Transact API partial product cache")

	// Fetch a list of partial products from the Transact API via a report
	reportRows, err := p.Scraper.GetInventoryCSV(p.csvReportName,
		p.reportPollPeriod, p.reportPollTimeout, p.reportType)
	if err != nil {
		// Report error,
		// but continue the goroutine
		p.logger.
			Error().
			Err(err).
			Msg("an error occurred while fetching Transact API partial product cache")
		return
	}

	// Parse each CSV row individually -- there are no headers :)
	productsMap := make(map[string]map[string]products.PartialProduct)
	totalLoaded := 0
	for _, csvRow := range reportRows {
		result := p.parseCSVRow(csvRow)
		if result == nil {
			continue
		}

		// Initialize the inner map if needed
		location := result.LocationIdentifier
		if _, ok := productsMap[location]; !ok {
			productsMap[location] = make(map[string]products.PartialProduct)
		}

		// Load the product into the cache
		productsMap[location][result.PartialProduct.ID] = result.PartialProduct
		totalLoaded++
	}

	p.logger.
		Info().
		Int("raw_row_count", len(reportRows)).
		Int("imported_row_count", totalLoaded).
		Str("delay_until_next", delayUntilNext).
		Msg("reloaded Transact API partial product cache")

	// Load the products into the cache
	p.Cache.Load(productsMap)
}

type parseResult struct {
	products.PartialProduct
	LocationIdentifier string
}

func (p *Provider) parseCSVRow(row []string) *parseResult {
	// Scan each cell until it sees the profit center prefix
	for i, cell := range row {
		if strings.HasPrefix(cell, p.profitCenterPrefix) {
			locName := strings.TrimSpace(strings.TrimPrefix(cell, p.profitCenterPrefix))

			// Ensure array accesses are within bounds
			if i+p.csvReportNameColumnOffset >= len(row) ||
				i+p.csvReportIDColumnOffset >= len(row) ||
				i+p.csvReportQuantityColumnOffset >= len(row) {
				return nil
			}

			name := row[i+p.csvReportNameColumnOffset]
			id := row[i+p.csvReportIDColumnOffset]
			if name == "" || id == "" {
				return nil
			}

			amountRaw := row[i+p.csvReportQuantityColumnOffset]
			amount, err := strconv.Atoi(amountRaw)
			if err != nil {
				return nil
			}

			// Don't let negative item amounts get past parsing
			if amount < 0 {
				amount = 0
			}

			product := products.PartialProduct{
				Name:   name,
				ID:     id,
				Amount: amount,
			}

			return &parseResult{
				PartialProduct:     product,
				LocationIdentifier: locName,
			}
		}
	}

	return nil
}

// Periodically reloads the session
func (p *Provider) periodReloadSession() {
	humanDuration := durafmt.Parse(p.reloadSessionPeriod).LimitFirstN(2).String()
	p.logger.
		Info().
		Str("interval", humanDuration).
		Msg("started timer to reload Transact API session")
	for {
		select {
		case <-p.stopReloadSession:
			return
		case <-time.After(p.reloadSessionPeriod):
			err := p.Scraper.ReloadSession()
			if err != nil {
				// Report error,
				// but continue the goroutine
				p.logger.
					Error().
					Err(err).
					Msg("an error occurred while reloading Transact API session")
			} else {
				p.logger.
					Info().
					Str("transact_version", p.Scraper.ClientVersion).
					Str("delay_until_next", humanDuration).
					Msg("reloaded Transact API session")
			}
		}
	}
}

// Disconnect stops all periodic goroutines
// (for re-authentication and fetching)
func (p *Provider) Disconnect(ctx context.Context) error {
	p.stopFetch <- struct{}{}
	p.stopReloadSession <- struct{}{}

	return nil
}
