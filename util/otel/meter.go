package otel

import (
	"github.com/prometheus/client_golang/prometheus"

	promex "go.opentelemetry.io/otel/exporters/prometheus"
	otel "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdk "go.opentelemetry.io/otel/sdk/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

func MeterProvider(reg prometheus.Registerer) (otel.MeterProvider, error) {
	if reg == nil {
		return noop.NewMeterProvider(), nil
	}

	ex, err := promex.New(promex.WithRegisterer(reg))
	if err != nil {
		return nil, err
	}

	return sdk.NewMeterProvider(sdk.WithReader(ex)), nil
}

func TelemetrySettings(reg prometheus.Registerer) (component.TelemetrySettings, error) {
	var set component.TelemetrySettings

	mp, err := MeterProvider(reg)
	if err != nil {
		return set, err
	}

	set.MeterProvider = mp
	set.LeveledMeterProvider = func(_ configtelemetry.Level) otel.MeterProvider {
		return mp
	}
	return set, nil
}
