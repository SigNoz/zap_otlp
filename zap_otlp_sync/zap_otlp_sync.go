package zap_otlp_sync

import (
	"context"

	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	collpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	cv1 "go.opentelemetry.io/proto/otlp/common/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"
	rv1 "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type Options struct {
	BatchSize      int
	ResourceSchema string
	Scope          *instrumentation.Scope
	Resource       *resource.Resource
}

func NewOtlpSyncer(conn *grpc.ClientConn, options Options) *OtelSyncer {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if options.BatchSize == 0 {
		options.BatchSize = 100
	}

	var rattrs *rv1.Resource
	if options.Resource != nil {
		iter := options.Resource.Iter()
		attrs := make([]*cv1.KeyValue, 0, iter.Len())
		for iter.Next() {
			attr := iter.Attribute()
			attrs = append(attrs, &cv1.KeyValue{
				Key:   string(attr.Key),
				Value: &cv1.AnyValue{Value: &cv1.AnyValue_StringValue{StringValue: attr.Value.Emit()}},
			})
		}
		rattrs = &rv1.Resource{Attributes: attrs}
	}

	var instrumentationScope *cv1.InstrumentationScope
	if options.Scope != nil {
		instrumentationScope = &cv1.InstrumentationScope{
			Name:    options.Scope.Name,
			Version: options.Scope.Version,
		}
	}

	return &OtelSyncer{
		ctx,
		cancel,
		rattrs,
		options.ResourceSchema,
		instrumentationScope,
		[][]byte{},
		collpb.NewLogsServiceClient(conn),
		options.BatchSize,
	}
}

type OtelSyncer struct {
	ctx       context.Context
	close     context.CancelFunc
	res       *rv1.Resource
	resSchema string
	scope     *cv1.InstrumentationScope
	values    [][]byte

	client    collpb.LogsServiceClient
	batchSize int
}

func (l OtelSyncer) pushToSigNoz() (err error) {
	rec := []*lpb.LogRecord{}

	for _, v := range l.values {
		r := &lpb.LogRecord{}
		err = proto.Unmarshal([]byte(v), r)
		if err != nil {
			return err
		}
		rec = append(rec, r)
	}

	sl := &lpb.ScopeLogs{LogRecords: rec}
	if l.scope != nil {
		sl.SchemaUrl, sl.Scope = l.resSchema, l.scope
	}

	rl := &lpb.ResourceLogs{ScopeLogs: []*lpb.ScopeLogs{sl}}
	if l.res != nil {
		rl.SchemaUrl, rl.Resource = l.resSchema, l.res
	}
	_, err = l.client.Export(context.Background(), &collpb.ExportLogsServiceRequest{
		ResourceLogs: []*lpb.ResourceLogs{rl},
	})
	// TODO: how to handle partial failure and error
	if err != nil {
		return err
	}

	return nil
}

func (l *OtelSyncer) Write(record []byte) (n int, err error) {
	// create deep copyy of the data
	data := make([]byte, len(record))
	copy(data, record)
	l.values = append(l.values, data)
	if len(l.values) >= l.batchSize {
		err := l.Sync()
		if err != nil {
			return 0, err
		}
		l.values = [][]byte{}
	}
	return len(record), nil
}

func (l OtelSyncer) Sync() error {
	if err := l.pushToSigNoz(); err != nil {
		return err
	}
	return nil
}

func (l OtelSyncer) Close() error {
	l.close()
	return nil
}
