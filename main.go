package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Masterminds/semver"
	"github.com/VictoriaMetrics/metrics"
	"github.com/afoninsky/version-exporter/probers"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/memblob"
	_ "gocloud.dev/blob/s3blob"
	"gocloud.dev/gcerrors"
	"gopkg.in/yaml.v2"
)

var fListen = flag.String("listen", "127.0.0.1:8080", "The address to listen on for HTTP requests")
var fConfig = flag.String("config", "./probes.yaml", "Configuration file path")
var fEndpoint = flag.String("endpoint", "/metrics", "Metrics HTTP endpoint")
var fStorage = flag.String("storage", "mem://", "Storage path")

type probes map[string]struct {
	Interval time.Duration `yaml:"interval"`
	Type     string        `yaml:"type"`
	Config   string        `yaml:"config"`
}

func main() {
	flag.Parse()

	cfg, err := loadConfig(*fConfig)
	if err != nil {
		log.Fatal(err)
	}

	bucket, err := blob.OpenBucket(context.Background(), *fStorage)
	if err != nil {
		log.Fatal(err)
	}

	for name, probe := range cfg {
		log.Printf(`Testing %s with interval %s`, name, probe.Interval)
		p, err := probers.New(name, probe.Config)
		if err != nil {
			log.Fatal(err)
		}
		go startProber(name, p, probe.Interval, bucket)
	}

	http.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		metrics.WritePrometheus(w, false)
	})

	log.Printf("Expose metrics: %s%s", *fListen, *fEndpoint)
	log.Fatal(http.ListenAndServe(*fListen, nil))
}

func loadConfig(cfgPath string) (probes, error) {
	f, err := os.Open(cfgPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg probes
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func startProber(name string, p probers.Prober, d time.Duration, storage *blob.Bucket) {
	for {
		time.Sleep(d)
		v, err := getCurrentVersion(storage, name)
		if err != nil {
			log.Fatal(err)
		}
		newV, err := p.Probe(v)

		if err != nil {
			log.Printf("[ERROR] %s: %v", name, err)
			continue
		}
		if v == nil || v.String() == newV.String() {
			continue
		}
		if err := saveVersion(storage, name, newV); err != nil {
			log.Fatal(err)
		}
		log.Printf("%s, new version: %s -> %s\n", name, v, newV)
		labels := fmt.Sprintf(`semver_release{probe="%s",version="%s",version_major="%d",version_minor="%d",version_patch="%d"}`,
			name,
			newV,
			newV.Major(),
			newV.Minor(),
			newV.Patch(),
		)
		metrics.GetOrCreateCounter(labels).Set(1)
	}
}

// stores value in storage
func saveVersion(storage *blob.Bucket, name string, v string) error {
	w, err := storage.NewWriter(context.Background(), name, nil)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, v)
	if err != nil {
		return err
	}
	return w.Close()
}

// reads value from storage
func getCurrentVersion(storage *blob.Bucket, name string) (string, error) {
	r, err := storage.NewReader(context.Background(), name, nil)
	if err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			v, _ := semver.NewVersion("0.0.0")
			return v.String(), nil
		}
		return "", err
	}
	defer r.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	return buf.String(), nil
}
