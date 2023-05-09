// similar to https://github.com/MrAlias/otlpr/blob/main/example/main.go

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"go.opentelemetry.io/otel/sdk/resource"
	sdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"

	zapotlp "github.com/SigNoz/zap_otlp"
	zapotlpencoder "github.com/SigNoz/zap_otlp/zap_otlp_encoder"
	zapotlpsync "github.com/SigNoz/zap_otlp/zap_otlp_sync"
)

var targetPtr = flag.String("target", "127.0.0.1:4317", "OTLP target")

const (
	lib    = "github.com/MrAlias/otlpr/example"
	libVer = "v0.1.0"
)

type App struct {
	logger *zap.Logger
	tracer trace.Tracer
}

func NewApp(tracer trace.Tracer, logger *zap.Logger) App {
	return App{tracer: tracer, logger: logger}
}

func (a App) Hello(ctx context.Context, user string) error {

	var span trace.Span
	ctx, span = a.tracer.Start(ctx, "Hello")
	defer span.End()

	a.logger.Info("hello from the function to user: "+user, zap.String("user", user), zapotlp.SpanCtx(ctx))

	return nil
}

func setup(ctx context.Context, conn *grpc.ClientConn) (trace.Tracer, *zap.Logger, error) {
	// exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	// if err != nil {
	// 	return nil, zap.NewNop(), err
	// }

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("example application"),
	)

	// Use a syncer for demo purposes only.
	// add sdk.WithSyncer(exp) for exporting traces
	tp := sdk.NewTracerProvider(sdk.WithResource(res))
	tracer := tp.Tracer(lib, trace.WithInstrumentationVersion(libVer))

	config := zap.NewProductionEncoderConfig()
	otlpEncoder := zapotlpencoder.NewOTLPEncoder(config)
	consoleEncoder := zapcore.NewConsoleEncoder(config)
	defaultLogLevel := zapcore.DebugLevel

	ws := zapcore.AddSync(zapotlpsync.NewOtlpSyncer(conn, 100, map[string]interface{}{"service": "myservice"}))
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, os.Stdout, defaultLogLevel),
		zapcore.NewCore(otlpEncoder, zapcore.NewMultiWriteSyncer(ws), defaultLogLevel),
	)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// l = otlpr.WithResource(l, res)
	// scope := instrumentation.Scope{Name: lib, Version: libVer}
	// l = otlpr.WithScope(l, scope)

	return tracer, logger, nil
}

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	conn, err := grpc.DialContext(ctx, *targetPtr, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	tracer, logger, err := setup(ctx, conn)
	if err != nil {
		log.Fatal(err)
	}

	var span trace.Span
	ctx, span = tracer.Start(ctx, "main")
	defer span.End()

	app := NewApp(tracer, logger)

	app.Hello(ctx, "user: xyz")
	app.Hello(ctx, "user: newuser")

	logger.Sync()
}
