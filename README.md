# Zap OTLP

inspired from https://github.com/MrAlias/otlpr

This plugin helps you send logs from [zap](https://github.com/uber-go/zap) logger to a OTLP endpoint in your Go application.

## Getting Started
* Create two encoders. One for console and the other for OTLP.
    ```
  	otlpEncoder := zapotlpencoder.NewOTLPEncoder(config)
	consoleEncoder := zapcore.NewConsoleEncoder(config)
    ```
* Initialize the OTLP sync to send data to OTLP endpoint
    ```
    var targetPtr = flag.String("target", "127.0.0.1:4317", "OTLP target")
    var grpcInsecure = os.Getenv("OTEL_EXPORTER_OTLP_INSECURE")

    ...

    var secureOption grpc.DialOption
	if strings.ToLower(grpcInsecure) == "false" || grpcInsecure == "0" || strings.ToLower(grpcInsecure) == "f" {
		secureOption = grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))
	} else {
		secureOption = grpc.WithTransportCredentials(insecure.NewCredentials())
	}
    conn, err := grpc.DialContext(ctx, *targetPtr, grpc.WithBlock(), secureOption, grpc.WithTimeout(time.Duration(5)*time.Second))
	if err != nil {
		log.Fatal(err)
	}

    otlpSync := zapotlpsync.NewOtlpSyncer(conn, zapotlpsync.Options{
		BatchSize:      100,
		ResourceSchema: semconv.SchemaURL,
		Resource:       resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("example application"),
        ),
	})
    ```
* Configure zap to use the encoders and sync.
    ```
    core := zapcore.NewTee(
        zapcore.NewCore(consoleEncoder, os.Stdout, defaultLogLevel),
        zapcore.NewCore(otlpEncoder, otlpSync, defaultLogLevel),
    )
    logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
    ```
See the [example](./example/main.go) for a working example application.

## Configurations
1) Batching
   The default batch size is 100, you can change the batch size in `zapotlpsync.Options`
    ```
    zapotlpsync.Options{
        BatchSize:      <batch size>,
        .....
    ```
2) BatchInterval:
    Time duration after which a batch will be sent regardless of size. Default is 5seconds. You can 
    change it in `zapotlpsync.Options`
    ```
    zapotlpsync.Options{
		BatchInterval:      <interval_in_secs>,
        .....
    ```

## Send Trace Details
* The trace details are not populated automatically. You will have to add it in your log lines by passing the context to `zapotlp.SpanCtx()`
  ```
  a.logger.Named("test_logger").Info("trace_test", zapotlp.SpanCtx(ctx))
  ```

## To send data to signoz cloud

Set these environment variables

```
OTEL_EXPORTER_OTLP_HEADERS=signoz-access-token=<SIGNOZ_INGESTION_KEY>
OTEL_EXPORTER_OTLP_INSECURE=false
```

Replace `<SIGNOZ_INGESTION_KEY>` with your ingestion token

target = ingest.{region}.signoz.cloud:443
