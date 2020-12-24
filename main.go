package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/afoninsky/semver-exporter/probers"
	"github.com/afoninsky/semver-exporter/probers/helm"
	"gopkg.in/yaml.v2"
)

var fListen = flag.String("listen", "127.0.0.1:8080", "The address to listen on for HTTP requests")
var fConfig = flag.String("config", "./probes.yaml", "Configuration file path")
var fEndpoint = flag.String("endpoint", "/metrics", "Metrics HTTP endpoint")

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

	if err := createProbers(cfg); err != nil {
		log.Fatal(err)
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

func createProbers(cfg probes) error {
	for name, probe := range cfg {
		log.Printf(`Testing %s with interval %s`, name, probe.Interval)
		switch probe.Type {
		case "helm":
			p, err := helm.New(name, probe.Config)
			if err != nil {
				return err
			}
			go startProber(name, p, probe.Interval)
		default:
			return fmt.Errorf(`type %s is not supported`, name)
		}
	}
	return nil
}

func startProber(name string, p probers.Prober, d time.Duration) {
	for {
		if err := p.Probe(); err != nil {
			log.Printf("[ERROR] %s: %v", name, err)
		}
		time.Sleep(d)
	}
}

// // 	// metrics.GetOrCreateCounter(`foo{bar="baz",aaa="b"}`).Set(1)
