package tracing

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	otelsdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config represents a tracing configuration used upon initialization.
type Config struct {
	ServiceName             string
	OtelCollectorEndpoint   string
	JaegerCollectorEndpoint string
	// 0.0 (none) to 1.0 (all)
	SamplingRatio float64
}

const (
	defaultTracerName = "otel-instrumented-demo-apps"
)

var defaultTracer = otel.Tracer(defaultTracerName)

func Default() trace.Tracer {
	return defaultTracer
}

func InitProvider(cfg *Config) (*otelsdk.TracerProvider, error) {
	var err error
	tp := &otelsdk.TracerProvider{}
	if cfg.OtelCollectorEndpoint != "" {
		tp, err = initOtelTracer(cfg)
	} else if cfg.JaegerCollectorEndpoint != "" {
		tp, err =  initJaegerTracer(cfg)
	}

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, err
}

func initJaegerTracer(cfg *Config) (*otelsdk.TracerProvider, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.JaegerCollectorEndpoint)))
	if err != nil {
		return nil, err
	}

	tp := otelsdk.NewTracerProvider(
		// Always be sure to batch in production.
		otelsdk.WithBatcher(exp),
		// Record information about this application in a Resource.
		otelsdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.ServiceName),
		)),
	)

	return tp, nil
}

func initOtelTracer(cfg *Config) (*otelsdk.TracerProvider, error) {
	conn, err := grpc.DialContext(context.Background(), cfg.OtelCollectorEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(context.Background(), otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := otelsdk.NewTracerProvider(
		otelsdk.WithSampler(otelsdk.AlwaysSample()),
		// Register the trace exporter with a TracerProvider, using a batch
		// span processor to aggregate spans before export.
		otelsdk.WithBatcher(traceExporter),
		otelsdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.ServiceName),
		)),
	)

	return tp, nil
}
