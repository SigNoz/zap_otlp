package zap_otlp_encoder

// adapted from https://github.com/uber-go/zap/blob/master/zapcore/json_encoder.go
// and https://github.com/uber-go/zap/blob/master/zapcore/console_encoder.go

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"sync"
	"time"
	"unicode/utf8"

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
	enc.reflectBuf = nil
	enc.reflectEnc = nil
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
	reflectBuf *buffer.Buffer
	reflectEnc zapcore.ReflectedEncoder
}

// NewkvEncoder creates a key=value encoder
func NewOTLPEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	if cfg.SkipLineEnding {
		cfg.LineEnding = ""
	} else if cfg.LineEnding == "" {
		cfg.LineEnding = zapcore.DefaultLineEnding
	}

	// If no EncoderConfig.NewReflectedEncoder is provided by the user, then use default
	if cfg.NewReflectedEncoder == nil {
		cfg.NewReflectedEncoder = defaultReflectedEncoder
	}

	return &otlpEncoder{
		EncoderConfig: &cfg,
		buf:           bufferPool.Get(),
		log:           &lpb.LogRecord{},
	}
}

func (enc *otlpEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	// todo : implement this
	return nil
}

func (enc *otlpEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	// todo : implement this
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
	// todo: implement this
}

func (enc *otlpEncoder) AddComplex64(key string, val complex64) {
	// todo: implement this
}

func (enc *otlpEncoder) AddDuration(key string, val time.Duration) {
	// todo: implement this
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
	// todo: implement this
}

func (enc *otlpEncoder) AddReflected(key string, obj interface{}) error {
	return nil
}

func (enc *otlpEncoder) OpenNamespace(key string) {
	// enc.addKey(key)
	// enc.buf.AppendByte('{')
	// enc.openNamespaces++
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
		// final.AddTime(final.TimeKey, ent.Time)
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

		// final.addKey(final.NameKey)
		// cur := final.buf.Len()
		// nameEncoder := final.EncodeName

		// // if no name encoder provided, fall back to FullNameEncoder for backwards
		// // compatibility
		// if nameEncoder == nil {
		// 	nameEncoder = zapcore.FullNameEncoder
		// }

		// nameEncoder(ent.LoggerName, final)
		// if cur == final.buf.Len() {
		// 	// User-supplied EncodeName was a no-op. Fall back to strings to
		// 	// keep output JSON valid.
		// 	final.AppendString(ent.LoggerName)
		// }
	}
	// if ent.Caller.Defined {
	// 	if final.CallerKey != "" {
	// 		final.addKey(final.CallerKey)
	// 		cur := final.buf.Len()
	// 		final.EncodeCaller(ent.Caller, final)
	// 		if cur == final.buf.Len() {
	// 			// User-supplied EncodeCaller was a no-op. Fall back to strings to
	// 			// keep output JSON valid.
	// 			final.AppendString(ent.Caller.String())
	// 		}
	// 	}
	// 	if final.FunctionKey != "" {
	// 		final.addKey(final.FunctionKey)
	// 		final.AppendString(ent.Caller.Function)
	// 	}
	// }
	if final.MessageKey != "" {
		final.log.Body = &v1.AnyValue{
			Value: &v1.AnyValue_StringValue{
				StringValue: ent.Message,
			},
		}
		// final.addKey(enc.MessageKey)
		// final.addKey("message")
		// final.AppendString(ent.Message)
	}
	// if enc.buf.Len() > 0 {
	// 	final.addElementSeparator()
	// 	final.buf.Write(enc.buf.Bytes())
	// }
	addFields(final, fields)
	// final.closeOpenNamespaces()
	// if ent.Stack != "" && final.StacktraceKey != "" {
	// 	final.AddString(final.StacktraceKey, ent.Stack)
	// }
	// final.buf.AppendByte('}')
	// final.buf.AppendString(final.LineEnding)

	// b, _ := json.Marshal(final.log)
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

func (enc *otlpEncoder) addElementSeparator() {
	last := enc.buf.Len() - 1
	if last < 0 {
		return
	}
	switch enc.buf.Bytes()[last] {
	case '{', '[', ':', ',', ' ':
		return
	default:
		enc.buf.AppendByte(',')
		if enc.spaced {
			enc.buf.AppendByte(' ')
		}
	}
}

func (enc *otlpEncoder) appendFloat(val float64, bitSize int) {
	switch {
	case math.IsNaN(val):
		enc.buf.AppendString(`"NaN"`)
	case math.IsInf(val, 1):
		enc.buf.AppendString(`"+Inf"`)
	case math.IsInf(val, -1):
		enc.buf.AppendString(`"-Inf"`)
	default:
		enc.buf.AppendFloat(val, bitSize)
	}
}

// safeAddString JSON-escapes a string and appends it to the internal buffer.
// Unlike the standard library's encoder, it doesn't attempt to protect the
// user from browser vulnerabilities or JSONP-related problems.
func (enc *otlpEncoder) safeAddString(s string) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.AppendString(s[i : i+size])
		i += size
	}
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s []byte.
func (enc *otlpEncoder) safeAddByteString(s []byte) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.Write(s[i : i+size])
		i += size
	}
}

// tryAddRuneSelf appends b if it is valid UTF-8 character represented in a single byte.
func (enc *otlpEncoder) tryAddRuneSelf(b byte) bool {
	if b >= utf8.RuneSelf {
		return false
	}
	if b >= 0x20 && b != '\\' && b != '"' {
		enc.buf.AppendByte(b)
		return true
	}
	switch b {
	case '\\', '"':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte(b)
	case '\n':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('n')
	case '\r':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('r')
	case '\t':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('t')
	default:
		// Encode bytes < 0x20, except for the escape sequences above.
		enc.buf.AppendString(`\u00`)
		enc.buf.AppendByte(_hex[b>>4])
		enc.buf.AppendByte(_hex[b&0xF])
	}
	return true
}

func (enc *otlpEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		enc.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}

func defaultReflectedEncoder(w io.Writer) zapcore.ReflectedEncoder {
	enc := json.NewEncoder(w)
	// For consistency with our custom JSON encoder.
	enc.SetEscapeHTML(false)
	return enc
}
