package lua

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

var s *State

func TestMain(m *testing.M) {
	flag.Parse()
	fmt.Println("test begin")
	s = NewState()
	s.OpenLibs()
	c := m.Run()
	fmt.Println("top=", s.GetTop())
	fmt.Println("regnum=", len(s.g.reg), "freeidx=", len(s.g.freeidx), "tplen=", len(s.g.tp))
	s.GC()
	for k, reft := range s.g.tp {
		s.unregister(reft)
		delete(s.g.tp, k)
	}
	fmt.Println("regnum=", len(s.g.reg), "freeidx=", len(s.g.freeidx), "tplen=", len(s.g.tp))
	s.Close()
	fmt.Println("regnum=", len(s.g.reg), "freeidx=", len(s.g.freeidx), "tplen=", len(s.g.tp))
	fmt.Println("test end")
	os.Exit(c)
}

func TestCallMe(t *testing.T) {
	fmt.Println("begin top=", s.GetTop())
	fmt.Println("LoadFile ", s.LoadFile("luago_test.lua"))
	fmt.Println("PCallk ", s.PCallk(0, nil))
	ret, ok := s.Call("test.CallMe", 23, "haha")
	fmt.Println("Call ", ret, ok)
	ret, ok = s.Call("NotExist", 3)
	fmt.Println("Call ", ret, ok)
	fmt.Println("end top=", s.GetTop())
}

type luaStruct struct {
	x   int
	str string
}

func luaFunc(s *State) int {
	switch t := s.ToVariant(1).(type) {
	case *luaStruct:
		data, ok := s.ToVariant(2).(string)
		if !ok {
			fmt.Println("error luaFunc 2")
			return 0
		}
		fmt.Println("luaFunc ok ")
		s.PushVariant(data)
		s.PushVariant(t)
		t.x = 11
		t.str = "after"
	case luaStruct:
		data, ok := s.ToVariant(2).(string)
		if !ok {
			fmt.Println("error luaFunc 2")
			return 0
		}
		fmt.Println("luaFunc ok ")
		s.PushVariant(data)
		t.x = 11
		t.str = "after"
		s.PushVariant(t)
	default:
		fmt.Println("error luaFunc 1")
		return 0
	}
	return 2
}

func luaMethod(s *State) int {
	t, ok := s.ToVariant(1).(*luaStruct)
	if !ok {
		fmt.Println("error luaMethod 1")
		return 0
	}
	arg, ok := s.ToVariant(2).(string)
	if !ok {
		fmt.Println("error luaMethod 2")
		return 0
	}
	fmt.Println("t=,arg=", t, arg)
	t.x = 100
	t.str = "haha"
	return s.GetTop()
}

func testReg1() {
	fmt.Println("top1=", s.GetTop())
	v := &luaStruct{x: 5, str: "abc"}
	fmt.Println("v=", v)
	ret, ok := s.Call("test.Register", v, "arg2")
	fmt.Println("top2=", s.GetTop(), ret, ok)
	fmt.Println("ret=", ret[1].(*luaStruct))
	fmt.Println("v=", v)
}
func testReg2() {
	fmt.Println("top3=", s.GetTop())
	v := luaStruct{x: 5, str: "abc"}
	fmt.Println("v=", v)
	ret, ok := s.Call("test.Register", v, "arg2")
	fmt.Println("top4=", s.GetTop(), ret, ok)
	fmt.Println("ret=", ret[1].(luaStruct))
	fmt.Println("v=", v)
}
func TestRegister(t *testing.T) {
	fmt.Println("begin top=", s.GetTop())
	s.RegFunc("gofunc.test.luaFunc", luaFunc)
	testReg1()
	testReg2()
	s.RegMethod((*luaStruct)(nil), "mymethod", luaMethod)
	v := &luaStruct{x: 2, str: "dd"}
	ret, ok := s.Call("test.Method", v, "meth")
	fmt.Println("ret=", ret[0].(*luaStruct))
	fmt.Println(ret, ok, v)
	r := s.NewReference()
	defer r.Release()
	r.Sub("gofunc").Sub("test").SetFd("luaFunc", nil)
	fmt.Println("end top=", s.GetTop())
}

func TestReference(t *testing.T) {
	fmt.Println("top1=", s.GetTop())
	r := s.NewReference()
	defer func() {
		r.Release()
		fmt.Println("top3=", s.GetTop())
	}()
	subr := r.Sub("RefTable")
	v := subr.Sub(1).Sub(3).Sub("haha").Get().(string)
	fmt.Println("v=", v)
	subr.Sub(1).SetFd("b", "bvalue")
	subr.Sub(1).SetFd("c", "cvalue")
	subr.ForEach(func(k interface{}, v *Reference) bool {
		fmt.Println("tra top=", s.GetTop(), "k=", k, "v=", v.Get())
		v.ForEach(func(k interface{}, v *Reference) bool {
			fmt.Println("sub top=", s.GetTop(), "k=", k, "v=", v.Get())
			return false
		})
		return false
	})
	fmt.Println("top2=", s.GetTop())
}

func TestYield(t *testing.T) {
	// yield and anything run setjmp/longjmp on golang1.5.2 will crash can cannot be tested yet
}
