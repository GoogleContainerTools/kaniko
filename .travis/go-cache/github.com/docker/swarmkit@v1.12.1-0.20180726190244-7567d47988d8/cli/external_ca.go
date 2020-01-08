package cli

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/swarmkit/api"
)

// ExternalCAOpt is a Value type for parsing external CA specifications.
type ExternalCAOpt struct {
	values []*api.ExternalCA
}

// Set parses an external CA option.
func (m *ExternalCAOpt) Set(value string) error {
	parsed, err := parseExternalCA(value)
	if err != nil {
		return err
	}

	m.values = append(m.values, parsed)
	return nil
}

// Type returns the type of this option.
func (m *ExternalCAOpt) Type() string {
	return "external-ca"
}

// String returns a string repr of this option.
func (m *ExternalCAOpt) String() string {
	externalCAs := []string{}
	for _, externalCA := range m.values {
		repr := fmt.Sprintf("%s: %s", externalCA.Protocol, externalCA.URL)
		externalCAs = append(externalCAs, repr)
	}
	return strings.Join(externalCAs, ", ")
}

// Value returns the external CAs
func (m *ExternalCAOpt) Value() []*api.ExternalCA {
	return m.values
}

// parseExternalCA parses an external CA specification from the command line,
// such as protocol=cfssl,url=https://example.com.
func parseExternalCA(caSpec string) (*api.ExternalCA, error) {
	csvReader := csv.NewReader(strings.NewReader(caSpec))
	fields, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	externalCA := api.ExternalCA{
		Options: make(map[string]string),
	}

	var (
		hasProtocol bool
		hasURL      bool
	)

	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)

		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid field '%s' must be a key=value pair", field)
		}

		key, value := parts[0], parts[1]

		switch strings.ToLower(key) {
		case "protocol":
			hasProtocol = true
			if strings.ToLower(value) == "cfssl" {
				externalCA.Protocol = api.ExternalCA_CAProtocolCFSSL
			} else {
				return nil, fmt.Errorf("unrecognized external CA protocol %s", value)
			}
		case "url":
			hasURL = true
			externalCA.URL = value
		default:
			externalCA.Options[key] = value
		}
	}

	if !hasProtocol {
		return nil, errors.New("the external-ca option needs a protocol= parameter")
	}
	if !hasURL {
		return nil, errors.New("the external-ca option needs a url= parameter")
	}

	return &externalCA, nil
}
