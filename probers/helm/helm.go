package helm

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/Masterminds/semver"
	"github.com/go-playground/validator"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Config contains configuration options
type Config struct {
	Chart      string `yaml:"chart" validate:"required"`
	Repository string `yaml:"repository" validate:"required,uri"`
	Constraint string `yaml:"constraint"`
}

// Helm implements {probe} component
// checks the latest version of specified helm chart in helm repository
type Helm struct {
	url        *url.URL
	data       *repoStruct
	constraint *semver.Constraints
	client     *retryablehttp.Client
	chartName  string
}

// contains fetched from "index.yaml" repository information
type repoStruct struct {
	APIVersion string                   `yaml:"apiVersion"`
	Generated  string                   `yaml:"generated"`
	Entries    map[string][]chartStruct `yaml:"entries"`
}
type chartStruct struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	AppVersion  string `yaml:"appVersion"`
	Description string `yaml:"description"`
}

// New creates component instance
func New(rawCfg string) (Helm, error) {
	r := Helm{}
	cfg, err := r.config(rawCfg)
	if err != nil {
		return r, err
	}

	// parse repo name
	u, err := url.Parse(cfg.Repository)
	if err != nil {
		return r, errors.Wrap(err, "parse url")
	}
	u.Path = filepath.Join("/", u.Path, "index.yaml")
	r.url = u

	c := cfg.Constraint
	if c == "" {
		c = "*.*.*"
	}
	con, err := semver.NewConstraint(c)
	if err != nil {
		return r, errors.Wrap(err, "parse constraint")
	}
	r.constraint = con

	// init retryable client
	r.client = retryablehttp.NewClient()
	r.client.RetryMax = 3
	r.client.Logger = newLeveledLogger()

	r.chartName = cfg.Chart

	return r, nil
}

// Probe returns the latest version of the helm chart with the respect of constraints
func (r Helm) Probe(current string) (string, error) {

	if current == "" {
		current = "0.0.0"
	}
	cv, err := semver.NewVersion(current)
	if err != nil {
		return "", errors.Wrap(err, "parse version")
	}
	data, err := r.fetch()
	if err != nil {
		return "", errors.Wrap(err, "fetch repo")
	}
	releases, ok := data.Entries[r.chartName]
	if !ok {
		return "", errors.Errorf(`chart "%s" was not found in %s`, r.chartName, r.url.Host)
	}

	for _, release := range releases {
		v, err := semver.NewVersion(release.Version)
		if err != nil {
			// unable to parse release version as semantic version
			continue
		}
		if !r.constraint.Check(v) {
			// chart version does not meet the constraint
			continue
		}
		if v.GreaterThan(cv) {
			cv = v
		}
	}

	return cv.String(), nil
}

func (r Helm) config(rawCfg string) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal([]byte(rawCfg), &cfg); err != nil {
		return cfg, errors.Wrap(err, "parse config")
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return cfg, errors.Wrap(err, "validate config")
	}

	return cfg, nil
}

func (r Helm) fetch() (repoStruct, error) {
	repo := repoStruct{}
	res, err := r.client.Get(r.url.String())
	if err != nil {
		return repo, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return repo, fmt.Errorf("invalid status code: %d", res.StatusCode)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return repo, err
	}
	if err := yaml.Unmarshal(body, &repo); err != nil {
		return repo, err
	}
	if repo.APIVersion != "v1" {
		return repo, errors.New("supports only v1 chart manifests")
	}
	return repo, nil
}
