# Zap OTLP


inspired from https://github.com/MrAlias/otlpr

otlp encoder and sync for zap



## To send data to signoz cloud

Set these environment variables

```
OTEL_EXPORTER_OTLP_HEADERS=signoz-access-token=<SIGNOZ_INGESTION_KEY>
OTEL_EXPORTER_OTLP_INSECURE=false
```

Replace `<SIGNOZ_INGESTION_KEY>` with your ingestion token
