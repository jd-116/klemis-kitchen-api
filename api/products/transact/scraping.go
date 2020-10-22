package transact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

// Scraper struct to hold authentication state
type Scraper struct {
	Ready         bool
	ClientVersion string
	client        *http.Client
	baseURL       string
	tenant        string
	username      string
	password      string
	authToken     string
	sync.Mutex
}

// NewScraper creates a new instance of the scraper
func NewScraper(baseURL string, tenant string, username string, password string) (*Scraper, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &Scraper{
		Ready:    false,
		baseURL:  baseURL,
		tenant:   tenant,
		username: username,
		password: password,
		client: &http.Client{
			Jar: jar,
		},
	}, nil
}

// ReloadSession reloads the session on the scraper
func (s *Scraper) ReloadSession() error {
	s.Lock()
	defer s.Unlock()

	// Clear the state
	s.client.Jar = nil
	s.authToken = ""
	s.Ready = false
	s.ClientVersion = ""

	// Create a new cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	s.client.Jar = jar

	// Get the new client version
	clientVersion, err := s.getClientVersion()
	if err != nil {
		return err
	}
	s.ClientVersion = clientVersion

	// Obtain a new session cookie
	err = s.getSessionCookie()
	if err != nil {
		return err
	}

	// Obtain a new authentication token
	authToken, err := s.getAuthenticationToken()
	if err != nil {
		return err
	}

	// Load the auth token into the scraper
	s.authToken = authToken
	s.Ready = true
	return nil
}

// Expected JSON from items request
type itemsWithInventoryResponse struct {
	Result itemsWithInventoryResult `json:"GetItemsWithInventoryMainResult"`
}

type itemsWithInventoryResult struct {
	Items []map[string]interface{} `json:"RootResults"`
}

// GetItemsForClass attempts to get all items with inventory for the given item class name
func (s *Scraper) GetItemsForClass(className string) ([]map[string]interface{}, error) {
	// Get a lock on the session lock
	s.Lock()
	defer s.Unlock()

	if !s.Ready {
		// Cannot process request
		return nil, errors.New("session has not been initialized")
	}

	url := s.baseURL + "/QPWebOffice-Web-QuadPointDomain.svc/JSON/GetItemsWithInventoryMain"
	method := "GET"

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.authToken))

	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Parse JSON into expected shape
	result := itemsWithInventoryResponse{}
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	// Filter items by class name
	items := make([]map[string]interface{}, 0)
	originalCount := len(result.Result.Items)
	for _, resultItem := range result.Result.Items {
		if itemClassName, ok := resultItem["class_name"]; ok && itemClassName == className {
			items = append(items, resultItem)
		}
	}
	log.Printf("scraped %d -> %d raw items from the Transact API\n", originalCount, len(items))

	return items, nil
}

// Attempts to obtain a new session cookie from the Transact API,
// and if successful, stores it in the cookie jar contained within the Scraper
func (s *Scraper) getSessionCookie() error {
	url := s.baseURL + "/QPWebOffice-Web-AuthenticationService.svc/JSON/LoggedIn"
	method := "POST"
	payload := strings.NewReader("{}")

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return err
	}

	req.Header.Add("Referer", s.baseURL+"/?tenant="+s.tenant)
	req.Header.Add("Content-Type", "application/json")

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Look for the set-cookie header
	if _, ok := res.Header["Set-Cookie"]; ok {
		// Assume good
		return nil
	}

	return errors.New("no cookie header found when attempting to get a session cookie")
}

// Attempts to get the client version to use
func (s *Scraper) getClientVersion() (string, error) {
	url := s.baseURL + "/?tenant=" + s.tenant
	method := "GET"

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return "", err
	}

	res, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	doc, err := htmlquery.Parse(res.Body)
	title, err := htmlquery.Query(doc, "//title")
	if err != nil {
		return "", err
	}

	// Extract the client version from the node
	text := &bytes.Buffer{}
	collectText(title, text)
	titleStr := text.String()
	if strings.HasPrefix(titleStr, "QuadPoint Cloud ") {
		// Extract the version from the page title string
		version := strings.TrimPrefix(titleStr, "QuadPoint Cloud ")
		return version, nil
	}

	return "", fmt.Errorf("malformed page title '%s'; expecting 'QuadPoint Cloud X.X.X.X'", titleStr)
}

// Collects all the inner text for a given HTML node
func collectText(n *html.Node, buf *bytes.Buffer) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, buf)
	}
}

// Attempts to get a new authentication token by sending a request to the login route
// using the current session cookie
func (s *Scraper) getAuthenticationToken() (string, error) {
	url := s.baseURL + "/QPWebOffice-Web-AuthenticationService.svc/JSON/Authenticate"
	method := "POST"
	payloadJSON := map[string]interface{}{
		"isPersistent":   true,
		"customData":     "",
		"dotNetLogicVer": 1,
		"clientVersion":  s.ClientVersion,
		"userName":       s.username,
		"password":       s.password,
		"reset":          "***",
		"id":             "***",
	}
	payload, err := json.Marshal(payloadJSON)
	if err != nil {
		return "", nil
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}

	req.Header.Add("Referer", s.baseURL+"/?tenant="+s.tenant)
	req.Header.Add("Content-Type", "application/json")

	res, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// Look for the authorization header
	if authorization, ok := res.Header["Authorization"]; ok && len(authorization) >= 1 {
		authValue := authorization[0]
		if strings.HasPrefix(authValue, "Bearer ") {
			// Extract the auth token from the header value
			version := strings.TrimPrefix(authValue, "Bearer ")
			return version, nil
		}

		return "", fmt.Errorf("malformed authorization token '%s'; expecting 'Bearer X'", authValue)
	}

	return "", errors.New("no authorization header found when attempting to get a session cookie")
}
