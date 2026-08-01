// Package telemetry inicializa OpenTelemetry (Seção 20: logs, métricas e traces
// do próprio backend). Nesta fase os exporters escrevem em stdout — trocar por
// um OTLP exporter apontando para o collector é configuração, não redesenho.
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type Providers struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	Tracer         trace.Tracer
	Meter          metric.Meter
}

func Setup(ctx context.Context, serviceName string) (*Providers, func(context.Context) error, error) {
	// Deliberadamente não mesclado com resource.Default(): o SDK atual bundla
	// uma versão de semconv diferente da usada aqui, e resource.Merge falha
	// com "conflicting Schema URL" ao combinar dois schemas distintos. Um
	// resource com schema único evita o conflito; atributos adicionais podem
	// ser somados aqui no futuro usando a mesma versão de semconv.
	res := resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(serviceName))

	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, nil, fmt.Errorf("criar trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	metricExporter, err := stdoutmetric.New()
	if err != nil {
		return nil, nil, fmt.Errorf("criar metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	providers := &Providers{
		TracerProvider: tp,
		MeterProvider:  mp,
		Tracer:         tp.Tracer(serviceName),
		Meter:          mp.Meter(serviceName),
	}

	shutdown := func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return err
		}
		return mp.Shutdown(ctx)
	}

	return providers, shutdown, nil
}
