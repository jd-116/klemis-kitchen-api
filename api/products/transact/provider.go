package transact

import (
	"context"
	"log"
	"time"

	"github.com/hako/durafmt"

	"github.com/jd-116/klemis-kitchen-api/products"
	"github.com/jd-116/klemis-kitchen-api/util"
)

// Bundles together a stateful item provider via the Transact API,
// including an active session with the API
// and a cache in front of it
//
// Safe to copy and keep multiple references
type Provider struct {
	stopFetch         chan struct{}
	stopReloadSession chan struct{}

	// Config values
	fetchPeriod         time.Duration
	reloadSessionPeriod time.Duration
	productClassName    string

	*Scraper
	*products.Cache
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

	productClassName, err := util.GetEnv("Transact product class name", "TRANSACT_PRODUCT_CLASS_NAME")
	if err != nil {
		return nil, err
	}

	// Create the scraper
	scraper, err := NewScraper(baseUrl, tenant, username, password)
	if err != nil {
		return nil, err
	}

	return &Provider{
		stopFetch:         make(chan struct{}),
		stopReloadSession: make(chan struct{}),

		fetchPeriod:         fetchPeriod,
		reloadSessionPeriod: reloadSessionPeriod,
		productClassName:    productClassName,

		Scraper: scraper,
		Cache:   &products.Cache{},
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
	// Fetch a list of partial products from the Transact API
	rawPartialProducts, err := p.Scraper.GetItemsForClass(p.productClassName)
	if err != nil {
		// Report error,
		// but continue the goroutine
		log.Println("an error occurred while fetching Transact API partial product cache:")
		log.Println(err)
		return
	}

	// Build the cache map
	productsMap := make(map[string]map[string]products.PartialProduct)
	totalLoaded := 0
	for _, partialProduct := range rawPartialProducts {
		// TODO remove hard-coded location identifier once
		// location matching is implement
		location := "main_quad"

		// Initialize the inner map if needed
		if _, ok := productsMap[location]; !ok {
			productsMap[location] = make(map[string]products.PartialProduct)
		}

		// Construct the partial product by parsing it
		rawName, nameOk := partialProduct["label"]
		rawId, idOk := partialProduct["number"]
		if !(nameOk && idOk) {
			continue
		}

		// Parse name/id to string
		name, nameOk := rawName.(string)
		id, idOk := rawId.(string)
		if !(nameOk && idOk) {
			continue
		}

		// Parse the amount (optional)
		amount := 0
		if rawAmount, ok := partialProduct["amount"]; ok {
			// Load the amount if it is an int and is greater than 0
			if amountInt, ok := rawAmount.(int); ok && amountInt > 0 {
				amount = amountInt
			}
		}

		product := products.PartialProduct{
			Name:   name,
			ID:     id,
			Amount: amount,
		}

		// Load the product into the cache
		productsMap[location][product.ID] = product
		totalLoaded++
	}

	log.Printf("fetched Transact API partial product cache (%d total); fetching again in %s\n",
		totalLoaded, delayUntilNext)

	// Load the products into the cache
	p.Cache.Load(productsMap)
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
