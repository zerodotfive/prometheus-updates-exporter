package exporter

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	fqdn "github.com/Showmax/go-fqdn"
	"github.com/prometheus/client_golang/prometheus"
)

type UpdatesExporter struct {
	mutex sync.RWMutex

	package_upgradable *prometheus.GaugeVec
	reboot_required    prometheus.Gauge
}

func (self *UpdatesExporter) Update(ctx context.Context) {
	cmd := exec.CommandContext(ctx, "apt", "list", "--upgradable")
	cmd.Env = append(os.Environ(), "LC_ALL=C")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		log.Printf("APT list error: %v: %s", err, stderr.String())
		return
	}

	packages := make(map[string]struct{})

	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		if !strings.Contains(line, "upgradable from:") {
			continue
		}

		name, _, found := strings.Cut(fields[0], "/")
		if !found || name == "" {
			continue
		}

		packages[name] = struct{}{}
	}

	_, err = os.Stat("/var/run/reboot-required")
	rebootRequired := 0.0
	if err == nil {
		rebootRequired = 1
	} else if !os.IsNotExist(err) {
		log.Printf("Reboot required check error: %v", err)
		return
	}

	self.mutex.Lock()
	defer self.mutex.Unlock()

	self.package_upgradable.Reset()

	for name := range packages {
		self.package_upgradable.WithLabelValues(name).Set(1)
	}

	self.reboot_required.Set(rebootRequired)

	log.Printf("found %d upgradable packages, reboot required: %.0f", len(packages), rebootRequired)
}

func (self *UpdatesExporter) Run(ctx context.Context) {
	self.Update(ctx)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			self.Update(ctx)
		}
	}
}

func (self *UpdatesExporter) Collect(ch chan<- prometheus.Metric) {
	self.mutex.RLock()
	defer self.mutex.RUnlock()

	self.package_upgradable.Collect(ch)
	ch <- self.reboot_required
}

func (self *UpdatesExporter) Describe(ch chan<- *prometheus.Desc) {
	self.package_upgradable.Describe(ch)
	ch <- self.reboot_required.Desc()
}

func CreateUpdatesExporter() (*UpdatesExporter, error) {
	hostname, err := fqdn.FqdnHostname()
	if err != nil {
		return nil, err
	}

	return &UpdatesExporter{
		package_upgradable: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "updates_exporter",
			Name:        "package_upgradable",
			Help:        "Whether an package has an available upgrade",
			ConstLabels: prometheus.Labels{"hostname": hostname},
		}, []string{"package"}),

		reboot_required: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   "updates_exporter",
			Name:        "reboot_required",
			Help:        "Whether a system reboot is required",
			ConstLabels: prometheus.Labels{"hostname": hostname},
		}),
	}, nil
}
