module github.com/SigNoz/zap_otlp/example

go 1.19

replace (
	github.com/SigNoz/zap_otlp/zap_otlp_encoder => ../zap_otlp_encoder

	github.com/SigNoz/zap_otlp/zap_otlp_sync => ../zap_otlp_sync
)

require (
	github.com/SigNoz/zap_otlp/zap_otlp_encoder v0.0.0-00010101000000-000000000000
	github.com/SigNoz/zap_otlp/zap_otlp_sync v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.24.0
	google.golang.org/grpc v1.55.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.7.0 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/goleak v1.2.1 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230306155012-7f2fa6fef1f4 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)
