package probers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	Cfg        string
	Version    string
	Expect     string
	InitError  string
	ProbeError string
}

var testSuits map[string][]testCase

func init() {
	testSuits = map[string][]testCase{}

	// test helm chart
	helmDefaultCfg := "chart: ingress-nginx\nrepository: https://kubernetes.github.io/ingress-nginx"
	helmLatestVersion := "3.17.0"
	testSuits["helm"] = []testCase{
		testCase{
			Cfg:       "chart: nosuch\nrepository: invalid-url",
			InitError: "validate config",
		},
		testCase{
			Cfg:        "chart: nosuch\nrepository: https://kubernetes.github.io/ingress-nginx",
			ProbeError: "was not found",
		},
		testCase{
			Cfg:    "chart: ingress-nginx\nrepository: https://kubernetes.github.io/ingress-nginx\nconstraint: <3.0.0",
			Expect: "2.16.0",
		},
		testCase{
			Cfg:    helmDefaultCfg,
			Expect: helmLatestVersion,
		},
		testCase{
			Cfg:     helmDefaultCfg,
			Version: helmLatestVersion,
			Expect:  helmLatestVersion,
		},
		testCase{
			Cfg:     helmDefaultCfg,
			Version: "1000.1000.1000",
			Expect:  "1000.1000.1000",
		},
		testCase{
			Cfg:        helmDefaultCfg,
			Version:    "non-semantic-version",
			ProbeError: "parse version",
		},
	}
}

func TestSuits(t *testing.T) {
	assert := assert.New(t)
	for pType, testCases := range testSuits {
		for _, c := range testCases {

			p, err := New(pType, c.Cfg)

			if c.InitError != "" {
				assert.Errorf(err, fmt.Sprintf("%s: should return error if invalid config passed", pType))
				assert.Containsf(err.Error(), c.InitError, fmt.Sprintf(`%s: error should contain string "%s"`, pType, c.InitError))
				continue
			}

			v, err := p.Probe(c.Version)

			if c.ProbeError != "" {
				assert.Errorf(err, fmt.Sprintf("%s: should return error during probe", pType))
				assert.Containsf(err.Error(), c.ProbeError, fmt.Sprintf(`%s: error should contain string "%s"`, pType, c.ProbeError))
				continue
			}

			assert.Equalf(c.Expect, v, fmt.Sprintf(`%s: probe should return "%s"`, pType, c.Expect))
		}
	}
}
