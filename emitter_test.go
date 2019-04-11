package eventbus

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Example() {
	events := New()
	defer events.Close()
	events.Once("once", func() {
		fmt.Printf("receive event once\n")
	})
	events.Send("once")
	events.Send("once")

	// Output:
	// receive event once
}

func add(a, b, c int) {
	fmt.Printf("receive event add, args are: a %d, b %d c %d\n", a, b, c)
}

func makePanic() {
	panic("raise")
}

func addOnce(a, b, c int) {
	fmt.Printf("receive event addOnce, args are: a %d, b %d c %d\n", a, b, c)
}

func TestEventBus_On(t *testing.T) {
	events := New()
	defer events.Close()
	err := events.On("add", add)
	assert.NoError(t, err)

	err = events.On("add", add)
	assert.EqualError(t, err, ErrExists.Error())
}

func TestEventBus_Once(t *testing.T) {
	events := New()
	defer events.Close()
	events.Once("addOnce", addOnce)

	err := events.Send("addOnce", 1, 2, 3)
	assert.NoError(t, err)
	err = events.Send("addOnce", 1, 2, 3)
	assert.EqualError(t, err, ErrNotFound.Error())

	// 再次注册
	events.Once("addOnce", addOnce)
	err = events.Send("addOnce", 1, 2, 3)
	assert.NoError(t, err)

}

func TestEventBus_Send(t *testing.T) {
	events := New()
	defer events.Close()
	events.On("add", add)

	err := events.Send("add", 1, 2, 3)
	assert.NoError(t, err)

	err = events.Send("add", 1, 2)
	assert.EqualError(t, err, ErrArgsNotMatch.Error())

	events.Send("add", 1, 2, 3)
	e, _ := events.events.Load("add")
	runtime.Gosched()
	assert.Equal(t, int(e.(*event).callTimes), 2)
}

func TestEventBus_Remove(t *testing.T) {
	events := New()
	defer events.Close()
	events.On("add", add)
	events.Remove("add")
	err := events.On("add", add)
	assert.NoError(t, err)
}

func TestPanic(t *testing.T) {
	events := New()
	defer events.Close()
	events.Once("panic", makePanic)
	err := events.Send("panic")
	assert.NoError(t, err)

	runtime.Gosched()
	// panic 之后也会注销once
	err = events.Send("panic")
	assert.EqualError(t, err, ErrNotFound.Error())
}
