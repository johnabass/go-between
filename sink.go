package hub

import (
	"errors"
	"reflect"
)

var (
	ErrInvalidListener  = errors.New("A listener must be a function, channel, or have exactly (1) method")
	ErrInvalidFunction  = errors.New("A listener function or method must have exactly (1) input and no outputs")
	ErrInvalidChannel   = errors.New("A listener channel must be bidirectional or send-only")
	ErrInvalidEventType = errors.New("Event types cannot be interfaces")
)

// sink is an internal strategy interface for sending event objects to destinations
type sink interface {
	send(reflect.Value)
}

type sinkFunc struct {
	f reflect.Value
}

func (sf *sinkFunc) send(v reflect.Value) {
	sf.f.Call([]reflect.Value{v})
}

type sinkChan struct {
	c reflect.Value
}

func (sc *sinkChan) send(v reflect.Value) {
	sc.c.Send(v)
}

type sinkMethod struct {
	r reflect.Value // receiver
	m reflect.Value // method function itself
}

func (sm *sinkMethod) send(v reflect.Value) {
	sm.m.Call(
		[]reflect.Value{sm.r, v},
	)
}

// newSink reflects on t and determines the event type and a sink strategy for sending the event
func newSink(t interface{}) (reflect.Type, sink, error) {
	listenerType := reflect.TypeOf(t)
	if listenerType == nil {
		return nil, nil, ErrInvalidListener
	}

	var (
		eventType reflect.Type
		s         sink
	)

	switch {
	case listenerType.Kind() == reflect.Func:
		if listenerType.NumIn() != 1 || listenerType.NumOut() != 0 {
			return nil, nil, ErrInvalidFunction
		}

		eventType = listenerType.In(0)
		s = &sinkFunc{f: reflect.ValueOf(t)}

	case listenerType.Kind() == reflect.Chan:
		if listenerType.ChanDir() == reflect.RecvDir {
			return nil, nil, ErrInvalidChannel
		}

		eventType = listenerType.Elem()
		s = &sinkChan{c: reflect.ValueOf(t)}

	case listenerType.NumMethod() == 1:
		m := listenerType.Method(0)

		// for a method, we include the receiver, which is the first parameter
		// that means the number of in parameters should be (2), with the event
		// object being the second parameter
		if m.Func.Type().NumIn() != 2 || m.Func.Type().NumOut() != 0 {
			return nil, nil, ErrInvalidFunction
		}

		eventType = m.Func.Type().In(1)
		s = &sinkMethod{r: reflect.ValueOf(t), m: m.Func}

	default:
		return nil, nil, ErrInvalidListener
	}

	if eventType.Kind() == reflect.Interface {
		return nil, nil, ErrInvalidEventType
	}

	return eventType, s, nil
}
