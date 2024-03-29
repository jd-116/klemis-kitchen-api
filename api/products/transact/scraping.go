package transact

import (
	"bytes"
	"encoding/csv"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/hako/durafmt"
	"github.com/rs/zerolog"
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
	logger zerolog.Logger
}

// NewScraper creates a new instance of the scraper
func NewScraper(
	baseURL string,
	tenant string,
	username string,
	password string,
	logger zerolog.Logger,
) (*Scraper, error) {
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
		logger: logger,
	}, nil
}

// ReloadSession reloads the session on the scraper
func (s *Scraper) ReloadSession() error {
	s.Lock()
	defer s.Unlock()

	s.logger.Info().Msg("reloading Transact session")

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

	s.logger.Info().Msg("successfully reloaded Transact session")

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
	s.logFavoriteReportNames(allFavoriteReports)

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
	if err != nil {
		return nil, err
	}

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
func (s *Scraper) logFavoriteReportNames(allFavoriteReports []map[string]interface{}) {
	// Log all favorite reports' names
	allReports := []string{}
	for _, favoriteReport := range allFavoriteReports {
		if name, ok := favoriteReport["name"]; ok {
			if asStr, ok := name.(string); ok {
				allReports = append(allReports, asStr)
			}
		}
	}
	s.logger.
		Info().
		Strs("all_reports", allReports).
		Msg("favorite reports scraped from Transact")
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

	err := s.submitReport(report, reportType, reportID, reportName)
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
	urlWithoutAuth := s.baseURL + "/BinaryDataService.svc/HistoryReport/" + escapedFilename + "/CSV"
	url := urlWithoutAuth + "?jwthidden=" + s.authToken
	method := "GET"

	s.logger.
		Info().
		Str("url_without_auth", urlWithoutAuth).
		Str("method", method).
		Str("report_name", reportName).
		Msg("downloading report file from Transact")

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

	contents := buf.String()
	s.logger.
		Info().
		Int("file_length", len(contents)).
		Str("report_name", reportName).
		Msg("successfully downloaded report file from Transact")

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

	s.logger.
		Info().
		Str("url", url).
		Str("method", method).
		Bytes("body", isReportReadyRequestBody).
		Int("report_id", reportID).
		Str("report_name", reportName).
		Msg("polling the Transact API to determine if report is ready")

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
		s.logger.
			Info().
			Str("report_url", *result.Result.File).
			Int("report_id", reportID).
			Str("report_name", reportName).
			Msg("report was ready")

		return result.Result.File, nil
	}

	s.logger.Info().Msg("report was not ready")

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

	s.logger.
		Info().
		Str("url", url).
		Str("method", method).
		Msg("getting all favorite reports")

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

	s.logger.Info().Msg("successfully got all favorite reports")

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

func deepCopyMap(m map[string]interface{}) (map[string]interface{}, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err := enc.Encode(m)
	if err != nil {
		return nil, err
	}
	var copy map[string]interface{}
	err = dec.Decode(&copy)
	if err != nil {
		return nil, err
	}
	return copy, nil
}

// submitReport sends the given report definition to the API
// and requests it be generated.
// Note: s.Mutex should be locked before calling this function
func (s *Scraper) submitReport(reportDefinition map[string]interface{},
	reportType string, reportID int, reportName string) error {

	// Create the submit report body JSON
	reportDefinitionCopy, err := deepCopyMap(reportDefinition)
	if err != nil {
		return fmt.Errorf("failed to deep copy report definition: %w", err)
	}
	reportDefinitionCopy["__type"] = reportType
	reportDefinitionCopy["last_filename"] = ""
	reportDefinitionCopy["last_run"] = nil
	reportDefinitionCopy["subject"] = ""
	reportDefinitionCopy["enabled"] = true
	// I'm not sure if this is neccessary,
	// but it's done in the actual requests in QuadPoint POS.
	reportDefinitionCopy["all_fields"] = fmt.Sprintf("%s %s %s", reportName, reportDefinitionCopy["report_name"], s.username)
	delete(reportDefinitionCopy, "queue_time")
	body := submitReportBody{
		// These values come from observing the requests in devtools
		ChangeSet: []submitReportChange{{
			ID:        0,
			Operation: 3,
			Entity:    reportDefinitionCopy,
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

	s.logger.
		Info().
		Str("url", url).
		Str("method", method).
		Str("report_name", reportName).
		Int("report_id", reportID).
		Str("report_type", reportType).
		Msg("requesting report to be generated")

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

	url = fmt.Sprintf("%s/api/v2/tenants/%s/reportjobs/%d/finalize", s.baseURL, s.tenant, reportID)
	method = "POST"

	s.logger.
		Info().
		Str("url", url).
		Str("method", method).
		Str("report_name", reportName).
		Int("report_id", reportID).
		Str("report_type", reportType).
		Msg("finalizing report request")

	req, err = http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.authToken))

	res, err = s.client.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("report request finalization for report '%s' failed", reportName)
	}

	s.logger.
		Info().
		Msg("successfully requested report to be generated")

	// Report submission was successful
	return nil
}

// Attempts to obtain a new session cookie from the Transact API,
// and if successful, stores it in the cookie jar contained within the Scraper
func (s *Scraper) getSessionCookie() error {
	url := s.baseURL + "/QPWebOffice-Web-AuthenticationService.svc/JSON/LoggedIn"
	method := "POST"
	payload := strings.NewReader("{}")

	s.logger.
		Info().
		Str("url", url).
		Str("method", method).
		Str("body", "{}").
		Msg("getting new Transact session cookie")

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
	if _, ok := res.Header["Set-Cookie"]; !ok {
		return errors.New("no cookie header found when attempting to get a session cookie")
	}

	s.logger.
		Info().
		Msg("successfully obtained new Transact session cookie")

	// Assume good
	return nil
}

// Attempts to get the client version to use
func (s *Scraper) getClientVersion() (string, error) {

	url := s.baseURL + "/?tenant=" + s.tenant
	method := "GET"

	s.logger.
		Info().
		Str("url", url).
		Str("method", method).
		Msg("getting current Transact client version")

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
	if err != nil {
		return "", fmt.Errorf("error parsing HTML response: %w", err)
	}
	title, err := htmlquery.Query(doc, "//title")
	if err != nil {
		return "", fmt.Errorf("error querying HTML response for title tag: %w", err)
	}

	// Extract the version from the page title string
	text := &bytes.Buffer{}
	collectText(title, text)
	titleStr := text.String()
	expectedTitlePrefixes := []string{"QuadPoint Cloud ", "Transact Cloud POS "}
	var version string
	parsed := false
	for _, prefix := range expectedTitlePrefixes {
		if strings.HasPrefix(titleStr, prefix) {
			version = strings.TrimPrefix(titleStr, prefix)
			parsed = true
			break
		}
	}
	if !parsed {
		return "", fmt.Errorf("malformed page title '%s'; expecting one of prefixes [%s]", titleStr, strings.Join(expectedTitlePrefixes, "', '"))
	}

	s.logger.
		Info().
		Str("client_version", version).
		Str("title", titleStr).
		Msg("successfully obtained current Transact client version")

	return version, nil
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

	s.logger.
		Info().
		Str("url", url).
		Str("method", method).
		Msg("attempting to log in and acquire authentication token")

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

			s.logger.
				Info().
				Int("token_length", len(token)).
				Msg("successfully logged in to Transact and obtained authentication token")

			return token, nil
		}

		return "", fmt.Errorf("malformed authorization token '%s'; expecting 'Bearer X'", authValue)
	}

	return "", errors.New("no authorization header found when attempting to get a session cookie")
}
