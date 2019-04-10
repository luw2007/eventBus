// Package eventBus 事件订阅器
package eventBus

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
)

// A Emitter 订阅器接口
// On/Once 绑定事件，绑定成功返回true，重复绑定返回false
// Once 执行一次后自动清除，On 可以一直执行
// Send 调用具体绑定方法实例
// remove 移除事件
type Emitter interface {
	On(eventKey string, call interface{}) error
	Once(eventKey string, call interface{}) error
	Send(eventKey string, args ...interface{}) error
	Remove(eventKey string)
}

var (
	ErrArgsNotMatch = errors.New("the number of input args not match")
	ErrEventType    = errors.New("event type error")
	ErrExists       = errors.New("event already exists")
	ErrNotCallable  = errors.New("event not callable")
	ErrNotFound     = errors.New("event not found")
	ErrRuntimePanic = errors.New("event runtime recover a panic")
)

// a event 事件，保存了事件的类型，名称和调用方法
type event struct {
	key       string
	once      bool
	call      interface{}
	args      []reflect.Value
	callTimes int32
}

// Call 事件执行方法
func (e *event) Call(args []interface{}) (err error) {
	if atomic.AddInt32(&e.callTimes, 1) > 1 && e.once {
		return ErrNotFound
	}
	// 构造入参
	f := reflect.ValueOf(e.call)
	for k, v := range args {
		e.args[k] = reflect.ValueOf(v)
	}
	defer func() {
		rec := recover()
		if rec != nil {
			stack := make([]byte, 4<<10) // 4 KB
			length := runtime.Stack(stack, true)
			fmt.Printf("[PANIC RECOVER] call %s panic: %s %s\n", e.key, rec, stack[:length])
			err = ErrRuntimePanic
		}
	}()
	f.Call(e.args)
	return err
}

func (e *event) String() string {
	return fmt.Sprintf("{key: %s, once: %t, callTimes: %d}", e.key, e.once, e.callTimes)
}

// EventBus 事件订阅器
type EventBus struct {
	// events 储存结构类似 map[string]*event,
	events sync.Map
}

// On 注册订阅器，注册之后将实例放入 events中。在Send中调用
func (p *EventBus) On(eventKey string, call interface{}) error {
	return p.on(&event{key: eventKey, call: call})
}

// Once 仅用一次注册订阅器，注册之后将实例放入 events中。在Send中调用，调用之后立即移除掉
func (p *EventBus) Once(eventKey string, call interface{}) error {
	return p.on(&event{key: eventKey, once: true, call: call})
}

func (p *EventBus) on(e *event) error {
	f := reflect.ValueOf(e.call)
	if f.Kind() != reflect.Func {
		return ErrNotCallable
	}
	// 初始化入参，每次send 都会从入参中重新填充
	e.args = make([]reflect.Value, f.Type().NumIn())
	if _, ok := p.events.LoadOrStore(e.key, e); ok {
		return ErrExists
	}
	return nil
}

// Send 调用事件，执行后注销once事件
func (p *EventBus) Send(eventKey string, args ...interface{}) error {
	m, ok := p.events.Load(eventKey)
	if !ok {
		return ErrNotFound
	}
	e, ok := m.(*event)
	if !ok {
		// 永远不会发生
		return ErrEventType
	}
	if len(args) != len(e.args) {
		return ErrArgsNotMatch
	}
	if e.once {
		// 这里不关心处理结果删除一次注册，
		// 如果需要保证成功执行，需要放到Call后面，判断Call结果
		defer p.Remove(eventKey)
	}
	return e.Call(args)
}

// Remove 移除事件
func (p *EventBus) Remove(eventkey string) {
	p.events.Delete(eventkey)
}

// 构建一个事件订阅器
func New() *EventBus {
	return &EventBus{}
}
