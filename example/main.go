// similar to https://github.com/MrAlias/otlpr/blob/main/example/main.go

package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"

	zapotlpencoder "github.com/SigNoz/zap_otlp/zap_otlp_encoder"
	zapotlpsync "github.com/SigNoz/zap_otlp/zap_otlp_sync"
)

var targetPtr = flag.String("target", "127.0.0.1:4317", "OTLP target")

type App struct {
	logger *zap.Logger
}

func NewApp(logger *zap.Logger) App {
	return App{logger: logger}
}

func (a App) Hello(ctx context.Context, user string) error {
	if user == "" {
		return errors.New("no user name provided")
	}
	return nil
}

func setup(ctx context.Context, conn *grpc.ClientConn) (*zap.Logger, error) {
	config := zap.NewProductionEncoderConfig()
	consoleEncoder := zapotlpencoder.NewOTLPEncoder(config)
	defaultLogLevel := zapcore.DebugLevel

	ws := zapcore.AddSync(zapotlpsync.NewOtlpSyncer(conn, map[string]interface{}{"service": "myservice"}))
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), ws), defaultLogLevel),
	)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	conn, err := grpc.DialContext(ctx, *targetPtr, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	logger, err := setup(ctx, conn)
	if err != nil {
		log.Fatal(err)
	}

	app := NewApp(logger)
	for _, user := range []string{"alice", ""} {
		if err := app.Hello(ctx, user); err != nil {
			logger.Error("failed to do xyz" + user)
		}
	}

	logger.Sync()
}
