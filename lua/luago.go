package lua

/*
#include <stdlib.h>
#include "lua.h"
#include "lualib.h"
#include "lauxlib.h"

int luago_finishpcallk(lua_State* L,int status,lua_KContext ctx);
void luago_initstate(lua_State* L);
void luago_pushgoclosure(lua_State* L,int ref);
int luago_togoclosure(lua_State* L,int idx);
int luago_upvalueindex(int idx);

*/
import "C"

import (
	"errors"
	"strings"
	"unsafe"
)

type State struct {
	s       *C.lua_State
	reg     []interface{}
	freeidx []uint
}

func NewState() *State {
	L := C.luaL_newstate()
	s := &State{s: L}
	C.lua_pushlightuserdata(L, unsafe.Pointer(L))
	C.lua_pushlightuserdata(L, unsafe.Pointer(s))
	C.lua_settable(L, C.LUA_REGISTRYINDEX)
	C.luago_initstate(L)
	return s
}

func getstate(L *C.lua_State) *State {
	C.lua_pushlightuserdata(L, unsafe.Pointer(L))
	C.lua_gettable(L, C.LUA_REGISTRYINDEX)
	s := C.lua_touserdata(L, -1)
	C.lua_settop(L, -2)
	return (*State)(s)
}

func (s *State) Close() {
	C.lua_close(s.s)
}

func (s *State) OpenLibs() {
	C.luaL_openlibs(s.s)
}

func (s *State) LoadFile(fname string) int {
	cname := C.CString(fname)
	defer C.free(unsafe.Pointer(cname))
	return int(C.luaL_loadfilex(s.s, cname, nil))
}

func (s *State) LoadString(str string) int {
	cs := C.CString(str)
	defer C.free(unsafe.Pointer(cs))
	return int(C.luaL_loadstring(s.s, cs))
}

func (s *State) LoadBuffer(buf []byte) int {
	return int(C.luaL_loadbufferx(s.s, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), nil, nil))
}

type GoClosure func(s *State) int

func (s *State) register(obj interface{}) uint {
	freelen := len(s.freeidx)
	if freelen == 0 {
		s.reg = append(s.reg, obj)
		return uint(len(s.reg)) - 1
	}
	idx := s.freeidx[freelen-1]
	s.reg[idx] = obj
	s.freeidx = s.freeidx[0 : freelen-1]
	return idx
}

func (s *State) unregister(idx uint) {
	if s.reg[idx] != nil {
		s.reg[idx] = nil
		s.freeidx = append(s.freeidx, idx)
	}
}

func (s *State) getreg(idx uint) interface{} {
	return s.reg[idx]
}

func (s *State) PCall(nargs int) int {
	return int(C.lua_pcallk(s.s, C.int(nargs), C.LUA_MULTRET, 0, 0, nil))
}

//export finishpcallk
func finishpcallk(L *C.lua_State, status C.int, ctx C.lua_KContext) int {
	s := getstate(L)
	kfunc := s.getreg(uint(ctx)).(func(*State, int) int)
	s.unregister(uint(ctx))
	return kfunc(s, int(status))
}

func (s *State) PCallk(nargs int, call func(*State, int) int) int {
	ref := s.register(call)
	status := int(C.lua_pcallk(s.s, C.int(nargs), C.LUA_MULTRET, 0, C.lua_KContext(ref), C.lua_KFunction(C.luago_finishpcallk)))
	s.unregister(ref)
	return status
}

func (s *State) GetGlobal(name string) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	C.lua_getglobal(s.s, cname)
}

func (s *State) SetGlobal(name string) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	C.lua_setglobal(s.s, cname)
}

func (s *State) GetField(idx int, k string) {
	ck := C.CString(k)
	defer C.free(unsafe.Pointer(ck))
	C.lua_getfield(s.s, C.int(idx), ck)
}

func (s *State) SetField(idx int, k string) {
	ck := C.CString(k)
	defer C.free(unsafe.Pointer(ck))
	C.lua_setfield(s.s, C.int(idx), ck)
}

func (s *State) GetTable(idx int) {
	C.lua_gettable(s.s, C.int(idx))
}

func (s *State) SetTable(idx int) {
	C.lua_settable(s.s, C.int(idx))
}

func (s *State) Pop(n int) {
	C.lua_settop(s.s, C.int(-n-1))
}

func (s *State) GetTop() int {
	return int(C.lua_gettop(s.s))
}

func (s *State) SetTop(idx int) {
	C.lua_settop(s.s, C.int(idx))
}

func (s *State) Remove(idx int) {
	C.lua_rotate(s.s, C.int(idx), C.int(-1))
	s.Pop(1)
}

func (s *State) ltype(idx int) int {
	return int(C.lua_type(s.s, C.int(idx)))
}

func (s *State) typename(idx int) string {
	return C.GoString(C.lua_typename(s.s, C.lua_type(s.s, C.int(idx))))
}

//export gofunccall
func gofunccall(L *C.lua_State) int {
	ref := C.luago_togoclosure(L, C.luago_upvalueindex(1))
	if ref >= 0 {
		s := getstate(L)
		goclosure := s.getreg(uint(ref)).(GoClosure)
		return goclosure(s)
	}
	return 0
}

//export gofuncgc
func gofuncgc(L *C.lua_State) int {
	ref := C.luago_togoclosure(L, 1)
	if ref >= 0 {
		s := getstate(L)
		s.unregister(uint(ref))
	}
	return 0
}

func (s *State) pushgoclosure(v GoClosure) {
	ref := s.register(v)
	C.luago_pushgoclosure(s.s, C.int(ref))
}

func (s *State) togoclosure(idx int) GoClosure {
	if C.lua_getupvalue(s.s, C.int(idx), 1) != nil {
		ref := C.luago_togoclosure(s.s, -1)
		s.Pop(1)
		if ref >= 0 {
			return s.getreg(uint(ref)).(GoClosure)
		}
	}
	return nil
}

func (s *State) pushstring(str string) {
	cs := C.CString(str)
	defer C.free(unsafe.Pointer(cs))
	C.lua_pushstring(s.s, cs)
}

func (s *State) PushValue(v interface{}) {
	switch v := v.(type) {
	default:
		C.lua_pushnil(s.s)
	case nil:
		C.lua_pushnil(s.s)
	case bool:
		if v {
			C.lua_pushboolean(s.s, 1)
		} else {
			C.lua_pushboolean(s.s, 0)
		}
	case string:
		s.pushstring(v)
	case []byte:
		C.lua_pushlstring(s.s, (*C.char)(unsafe.Pointer(&v[0])), C.size_t(len(v)))
	case float32:
		C.lua_pushnumber(s.s, C.lua_Number(v))
	case float64:
		C.lua_pushnumber(s.s, C.lua_Number(v))
	case int:
		C.lua_pushinteger(s.s, C.lua_Integer(v))
	case uint:
		C.lua_pushinteger(s.s, C.lua_Integer(v))
	case int32:
		C.lua_pushinteger(s.s, C.lua_Integer(v))
	case uint32:
		C.lua_pushinteger(s.s, C.lua_Integer(v))
	case int64:
		C.lua_pushinteger(s.s, C.lua_Integer(v))
	case uint64:
		C.lua_pushinteger(s.s, C.lua_Integer(v))
	case GoClosure:
		s.pushgoclosure(v)
	}
}

func (s *State) ToValue(idx int) interface{} {
	switch s.ltype(idx) {
	default:
		return nil
	case C.LUA_TNIL:
		return nil
	case C.LUA_TBOOLEAN:
		b := C.lua_toboolean(s.s, C.int(idx))
		if b == 0 {
			return false
		} else {
			return true
		}
	case C.LUA_TNUMBER:
		n := C.lua_tonumberx(s.s, C.int(idx), nil)
		i := C.lua_tointegerx(s.s, C.int(idx), nil)
		if float64(n) == float64(i) {
			return int64(i)
		} else {
			return float64(n)
		}
	case C.LUA_TSTRING:
		var len C.size_t
		cs := C.lua_tolstring(s.s, C.int(idx), &len)
		return C.GoStringN(cs, C.int(len))
	case C.LUA_TFUNCTION:
		return s.togoclosure(idx)
	}
}

func (s *State) Call(fname string, args []interface{}, call func(*State, []interface{}, error)) ([]interface{}, error) {
	top := C.lua_gettop(s.s)
	funcSplit := strings.Split(fname, ".")
	s.GetGlobal(funcSplit[0])
	if s.ltype(-1) == C.LUA_TNIL {
		s.Pop(1)
		return nil, errors.New("nil " + funcSplit[0])
	}
	for i, n := 1, len(funcSplit); i < n; i++ {
		if s.ltype(-1) != C.LUA_TTABLE {
			s.Pop(1)
			return nil, errors.New("nil " + strings.Join(funcSplit[0:i+1], "."))
		}
		s.GetField(-1, funcSplit[i])
		s.Remove(-2)
	}
	if s.ltype(-1) != C.LUA_TFUNCTION {
		err := errors.New(s.typename(-1) + fname)
		s.Pop(1)
		return nil, err
	}
	for _, arg := range args {
		s.PushValue(arg)
	}
	setrets := func(s *State, oldtop int) []interface{} {
		var rets []interface{}
		for i, n := oldtop+1, int(C.lua_gettop(s.s)); i <= n; i++ {
			rets = append(rets, s.ToValue(i))
		}
		return rets
	}
	var status int
	if call != nil {
		aftercall := func(s *State, status int) int {
			if status != C.LUA_OK && status != C.LUA_YIELD {
				C.lua_settop(s.s, top)
				call(s, nil, errors.New("yield call "+fname))
			} else {
				rets := setrets(s, int(top))
				C.lua_settop(s.s, top)
				call(s, rets, nil)
			}
			return 0
		}
		status = s.PCallk(len(args), aftercall)
	} else {
		status = s.PCall(len(args))
	}
	if status != C.LUA_OK && status != C.LUA_YIELD {
		C.lua_settop(s.s, top)
		return nil, errors.New("call " + fname)
	} else {
		rets := setrets(s, int(top))
		C.lua_settop(s.s, top)
		return rets, nil
	}
}
