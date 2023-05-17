package zap_otlp_sync

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	collpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	cv1 "go.opentelemetry.io/proto/otlp/common/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"
	rv1 "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type OtelSyncer struct {
	ctx       context.Context
	close     context.CancelFunc
	res       *rv1.Resource
	resSchema string
	scope     *cv1.InstrumentationScope
	values    [][]byte
	queue     chan []byte

	client     collpb.LogsServiceClient
	batchSize  int
	sendBatch  chan bool
	valueMutex sync.Mutex
	pushDataWg sync.WaitGroup
}

type Options struct {
	BatchSize      int
	BatchInterval  time.Duration
	ResourceSchema string
	Scope          *instrumentation.Scope
	Resource       *resource.Resource
}

func NewOtlpSyncer(conn *grpc.ClientConn, options Options) *OtelSyncer {
	ctx, cancel := context.WithCancel(context.Background())

	if options.BatchSize == 0 {
		options.BatchSize = 100
	}

	if options.BatchInterval == 0 {
		options.BatchInterval = time.Duration(5) * time.Second
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

	syncer := &OtelSyncer{
		ctx:        ctx,
		close:      cancel,
		res:        rattrs,
		resSchema:  options.ResourceSchema,
		scope:      instrumentationScope,
		values:     [][]byte{},
		queue:      make(chan []byte, options.BatchSize), // keeping queue size as batch size
		client:     collpb.NewLogsServiceClient(conn),
		batchSize:  options.BatchSize,
		sendBatch:  make(chan bool),
		valueMutex: sync.Mutex{},
		pushDataWg: sync.WaitGroup{},
	}

	go syncer.processQueue(options.BatchInterval)

	return syncer
}

func (l *OtelSyncer) processQueue(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			l.pushData()
		case data := <-l.queue:
			l.valueMutex.Lock()
			l.values = append(l.values, data)
			shouldExport := len(l.values) >= l.batchSize
			l.valueMutex.Unlock()
			if shouldExport {
				l.pushData()
			}
		case <-l.sendBatch:
			l.pushData()
		}
	}
}

func (l *OtelSyncer) pushData() (err error) {
	l.pushDataWg.Add(1)
	defer l.pushDataWg.Done()

	rec := []*lpb.LogRecord{}

	l.valueMutex.Lock()
	for _, v := range l.values {
		r := &lpb.LogRecord{}
		err = proto.Unmarshal([]byte(v), r)
		if err != nil {
			l.valueMutex.Unlock()
			return err
		}
		rec = append(rec, r)
	}
	l.values = [][]byte{} // clean the values
	l.valueMutex.Unlock()

	if len(rec) == 0 {
		return nil
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

	// just add the data to the queue
	l.queue <- data
	return len(data), nil
}

func (l *OtelSyncer) Sync() error {
	l.sendBatch <- true
	return nil
}

func (l *OtelSyncer) Close() error {
	err := l.Sync()
	if err != nil {
		return err
	}
	l.close()
	l.pushDataWg.Wait()
	return nil
}
