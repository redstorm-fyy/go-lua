package lua

/*
#include <stdlib.h>
#include "lua.h"
#include "lualib.h"
#include "lauxlib.h"

int finishpcallk(lua_State* L,int status,lua_KContext ctx);

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
	return s
}

func getState(L *C.lua_State) *State {
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

func (s *State) regkfunc(call func(*State, int)) int {

}

func (s *State) PCall(nargs int) int {
	return int(C.lua_pcallk(s.s, C.int(nargs), C.LUA_MULTRET, 0, 0, nil))
}

//export finishPCallk
func finishPCallk(L *C.lua_State, status C.int, ctx C.lua_KContext) int {
	s := getState(L)
	kfunc := s.getkfunc(ctx)
	s.unregkfunc(ctx)
	return kfunc(s, int(status))
}

func (s *State) PCallk(nargs int, call func(*State, int)) int {
	idx := s.regkfunc(call)
	status := int(C.lua_pcallk(s.s, C.int(nargs), C.LUA_MULTRET, 0, idx, C.lua_KFunction(C.finishpcallk)))
	s.unregkfunc(idx)
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

func (s *State) PushValue(v interface{}) {

}

func (s *State) ToValue(idx int) interface{} {
	return nil
}

func (s *State) Call(fname string, args ...interface{}) ([]interface{}, error) {
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
		err := errors.New(s.typename(-1) + strings.Join(funcSplit, "."))
		s.Pop(1)
		return nil, err
	}
	for _, arg := range args {
		s.PushValue(arg)
	}
	status := C.lua_pcallk(s.s, C.int(len(args)), C.LUA_MULTRET, 0, 0, nil)
	var rets []interface{}
	for i, n := int(top)+1, int(C.lua_gettop(s.s)); i <= n; i++ {
		rets = append(rets, s.ToValue(i))
	}
	C.lua_settop(s.s, top)
	return rets, nil
}
