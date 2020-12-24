package probers

import "github.com/Masterminds/semver"

// Prober describes common interface for semantic probers
type Prober interface {
	Probe(*semver.Version) (*semver.Version, error)
}
