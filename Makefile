all: deps prometheus-updates-exporter

deps:
	go get ./cmd/prometheus-updates-exporter

prometheus-updates-exporter:
	go build -o ./bin/prometheus-updates-exporter ./cmd/prometheus-updates-exporter

clean:
	rm -f ./bin/prometheus-updates-exporter
