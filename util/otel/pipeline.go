package otel

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
)

type Step = func(set Settings, next consumer.Metrics) (consumer.Metrics, error)

func Sink(ex consumer.Metrics) Step {
	return func(_ Settings, _ consumer.Metrics) (consumer.Metrics, error) {
		return ex, nil
	}
}

func Processor(fac processor.Factory, cfg component.Config) Step {
	if cfg == nil {
		cfg = fac.CreateDefaultConfig()
	}
	return func(set Settings, next consumer.Metrics) (consumer.Metrics, error) {
		return fac.CreateMetrics(context.Background(), set.Processor(), cfg, next)
	}
}

func Exporter(fac exporter.Factory, cfg component.Config) Step {
	if cfg == nil {
		cfg = fac.CreateDefaultConfig()
	}
	return func(set Settings, _ consumer.Metrics) (consumer.Metrics, error) {
		return fac.CreateMetricsExporter(context.Background(), set.Exporter(), cfg)
	}
}

func Use(reg prometheus.Registerer, steps ...Step) (Pipeline, error) {
	pl := pipeline{steps: make([]consumer.Metrics, len(steps))}

	tel, err := TelemetrySettings(reg)
	if err != nil {
		return nil, err
	}

	var next consumer.Metrics
	for i := len(steps) - 1; i >= 0; i-- {
		set := Settings{
			TelemetrySettings: tel,
		}

		create := steps[i]
		pl.steps[i], err = create(set, next)
		if err != nil {
			return nil, err
		}
		next = pl.steps[i]
	}

	return pl, nil
}

type Pipeline interface {
	component.Component
	consumer.Metrics
}

var _ Pipeline = pipeline{}

type pipeline struct {
	steps []consumer.Metrics
}

func (pl pipeline) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	if pl.steps == nil {
		return nil
	}
	return pl.steps[0].ConsumeMetrics(ctx, md)
}

func (pl pipeline) Start(ctx context.Context, host component.Host) error {
	for _, step := range pl.steps {
		if c, ok := step.(component.Component); ok {
			err := c.Start(ctx, host)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (pl pipeline) Shutdown(ctx context.Context) error {
	var errs error
	for _, step := range pl.steps {
		if c, ok := step.(component.Component); ok {
			err := c.Shutdown(ctx)
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

func (pl pipeline) Capabilities() consumer.Capabilities {
	for _, step := range pl.steps {
		if step.Capabilities().MutatesData {
			return step.Capabilities()
		}
	}
	return consumer.Capabilities{MutatesData: false}
}

type Settings processor.Settings

func (s Settings) Processor() processor.Settings {
	return processor.Settings(s)
}

func (s Settings) Exporter() exporter.Settings {
	return exporter.Settings(s)
}
