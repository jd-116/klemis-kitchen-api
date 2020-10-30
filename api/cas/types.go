package cas

import "encoding/xml"

// Identity contains the fields that are released for GT SSO
type Identity struct {
	Username  string
	FirstName string
	LastName  string
}

type samlValidateArguments struct {
	RequestID    string
	IssueInstant string
	Ticket       string
}

type soapEnvelope struct {
	XMLName xml.Name   `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    soapBody   `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Header  soapHeader `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"`
}

type soapBody struct {
	InnerXML []byte `xml:",innerxml"`
}

type soapHeader struct {
	InnerXML []byte `xml:",innerxml"`
}

type samlRequest struct {
	XMLName           xml.Name `xml:"urn:oasis:names:tc:SAML:1.0:protocol Request"`
	MajorVersion      string   `xml:"MajorVersion,attr"`
	MinorVersion      string   `xml:"MinorVersion,attr"`
	RequestID         string   `xml:"RequestID,attr"`
	IssueInstant      string   `xml:"IssueInstant,attr"`
	AssertionArtifact string   `xml:"urn:oasis:names:tc:SAML:1.0:protocol AssertionArtifact"`
}

type samlResponse struct {
	ResponseID   string        `xml:"ResponseID,attr"`
	IssueInstant string        `xml:"IssueInstant,attr"`
	Recipient    string        `xml:"Recipient,attr"`
	MajorVersion string        `xml:"MajorVersion,attr"`
	MinorVersion string        `xml:"MinorVersion,attr"`
	Status       samlStatus    `xml:"urn:oasis:names:tc:SAML:1.0:protocol Status"`
	Assertion    samlAssertion `xml:"urn:oasis:names:tc:SAML:1.0:assertion Assertion"`
}

type samlStatus struct {
	StatusCode samlStatusCode `xml:"urn:oasis:names:tc:SAML:1.0:protocol StatusCode"`
}

type samlStatusCode struct {
	Value string `xml:"Value,attr"`
}

type samlAssertion struct {
	AssertionID             string                      `xml:"AssertionID,attr"`
	IssueInstant            string                      `xml:"IssueInstant,attr"`
	Issuer                  string                      `xml:"Issuer,attr"`
	MajorVersion            string                      `xml:"MajorVersion,attr"`
	MinorVersion            string                      `xml:"MinorVersion,attr"`
	Conditions              samlConditions              `xml:"urn:oasis:names:tc:SAML:1.0:assertion Conditions"`
	AttributeStatement      samlAttributeStatement      `xml:"urn:oasis:names:tc:SAML:1.0:assertion AttributeStatement"`
	AuthenticationStatement samlAuthenticationStatement `xml:"urn:oasis:names:tc:SAML:1.0:assertion AuthenticationStatement"`
}

type samlConditions struct {
	NotBefore    string `xml:"NotBefore,attr"`
	NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
	InnerXML     []byte `xml:",innerxml"`
}

type samlAuthenticationStatement struct {
	AuthenticationInstant string      `xml:"AuthenticationInstant,attr"`
	AuthenticationMethod  string      `xml:"AuthenticationMethod,attr"`
	Subject               samlSubject `xml:"urn:oasis:names:tc:SAML:1.0:assertion Subject"`
}

type samlAttributeStatement struct {
	Subject    samlSubject     `xml:"urn:oasis:names:tc:SAML:1.0:assertion Subject"`
	Attributes []samlAttribute `xml:"urn:oasis:names:tc:SAML:1.0:assertion Attribute"`
}

type samlSubject struct {
	NameIdentifier      string                  `xml:"urn:oasis:names:tc:SAML:1.0:assertion NameIdentifier"`
	SubjectConfirmation samlSubjectConfirmation `xml:"urn:oasis:names:tc:SAML:1.0:assertion SubjectConfirmation"`
}

type samlSubjectConfirmation struct {
	ConfirmationMethod string `xml:"urn:oasis:names:tc:SAML:1.0:assertion ConfirmationMethod"`
}

type samlAttribute struct {
	AttributeName      string `xml:"AttributeName,attr"`
	AttributeNamespace string `xml:"AttributeNamespace,attr"`
	AttributeValue     string `xml:"urn:oasis:names:tc:SAML:1.0:assertion AttributeValue"`
}
