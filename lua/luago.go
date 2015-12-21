package lua

/*
#include <stdlib.h>
#include "lua.h"
#include "lualib.h"
#include "lauxlib.h"
#include "luago.h"
*/
import "C"

import (
	"reflect"
	"strings"
	"unsafe"
)

const LUA_OK = int(C.LUA_OK)
const LUA_YIELD = int(C.LUA_YIELD)

type global struct {
	reg     []interface{}
	freeidx []uint
	tp      map[reflect.Type]uint
}

type State struct {
	s *C.lua_State
	g *global
}

func NewState() *State {
	L := C.luaL_newstate()
	s := &State{s: L}
	C.lua_pushlightuserdata(L, unsafe.Pointer(L))
	C.lua_pushlightuserdata(L, unsafe.Pointer(s))
	C.lua_settable(L, C.LUA_REGISTRYINDEX)
	C.luago_initstate(L)
	s.g = &global{tp: make(map[reflect.Type]uint)}
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
	for _, reft := range s.g.tp {
		s.unregister(reft)
	}
	s.g.tp = nil
	C.lua_close(s.s)
	s.s = nil
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

func (s *State) register(obj interface{}) uint {
	freelen := len(s.g.freeidx)
	if freelen == 0 {
		s.g.reg = append(s.g.reg, obj)
		return uint(len(s.g.reg)) - 1
	}
	idx := s.g.freeidx[freelen-1]
	s.g.reg[idx] = obj
	s.g.freeidx = s.g.freeidx[0 : freelen-1]
	return idx
}

func (s *State) unregister(idx uint) {
	if s.g.reg[idx] != nil {
		s.g.reg[idx] = nil
		s.g.freeidx = append(s.g.freeidx, idx)
	}
}

func (s *State) getreg(idx uint) interface{} {
	return s.g.reg[idx]
}

func (s *State) NewThread() *State {
	L := C.lua_newthread(s.s)
	ts := &State{s: L, g: s.g}
	// there is no thread metatable to release the registry,so we do not use registry to save thread state.
	// otherwise,we create a different thread state in State.ToThread.so the same thread means State.s is same not *State is same
	//C.lua_pushlightuserdata(L, unsafe.Pointer(L))
	//C.lua_pushlightuserdata(L, unsafe.Pointer(ts))
	//C.lua_settable(L, C.LUA_REGISTRYINDEX)
	return ts
}

func (s *State) PushThread() {
	C.lua_pushthread(s.s)
}

func (s *State) XMove(to *State, n int) {
	C.lua_xmove(s.s, to.s, C.int(n))
}

func (s *State) ToThread(idx int) *State {
	L := C.lua_tothread(s.s, C.int(idx))
	return &State{s: L, g: s.g}
	//return getstate(L)
}

func (s *State) Equal(ts *State) bool {
	return s.s == ts.s
}

func (s *State) Resume(nargs int) int {
	return int(C.lua_resume(s.s, nil, C.int(nargs)))
}

type GoKFunction func(*State, int) int

//export finishpcallk
func finishpcallk(L *C.lua_State, status C.int, ctx C.lua_KContext) int {
	s := getstate(L)
	kfunc := s.getreg(uint(ctx)).(GoKFunction)
	s.unregister(uint(ctx))
	return kfunc(s, int(status))
}

func (s *State) PCallk(nargs int, call GoKFunction) int {
	if call == nil {
		return int(C.lua_pcallk(s.s, C.int(nargs), C.LUA_MULTRET, 0, 0, nil))
	}
	ref := s.register(call)
	status := int(C.lua_pcallk(s.s, C.int(nargs), C.LUA_MULTRET, 0, C.lua_KContext(ref), C.lua_KFunction(C.luago_finishpcallk)))
	s.unregister(ref)
	return status
}

func (s *State) GetGlobalTable() {
	C.lua_rawgeti(s.s, C.LUA_REGISTRYINDEX, C.LUA_RIDX_GLOBALS)
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

func (s *State) SetI(idx int, k int64) {
	C.lua_seti(s.s, C.int(idx), C.lua_Integer(k))
}

func (s *State) GetI(idx int, k int64) {
	C.lua_geti(s.s, C.int(idx), C.lua_Integer(k))
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

func (s *State) PushValue(idx int) {
	C.lua_pushvalue(s.s, C.int(idx))
}

func (s *State) Insert(idx int) {
	C.lua_rotate(s.s, C.int(idx), 1)
}

func (s *State) Replace(idx int) {
	C.lua_copy(s.s, -1, C.int(idx))
	s.Pop(1)
}

func (s *State) ltype(idx int) int {
	return int(C.lua_type(s.s, C.int(idx)))
}

func (s *State) Typename(idx int) string {
	return C.GoString(C.lua_typename(s.s, C.lua_type(s.s, C.int(idx))))
}

type GoClosure func(s *State) int

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

func (s *State) typeref(t reflect.Type) uint {
	if reft, ok := s.g.tp[t]; ok {
		return reft
	}
	reft := s.register(t)
	s.g.tp[t] = reft
	return reft
}

//export interfacegc
func interfacegc(L *C.lua_State) int {
	ref := C.luago_tointerface(L, 1)
	if ref != nil {
		s := getstate(L)
		if ref.refgc >= 0 {
			v := s.getreg(uint(ref.refv))
			luagc := s.getreg(uint(ref.refgc)).(func(interface{}))
			luagc(v)
			s.unregister(uint(ref.refgc))
		}
		s.unregister(uint(ref.refv))
	}
	return 0
}

func (s *State) PushInterface(v interface{}, luagc func(interface{})) {
	var ref C.struct_Interface
	t := reflect.TypeOf(v)
	ref.reft = C.int(s.typeref(t))
	ref.refv = C.int(s.register(v))
	if luagc != nil {
		ref.refgc = C.int(s.register(luagc))
	} else {
		ref.refgc = -1
	}
	C.luago_pushinterface(s.s, ref)
}

func (s *State) tointerface(idx int) interface{} {
	ref := C.luago_tointerface(s.s, C.int(idx))
	if ref != nil {
		return s.getreg(uint(ref.refv))
	}
	return nil
}

func (s *State) pushstring(str string) {
	cs := C.CString(str)
	defer C.free(unsafe.Pointer(cs))
	C.lua_pushstring(s.s, cs)
}

func (s *State) tostring(idx int) string {
	var len C.size_t
	cs := C.lua_tolstring(s.s, C.int(idx), &len)
	return C.GoStringN(cs, C.int(len))
}

func (s *State) pushboolean(b bool) {
	if b {
		C.lua_pushboolean(s.s, 1)
	} else {
		C.lua_pushboolean(s.s, 0)
	}
}

func (s *State) toboolean(idx int) bool {
	b := C.lua_toboolean(s.s, C.int(idx))
	if b == 0 {
		return false
	} else {
		return true
	}
}

func (s *State) PushVariant(v interface{}) {
	switch v := v.(type) {
	case nil:
		C.lua_pushnil(s.s)
	case bool:
		s.pushboolean(v)
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
	default:
		s.PushInterface(v, nil)
	}
}

func (s *State) ToVariant(idx int) interface{} {
	switch s.ltype(idx) {
	case C.LUA_TNIL:
		return nil
	case C.LUA_TBOOLEAN:
		return s.toboolean(idx)
	case C.LUA_TNUMBER:
		n := C.lua_tonumberx(s.s, C.int(idx), nil)
		i := C.lua_tointegerx(s.s, C.int(idx), nil)
		if float64(n) == float64(i) {
			return int64(i)
		} else {
			return float64(n)
		}
	case C.LUA_TSTRING:
		return s.tostring(idx)
	case C.LUA_TTABLE:
		return nil
	case C.LUA_TFUNCTION:
		return s.togoclosure(idx)
	case C.LUA_TUSERDATA:
		return s.tointerface(idx)
	default:
		return nil
	}
}

// extend functions without lua stack manipulate

func (s *State) pushfield(fname []string) bool {
	if len(fname) == 0 {
		return false
	}
	s.GetGlobal(fname[0])
	if s.ltype(-1) == C.LUA_TNIL {
		s.Pop(1)
		return false
	}
	for i, n := 1, len(fname); i < n; i++ {
		if s.ltype(-1) != C.LUA_TTABLE {
			s.Pop(1)
			return false
		}
		s.GetField(-1, fname[i])
		s.Remove(-2)
	}
	return true
}

func (s *State) getrets(top int) []interface{} {
	var rets []interface{}
	for i, n := top+1, int(C.lua_gettop(s.s)); i <= n; i++ {
		rets = append(rets, s.ToVariant(i))
	}
	return rets
}

func (s *State) Call(fname string, args ...interface{}) ([]interface{}, bool) {
	top := C.lua_gettop(s.s)
	funcSplit := strings.Split(fname, ".")
	if !s.pushfield(funcSplit) {
		return nil, false
	}
	if s.ltype(-1) != C.LUA_TFUNCTION {
		s.Pop(1)
		return nil, false
	}
	for _, arg := range args {
		s.PushVariant(arg)
	}
	status := s.PCallk(len(args), nil)
	if status != C.LUA_OK && status != C.LUA_YIELD {
		C.lua_settop(s.s, top)
		return nil, false
	}
	rets := s.getrets(int(top))
	C.lua_settop(s.s, top)
	return rets, true
}

func (s *State) RegFunc(fname string, f GoClosure) {
	fnames := strings.Split(fname, ".")
	s.GetGlobalTable() // _G
	for _, fn := range fnames[0 : len(fnames)-1] {
		s.GetField(-1, fn) // table field
		if s.ltype(-1) != C.LUA_TTABLE {
			s.Remove(-1)                 // table
			C.lua_createtable(s.s, 0, 0) // table newfield
			C.lua_pushvalue(s.s, -1)     // table newfield newfield
			s.SetField(-3, fn)           // table newfield (table[fn]=newfield)
		}
		s.Remove(-2) // field
	}
	s.pushgoclosure(f) // table f
	fn := fnames[len(fnames)-1]
	s.SetField(-2, fn) // table (table[fn]=f)
	s.Pop(1)
}

//export interfaceindex
func interfaceindex(L *C.lua_State) int {
	ref := C.luago_tointerface(L, 1)
	if ref != nil {
		C.lua_getmetatable(L, 1)                   // inter key meta
		C.lua_geti(L, -1, C.lua_Integer(ref.reft)) // inter key meta meta[tp]
		C.lua_pushvalue(L, 2)                      // inter key meta meta[tp] key
		C.lua_gettable(L, -2)                      // inter key meta meta[tp] meta[tp][key]
		return 1
	}
	return 0
}

func (s *State) RegMethod(v interface{}, name string, f GoClosure) {
	t := reflect.TypeOf(v)
	reft := s.typeref(t)
	C.luago_getinterfacemetatable(s.s)       // meta
	C.lua_geti(s.s, -1, C.lua_Integer(reft)) // meta meta[tp]
	if s.ltype(-1) == C.LUA_TNIL {
		s.Pop(1)                                 // meta
		C.lua_createtable(s.s, 0, 0)             // meta table
		C.lua_pushvalue(s.s, -1)                 // meta table table
		C.lua_seti(s.s, -3, C.lua_Integer(reft)) // meta table (meta[tp]=table)
	}
	s.pushgoclosure(f)   // meta meta[tp] f
	s.SetField(-2, name) // meta meta[tp] (meta[tp][name]=f)
	s.Pop(2)
}

func (s *State) ForEach(idx int, call func() bool) {
	if idx < 0 && idx > C.LUA_REGISTRYINDEX {
		top := s.GetTop()
		idx = top + idx + 1
	}
	for C.lua_pushnil(s.s); C.lua_next(s.s, C.int(idx)) != 0; {
		if call() {
			s.Pop(2)
			return
		}
		s.Pop(1)
	}
}

// used for simplify the traverse over deep tables

type Reference struct {
	s     *State
	lref  int
	child map[interface{}]*Reference
}

func (s *State) NewReference() *Reference {
	s.GetGlobalTable()
	lref := C.luaL_ref(s.s, C.LUA_REGISTRYINDEX)
	return &Reference{s: s, lref: int(lref), child: make(map[interface{}]*Reference)}
}

func (r *Reference) Release() {
	if r.lref != C.LUA_NOREF {
		r.lref = C.LUA_NOREF
		s := r.s
		for _, v := range r.child {
			v.Release()
		}
		r.child = nil
		C.luaL_unref(s.s, C.LUA_REGISTRYINDEX, C.int(r.lref))
	}
}

func (r *Reference) Sub(k interface{}) *Reference {
	if r == nil {
		return nil
	}
	s := r.s
	s.GetI(C.LUA_REGISTRYINDEX, int64(r.lref)) // r
	if s.ltype(-1) != C.LUA_TTABLE {
		s.Pop(1)
		return nil
	}
	if child, ok := r.child[k]; ok {
		s.Pop(1)
		return child
	}
	s.PushVariant(k) // r k
	s.GetTable(-2)   // r v
	if s.ltype(-1) == C.LUA_TNIL {
		s.Pop(2)
		return nil
	}
	lref := C.luaL_ref(s.s, C.LUA_REGISTRYINDEX) // r
	s.Pop(1)
	child := &Reference{s: s, lref: int(lref), child: make(map[interface{}]*Reference)}
	r.child[k] = child
	return child
}

func (r *Reference) Get() interface{} {
	if r == nil {
		return nil
	}
	s := r.s
	s.GetI(C.LUA_REGISTRYINDEX, int64(r.lref)) // r
	v := s.ToVariant(-1)
	s.Pop(1)
	return v
}

func (r *Reference) SetFd(k interface{}, v interface{}) {
	if r == nil {
		return
	}
	s := r.s
	s.GetI(C.LUA_REGISTRYINDEX, int64(r.lref)) // r
	if s.ltype(-1) != C.LUA_TTABLE {
		s.Pop(1)
		return
	}
	if child, ok := r.child[k]; ok {
		child.Release()
		delete(r.child, k)
	}
	s.PushVariant(k) // r k
	s.PushVariant(v) // r k v
	s.SetTable(-3)   // r
	s.Pop(1)
}

func (s *State) newref(idx int) *Reference {
	C.lua_pushvalue(s.s, C.int(idx))
	if s.ltype(-1) == C.LUA_TNIL {
		s.Pop(1)
		return nil
	}
	lref := C.luaL_ref(s.s, C.LUA_REGISTRYINDEX)
	return &Reference{s: s, lref: int(lref), child: make(map[interface{}]*Reference)}
}

func (r *Reference) ForEach(call func(k interface{}, v *Reference) bool) {
	if r == nil {
		return
	}
	s := r.s
	s.GetI(C.LUA_REGISTRYINDEX, int64(r.lref)) // r
	if s.ltype(-1) != C.LUA_TTABLE {
		s.Pop(1)
		return
	}
	s.ForEach(-1, func() bool {
		k := s.ToVariant(-2)
		v := s.newref(-1)
		defer v.Release()
		return call(k, v)
	})
	s.Pop(1)
}
