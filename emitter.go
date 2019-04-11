// Package eventbus 事件订阅器
package eventbus

import (
	"errors"
	"fmt"
	"reflect"
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
	argsNums  int
	callTimes int32
}

// Call 事件执行方法
func (e *event) Call(args []interface{}) (err error) {
	atomic.AddInt32(&e.callTimes, 1)
	// 构造入参
	f := reflect.ValueOf(e.call)
	in := make([]reflect.Value, f.Type().NumIn())
	for k, v := range args {
		in[k] = reflect.ValueOf(v)
	}
	defer func() {
		rec := recover()
		if rec != nil {
			fmt.Printf("[PANIC RECOVER] call %s panic: %s\n", e.key, rec)
			err = ErrRuntimePanic
		}
	}()
	f.Call(in)
	return
}

func (e *event) String() string {
	return fmt.Sprintf("{key: %s, once: %t, callTimes: %d}", e.key, e.once, e.callTimes)
}

type sender struct {
	e    *event
	args []interface{}
}

func (s *sender) Call() (err error) {
	return s.e.Call(s.args)
}

// EventBus 事件订阅器
type EventBus struct {
	// events 储存结构类似 map[string]*event,
	events sync.Map
	sender chan sender
	done   chan bool
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
	e.argsNums = f.Type().NumIn()
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
	if len(args) != e.argsNums {
		return ErrArgsNotMatch
	}
	if e.once {
		p.Remove(eventKey)
	}
	p.sender <- sender{e: e, args: args}
	return nil
}

// Remove 移除事件
func (p *EventBus) Remove(eventkey string) {
	p.events.Delete(eventkey)
}

// Close 发出停止信号
func (p *EventBus) Close() {
	p.done <- true
	// FIXME: 是否清理已经发不过来的任务？
}

// Loop 时间循环，后台消费sender的数据
func (p *EventBus) Loop() {
	for {
		select {
		case <-p.done:
			break
		case s := <-p.sender:
			go s.Call()
		}
	}
}

// New 构建一个事件订阅器
func New() *EventBus {
	bus := EventBus{
		sender: make(chan sender),
		done:   make(chan bool),
	}
	go bus.Loop()
	return &bus
}
