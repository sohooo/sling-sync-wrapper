package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"sling-sync-wrapper/internal/logging"
)

// Init sets up an OTEL tracer and returns it along with a shutdown function.
func Init(ctx context.Context, serviceName, missionClusterID, endpoint string) (trace.Tracer, func(context.Context) error) {
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint))
	if err != nil {
		logging.FromContext(ctx).Error("failed to create OTLP trace exporter", "err", err)
		os.Exit(1)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("mission_cluster_id", missionClusterID),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp.Tracer(serviceName), tp.Shutdown
}
