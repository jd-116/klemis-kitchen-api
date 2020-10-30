package cas

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"text/template"
	"time"

	"gopkg.in/cas.v2"

	"github.com/jd-116/klemis-kitchen-api/util"
	"github.com/segmentio/ksuid"
)

// Provider bundles together various structs
// involved in consuming CAS requests/implementing the flow
type Provider struct {
	url                  *url.URL
	samlValidateTemplate *template.Template
	httpClient           *http.Client
}

type samlValidateArguments struct {
	RequestID    string
	IssueInstant string
	Ticket       string
}

type soapEnvelope struct {
	XMLName xml.Name    `xml:"http://schemas.xmlsoap.org/soap/envelope/ Enelope"`
	Body    soapBody    `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Header  *soapHeader `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"`
}

type soapHeader struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"`
}

type soapBody struct {
	XMLName xml.Name   `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Status  samlStatus `xml:"urn:oasis:names:tc:SAML:1.0:protocol Status"`
}

type samlStatus struct {
	XMLName    xml.Name       `xml:"urn:oasis:names:tc:SAML:1.0:protocol Status"`
	StatusCode samlStatusCode `xml:"urn:oasis:names:tc:SAML:1.0:protocol StatusCode"`
}

type samlStatusCode struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:1.0:protocol StatusCode"`
	Value   string   `xml:"Value"`
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

	samlValidateTemplate, err := template.New("samlValidateTemplate").Parse(`
		<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">
			<SOAP-ENV:Header/>
			<SOAP-ENV:Body>
				<saml1p:Request xmlns:saml1p="urn:oasis:names:tc:SAML:1.0:protocol" MajorVersion="1" MinorVersion="1" RequestID="{{.RequestID}}" IssueInstant="{{.IssueInstant}}">
					<saml1p:AssertionArtifact>{{.Ticket}}</saml1p:AssertionArtifact>
				</saml1p:Request>
			</SOAP-ENV:Body>
		</SOAP-ENV:Envelope>
		`)
	if err != nil {
		return nil, err
	}

	return &Provider{
		url:                  casUrl,
		samlValidateTemplate: samlValidateTemplate,
		httpClient:           &http.Client{},
	}, nil
}

// Redirect attempts to send a redirect response that redirects to the SSO page,
// or returns an error if it failed
func (c *Provider) Redirect(w http.ResponseWriter, r *http.Request) error {
	// Get the original query URL without any queries
	requestURL, err := requestURL(r)
	if err != nil {
		return err
	}
	requestURL.RawQuery = ""

	// Construct the redirect URL to the GT SSO service
	redirectURL, err := c.url.Parse(path.Join(c.url.Path, "login"))
	if err != nil {
		return err
	}
	q := redirectURL.Query()
	q.Add("service", requestURL.String())
	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
	return nil
}

// ServiceValidate constructs and sends the service validate request to the CAS Server,
// parsing the body if successful
func (c *Provider) ServiceValidate(r *http.Request, ticket string) (*cas.AuthenticationResponse, error) {
	// Get the original query URL without any queries
	requestURL, err := requestURL(r)
	if err != nil {
		return nil, err
	}
	requestURL.RawQuery = ""

	// Construct the SAML Validate URL
	samlValidateURL, err := c.url.Parse(path.Join(c.url.Path, "samlValidate"))
	if err != nil {
		return nil, err
	}
	q := samlValidateURL.Query()
	q.Add("TARGET", requestURL.String())
	samlValidateURL.RawQuery = q.Encode()

	// Generate a random ID for the SAML request ID
	requestID, err := ksuid.NewRandom()
	if err != nil {
		return nil, err
	}

	// Construct the request body
	bodyArguments := samlValidateArguments{
		RequestID:    requestID.String(),
		IssueInstant: time.Now().UTC().Format(time.RFC3339),
		Ticket:       ticket,
	}
	buf := &bytes.Buffer{}
	err = c.samlValidateTemplate.Execute(buf, bodyArguments)
	if err != nil {
		return nil, err
	}

	// Create the request with all options
	method := http.MethodPost
	req, err := http.NewRequest(method, samlValidateURL.String(), buf)
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

	// Try to parse the response body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	log.Println()
	log.Println(string(body))
	log.Println()

	// Make sure the request succeeded
	if res.StatusCode != http.StatusOK {
		return nil, NewCASValidationFailedError()
	}

	soapResponse := soapEnvelope{}
	err = xml.Unmarshal(body, &soapResponse)
	if err != nil {
		return nil, err
	}
	log.Printf("%+v\n", soapResponse)
	log.Println()
	log.Printf("%#v\n", soapResponse)
	log.Println()

	// TODO remove
	authResponse, err := cas.ParseServiceResponse(body)
	if err != nil {
		return nil, err
	}

	return authResponse, nil
}

// requestURL determines an absolute URL from the http.Request.
// Taken from gopkg.in/cas.v2
func requestURL(r *http.Request) (*url.URL, error) {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		return nil, err
	}

	u.Host = r.Host
	if host := r.Header.Get("X-Forwarded-Host"); host != "" {
		u.Host = host
	}

	u.Scheme = "http"
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		u.Scheme = scheme
	} else if r.TLS != nil {
		u.Scheme = "https"
	}

	return u, nil
}

var (
	urlCleanParameters = []string{"gateway", "renew", "service", "ticket"}
)
