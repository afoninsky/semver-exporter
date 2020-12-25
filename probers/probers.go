package probers

import (
	"fmt"

	"github.com/afoninsky/version-exporter/probers/helm"
)

// Prober describes common interface for version probers
type Prober interface {
	Probe(string) (string, error)
}

func New(proberType string, rawCfg string) (Prober, error) {
	switch proberType {
	case "helm":
		return helm.New(rawCfg)
	default:
		return nil, fmt.Errorf(`prober type %s is not supported`, proberType)
	}
}
