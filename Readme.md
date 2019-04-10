# eventBus 事件订阅器

## A Emitter 订阅器接口
- On/Once 绑定事件，绑定成功返回true，重复绑定返回false
- Once 执行一次后自动清除，On 可以一直执行
- Send 调用具体绑定方法实例
- remove 移除事件

```go
type Emitter interface {
    On(eventKey string, call interface{}) error
    Once(eventKey string, call interface{}) error
    Send(eventKey string, args ...interface{}) error
    Remove(eventKey string)
}
```

## example
```go
events := New()
events.On("add", func(a, b int) {
    fmt.Printf("receive event add, args are: a %d, b %d = %d\n", a, b, a+b)
})
events.Send("add", 1, 2)
events.Send("add", 1, 2)
events.Once("once", func() {
    fmt.Printf("receive event once\n")
})
events.Send("once")
events.Send("once")

// Output:
// receive event add, args are: a 1, b 2 = 3
// receive event add, args are: a 1, b 2 = 3
// receive event once
```
