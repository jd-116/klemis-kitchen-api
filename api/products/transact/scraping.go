package transact

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/hako/durafmt"
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

// GetInventoryCSV is a goroutine that goes through the process
// of getting the inventory CSV via a report.
// Returns each CSV row as a string slice
func (s *Scraper) GetInventoryCSV(csvReportName string, pollPeriod time.Duration,
	pollTimeout time.Duration, reportType string) ([][]string, error) {

	// Get all favorite reports and then match the desired one
	allFavoriteReports, err := s.getFavoriteReports()
	if err != nil {
		return nil, err
	}
	logFavoriteReportNames(allFavoriteReports)

	// Try to find a matching report
	var matchedReport map[string]interface{} = nil
	for _, favoriteReport := range allFavoriteReports {
		if name, ok := favoriteReport["name"]; ok && name == csvReportName {
			// Found
			matchedReport = favoriteReport
			break
		}
	}
	if matchedReport == nil {
		return nil, fmt.Errorf("no matching favorite report found for name '%s'", csvReportName)
	}

	// Now, submit the request to generate the report
	reportFileName, err := s.submitAndWait(matchedReport, reportType, pollPeriod, pollTimeout)

	// Download the report
	reportContents, err := s.downloadReport(reportFileName)
	if err != nil {
		return nil, err
	}

	// Parse the CSV records
	csvReader := csv.NewReader(strings.NewReader(reportContents))
	csvReader.LazyQuotes = true
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	return records, nil
}

// Short utility function to log all favorite reports' names that were scraped
func logFavoriteReportNames(allFavoriteReports []map[string]interface{}) {
	// Log all favorite reports' names
	allReports := []string{}
	for _, favoriteReport := range allFavoriteReports {
		if name, ok := favoriteReport["name"]; ok {
			if asStr, ok := name.(string); ok {
				allReports = append(allReports, asStr)
			}
		}
	}
	allReportsJson, err := json.Marshal(allReports)
	if err != nil {
		log.Printf("could not dump favorite reports scraped from Transact: %v\n", err)
		log.Printf("non-JSON dump: %#v", allReports)
	} else {
		log.Printf("favorite reports scraped from Transact: %s\n", allReportsJson)
	}
}

// Submits a report generation request and polls until it is done,
// returning the filename to the resultant report when done
func (s *Scraper) submitAndWait(report map[string]interface{}, reportType string,
	pollPeriod time.Duration, pollTimeout time.Duration) (string, error) {

	s.Lock()
	defer s.Unlock()

	reportNameRaw, ok := report["name"]
	if !ok {
		return "", errors.New("no 'name' field found on report")
	}
	reportName, ok := reportNameRaw.(string)
	if !ok {
		return "", errors.New("invalid 'name' field found on report")
	}

	reportIDRaw, ok := report["id"]
	if !ok {
		return "", fmt.Errorf("no 'id' field found on report with name '%s'", reportName)
	}
	reportIDFloat, ok := reportIDRaw.(float64)
	if !ok {
		return "", fmt.Errorf("invalid 'id' field found on report with name '%s'", reportName)
	}
	reportID := int(reportIDFloat)

	err := s.submitReport(report, reportType)
	if err != nil {
		return "", err
	}

	// Keep polling the API until the report is ready,
	// and obtain the path from the response that indicates success
	timeout := time.After(pollTimeout)
	timeoutHumanDuration := durafmt.Parse(pollTimeout).LimitFirstN(2).String()
	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("report polling for report with name '%s' timed out after %s",
				reportName, timeoutHumanDuration)
		case <-time.After(pollPeriod):
			reportFile, err := s.isReportReady(reportID, reportName)
			if err != nil {
				return "", err
			}

			if reportFile == nil {
				// Poll again after delay
				continue
			}

			return *reportFile, nil
		}
	}
}

// downloadReport downloads a report file from the Transact API
func (s *Scraper) downloadReport(reportName string) (string, error) {
	s.Lock()
	defer s.Unlock()

	// Construct the report URL by escaping each path segment,
	// but not the slashes between them
	filename := "QuadPoint POS/" + reportName
	segments := strings.Split(filename, "/")
	escapedSegments := []string{}
	for _, segment := range segments {
		escapedSegments = append(escapedSegments, url.PathEscape(segment))
	}
	escapedFilename := strings.Join(escapedSegments, "/")
	url := s.baseURL + "/BinaryDataService.svc/HistoryReport/" + escapedFilename + "/CSV"
	url = url + "?jwthidden=" + s.authToken
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

	// Read the body into a string
	buf := new(strings.Builder)
	_, err = io.Copy(buf, res.Body)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

type isReportReadyBody struct {
	ScheduleID int `json:"scheduleId"`
}

type isReportReadyResponse struct {
	Result isReportReadyResult `json:"IsReportReadyResult"`
}

type isReportReadyResult struct {
	File    *string `json:"reportFile"`
	Ready   bool    `json:"reportReady"`
	Success bool    `json:"success"`
}

// isReportReady polls the Transact API to determine if the report is ready for downloading.
// If it is, it returns the filename from the API, which can be used with downloadReport
func (s *Scraper) isReportReady(reportID int, reportName string) (*string, error) {
	// Create the isReportReady body JSON
	isReportReadyRequestBody, err := json.Marshal(isReportReadyBody{ScheduleID: reportID})
	if err != nil {
		return nil, err
	}

	url := s.baseURL + "/QPWebOffice-Web-BusinessService.svc/JSON/IsReportReady"
	method := "POST"

	req, err := http.NewRequest(method, url, bytes.NewReader(isReportReadyRequestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.authToken))
	req.Header.Add("Content-Type", "application/json")

	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("report polling for report '%s' failed", reportName)
	}

	// Decode response body
	result := isReportReadyResponse{}
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	// See if the report is ready or if it has failed
	if !result.Result.Success {
		return nil, fmt.Errorf("report creation for '%s' failed", reportName)
	}
	if result.Result.Ready && result.Result.File != nil {
		return result.Result.File, nil
	}

	// Report isn't ready yet, but no error
	return nil, nil
}

type favoriteReportsResponse struct {
	Result favoriteReportsResult `json:"GetFavoritesResult"`
}

type favoriteReportsResult struct {
	Items []map[string]interface{} `json:"RootResults"`
}

// getFavoriteReports gets all favorite reports
func (s *Scraper) getFavoriteReports() ([]map[string]interface{}, error) {
	s.Lock()
	defer s.Unlock()

	url := s.baseURL + "/QPWebOffice-Web-QuadPointDomain.svc/JSON/GetFavorites"
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

	result := favoriteReportsResponse{}
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Result.Items, nil
}

type submitReportBody struct {
	ChangeSet []submitReportChange `json:"changeSet"`
}

type submitReportChange struct {
	Entity         map[string]interface{}     `json:"Entity"`
	OriginalEntity submitReportOriginalEntity `json:"OriginalEntity"`
	ID             int                        `json:"Id"`
	Operation      int                        `json:"Operation"`
}

type submitReportOriginalEntity struct {
	Type string `json:"__type"`
}

// submitReport sends the given report definition to the API
// and requests it be generated.
// Note: s.Mutex should be locked before calling this function
func (s *Scraper) submitReport(reportDefinition map[string]interface{},
	reportType string) error {

	reportName := "unknown"
	if name, ok := reportDefinition["name"]; ok {
		if nameStr, ok := name.(string); ok {
			reportName = nameStr
		}
	}

	// Create the submit report body JSON
	reportDefinition["__type"] = reportType
	body := submitReportBody{
		// These values come from observing the requests in devtools
		ChangeSet: []submitReportChange{{
			ID:        0,
			Operation: 3,
			Entity:    reportDefinition,
			OriginalEntity: submitReportOriginalEntity{
				Type: reportType,
			},
		}},
	}
	submitReportRequestBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := s.baseURL + "/QPWebOffice-Web-QuadPointDomain.svc/JSON/SubmitChanges"
	method := "POST"

	req, err := http.NewRequest(method, url, bytes.NewReader(submitReportRequestBody))
	if err != nil {
		return err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.authToken))
	req.Header.Add("Content-Type", "application/json")

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("report submission for report '%s' failed", reportName)
	}

	// Report submission was successful
	return nil
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
			token := strings.TrimPrefix(authValue, "Bearer ")

			// Check to see if the token is valid
			if token == "expired" {
				return "", errors.New("authorization token returned was expired; are account credentials correct?")
			}

			return token, nil
		}

		return "", fmt.Errorf("malformed authorization token '%s'; expecting 'Bearer X'", authValue)
	}

	return "", errors.New("no authorization header found when attempting to get a session cookie")
}
