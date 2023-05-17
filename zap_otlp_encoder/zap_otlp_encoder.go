package zap_otlp_encoder

// adapted from https://github.com/uber-go/zap/blob/master/zapcore/json_encoder.go
// and https://github.com/uber-go/zap/blob/master/zapcore/console_encoder.go

import (
	"encoding/base64"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"
)

const _hex = "0123456789abcdef"

var bufferPool = buffer.NewPool()

var _otlpPool = sync.Pool{New: func() interface{} {
	return &otlpEncoder{}
}}

func getOTLPEncoder() *otlpEncoder {
	return _otlpPool.Get().(*otlpEncoder)
}

func putOTLPEncoder(enc *otlpEncoder) {
	enc.EncoderConfig = nil
	enc.buf = nil
	enc.spaced = false
	enc.openNamespaces = 0
	// enc.reflectBuf = nil
	// enc.reflectEnc = nil
	enc.log = nil
	_otlpPool.Put(enc)
}

type otlpEncoder struct {
	*zapcore.EncoderConfig
	buf            *buffer.Buffer
	spaced         bool
	openNamespaces int

	log *lpb.LogRecord

	// for encoding generic values by reflection
	// reflectBuf *buffer.Buffer
	// reflectEnc zapcore.ReflectedEncoder
}

// NewOTLPEncoder creates a OTLP encoder
func NewOTLPEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	return &otlpEncoder{
		EncoderConfig: &cfg,
		buf:           bufferPool.Get(),
		log:           &lpb.LogRecord{},
	}
}

func (enc *otlpEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	// todo : implement this later
	return nil
}

func (enc *otlpEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	// todo : implement this later
	return nil
}

func (enc *otlpEncoder) AddBinary(key string, val []byte) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: base64.StdEncoding.EncodeToString(val)}})
}

func (enc *otlpEncoder) AddByteString(key string, val []byte) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: string(val)}})
}

func (enc *otlpEncoder) AddString(key, val string) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: val}})
}

func (enc *otlpEncoder) AddBool(key string, val bool) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_BoolValue{BoolValue: val}})
}

func (enc *otlpEncoder) AddComplex128(key string, val complex128) {
	// todo: implement this later
}

func (enc *otlpEncoder) AddComplex64(key string, val complex64) {
	// todo: implement this later
}

func (enc *otlpEncoder) AddDuration(key string, val time.Duration) {
	// TODO: allow user to specify the unit of duration using DurationEncoder in config.EncodeDuration

	// all duration will be in milliseconds
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: int64((val.Nanoseconds() / 1e6))}})
}

func (enc *otlpEncoder) AddFloat64(key string, val float64) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: val}})
}

func (enc *otlpEncoder) AddFloat32(key string, val float32) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: float64(val)}})
}

func (enc *otlpEncoder) AddInt64(key string, val int64) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: val}})
}

func (enc *otlpEncoder) AddUint64(key string, val uint64) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: int64(val)}})
}

func (enc *otlpEncoder) AddTime(key string, val time.Time) {
	// not required
}

func (enc *otlpEncoder) AddReflected(key string, obj interface{}) error {
	return nil
}

func (enc *otlpEncoder) OpenNamespace(key string) {
	// todo: implement this later
}

func (enc *otlpEncoder) AddInt(k string, v int)         {}
func (enc *otlpEncoder) AddInt32(k string, v int32)     {}
func (enc *otlpEncoder) AddInt16(k string, v int16)     {}
func (enc *otlpEncoder) AddInt8(k string, v int8)       {}
func (enc *otlpEncoder) AddUint(k string, v uint)       {}
func (enc *otlpEncoder) AddUint32(k string, v uint32)   {}
func (enc *otlpEncoder) AddUint16(k string, v uint16)   {}
func (enc *otlpEncoder) AddUint8(k string, v uint8)     {}
func (enc *otlpEncoder) AddUintptr(k string, v uintptr) {}

func (enc *otlpEncoder) Clone() zapcore.Encoder {
	clone := enc.clone()
	clone.buf.Write(enc.buf.Bytes())
	clone.log = &lpb.LogRecord{}
	*clone.log = *enc.log
	return clone
}

func (enc *otlpEncoder) clone() *otlpEncoder {
	clone := getOTLPEncoder()
	clone.EncoderConfig = enc.EncoderConfig
	clone.spaced = enc.spaced
	clone.openNamespaces = enc.openNamespaces
	clone.buf = bufferPool.Get()
	clone.log = &lpb.LogRecord{}
	return clone
}

var levelMap = map[string]uint{
	"trace": 1,
	"debug": 5,
	"info":  9,
	"warn":  13,
	"error": 17,
	"fatal": 21,
}

func (enc *otlpEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	final := enc.clone()
	if final.LevelKey != "" && final.EncodeLevel != nil {
		final.log.SeverityNumber = lpb.SeverityNumber(levelMap[ent.Level.String()])
		final.log.SeverityText = ent.Level.String()
	}
	if final.TimeKey != "" {
		final.log.TimeUnixNano = uint64(ent.Time.UnixNano())
	}
	if ent.LoggerName != "" && final.NameKey != "" {
		final.log.Attributes = append(enc.log.Attributes, &v1.KeyValue{
			Key: "logger",
			Value: &v1.AnyValue{
				Value: &v1.AnyValue_StringValue{
					StringValue: ent.LoggerName,
				},
			},
		})
	}
	if ent.Caller.Defined {
		if final.CallerKey != "" {
			final.addKeyVal(final.CallerKey, &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: ent.Caller.String()}})
		}
		if final.FunctionKey != "" {
			final.addKeyVal(final.FunctionKey, &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: ent.Caller.Function}})
		}
	}
	if final.MessageKey != "" {
		final.log.Body = &v1.AnyValue{
			Value: &v1.AnyValue_StringValue{
				StringValue: ent.Message,
			},
		}
	}

	addFields(final, fields)
	data, err := proto.Marshal(final.log)
	if err != nil {
		panic(err)
	}

	final.buf.AppendString(string(data))
	ret := final.buf
	putOTLPEncoder(final)

	return ret, nil
}

func (enc *otlpEncoder) addKeyVal(key string, val *v1.AnyValue) {
	if key == "trace_id" {
		traceId, err := trace.TraceIDFromHex(val.GetStringValue())
		if err == nil {
			enc.log.TraceId = traceId[:]
		}
		return
	}
	if key == "span_id" {
		spanId, err := trace.SpanIDFromHex(val.GetStringValue())
		if err == nil {
			enc.log.SpanId = spanId[:]
		}
		return
	}

	enc.log.Attributes = append(enc.log.Attributes, &v1.KeyValue{
		Key:   key,
		Value: val,
	})
}

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}
