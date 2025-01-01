/*
* 将结构体注册为服务
 */
package geerpc

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

// 服务端注册的结构体方法
type methodType struct {
	method    reflect.Method // 方法本身 eg: func(*geerpc.S, int, string) error
	ArgType   reflect.Type   // 参数类型 eg: int, string
	ReplyType reflect.Type   // 响应类型 eg: error
	numCalls  uint64
}

func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

// 创建参数实例
func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	// 判断参数类型是否是指针类型还是值类型
	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}

// 创建响应实例
func (m *methodType) newReplyv() reflect.Value {
	// reply必须是指针类型
	replyv := reflect.New(m.ReplyType.Elem())
	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

// 服务端注册的结构体
type service struct {
	name    string                 // 结构体名称
	typ     reflect.Type           // 结构体类型 eg: *geerpc.S
	rcvr    reflect.Value          // 结构体本身 eg: &geerpc.S{}
	methods map[string]*methodType // 存储结构体所有方法
}

// 将结构体注册为服务
func newService(rcvr any) *service {
	s := new(service) // 使用 new() 创建 service 对象
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name() // Indirect保证无论传递的是结构体指针还是结构体，都能获取到结构体名称
	s.typ = reflect.TypeOf(rcvr)
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}

	s.registerMethods()
	return s
}

// 注册结构体的所有可导出方法
func (s *service) registerMethods() {
	s.methods = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i) // 第 i 个方法
		mType := method.Type      // func(*geerpc.S, int, string) error
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}

		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuiltinType(argType) && !isExportedOrBuiltinType(replyType) {
			continue
		}
		// 可导出的方法添加到map中
		s.methods[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc server: register %s.%s\n", s.name, method.Name)
	}
}

// 判断参数和响应是否为导出类型或内置类型
func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

// 调用方法
func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	f := m.method.Func
	// 调用实际方法
	returnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv})
	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}
