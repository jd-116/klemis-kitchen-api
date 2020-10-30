package cas

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"gopkg.in/cas.v2"

	"github.com/jd-116/klemis-kitchen-api/util"
)

// Provider bundles together various structs
// involved in consuming CAS requests/implementing the flow
type Provider struct {
	client     *cas.Client
	httpClient *http.Client
}

// NewProvider creates sa new instance of the Provider
// and loads in options from the environment
func NewProvider() (*Provider, error) {
	casUrlStr, err := util.GetEnv("CAS base URL", "CAS_SERVER_URL")
	if err != nil {
		return nil, err
	}

	casUrl, err := url.Parse(casUrlStr)
	if err != nil {
		return nil, err
	}

	client := cas.NewClient(&cas.Options{
		URL: casUrl,
	})

	return &Provider{
		client:     client,
		httpClient: &http.Client{},
	}, nil
}

// Redirect attempts to send a redirect response that redirects to the SSO page,
// or returns an error if it failed
func (c *Provider) Redirect(w http.ResponseWriter, r *http.Request) error {
	// Get the redirect URL to the GT SSO service
	redirectUrl, err := c.client.LoginUrlForRequest(r)
	log.Println(redirectUrl)
	if err != nil {
		return err
	}

	http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
	return nil
}

// ServiceValidate constructs and sends the service validate request to the CAS Server,
// parsing the body if successful
func (c *Provider) ServiceValidate(r *http.Request, ticket string) (*cas.AuthenticationResponse, error) {
	validateUrl, err := c.client.ServiceValidateUrlForRequest(ticket, r)
	log.Println(validateUrl)
	if err != nil {
		return nil, err
	}

	// Create the request with all options
	method := http.MethodGet
	req, err := http.NewRequest(method, validateUrl, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(r.Context())

	// Make the request to the CAS server
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Make sure the request succeeded
	if res.StatusCode != http.StatusOK {
		return nil, NewCASValidationFailedError()
	}

	// Try to parse the response body
	body, err := ioutil.ReadAll(res.Body)
	authResponse, err := cas.ParseServiceResponse(body)
	if err != nil {
		return nil, err
	}

	return authResponse, nil
}
