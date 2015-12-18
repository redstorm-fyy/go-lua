
#include "_cgo_export.h"
#include "luago.h"

// macro wrappers
int luago_upvalueindex(int idx){
	return lua_upvalueindex(idx);
}

///////////////////////////////////////////////////////////////////////////

int luago_finishpcallk(lua_State* L,int status,lua_KContext ctx){
	return finishpcallk(L,status,ctx);
}

static int luago_gofunccall(lua_State* L){
	return gofunccall(L);
}

static int luago_gofuncgc(lua_State* L){
	return gofuncgc(L);
}

static int luago_interfacegc(lua_State* L){
	return interfacegc(L);
}

static int luago_interfaceindex(lua_State* L){
	return interfaceindex(L);
}

void luago_initstate(lua_State* L){
	luaL_newmetatable(L,"__gofunc");				// meta
	lua_pushcclosure(L,luago_gofuncgc,0);			// meta gc
	lua_setfield(L,-2,"__gc");						// meta (meta.__gc=gc)
	lua_pop(L,1);
	luaL_newmetatable(L,"__gointerface");			// meta
	lua_pushcclosure(L,luago_interfacegc,0);		// meta gc
	lua_setfield(L,-2,"__gc");						// meta (meta.__gc=gc)
	lua_pushcclosure(L,luago_interfaceindex,0); 	// meta indexfunc
	lua_setfield(L,-2,"__index");					// meta (meta.__index=indexfunc)
	lua_pop(L,1);
}

void luago_pushgoclosure(lua_State* L,int ref){
	int* udata=(int*)lua_newuserdata(L,sizeof(ref));// ref
	*udata=ref;
	luaL_setmetatable(L,"__gofunc");
	lua_pushcclosure(L,luago_gofunccall,1);// call
}

int luago_togoclosure(lua_State* L,int idx){
	int* udata=(int*)lua_touserdata(L,idx);
	if(udata){
		return *udata;
	}
	return -1;
}

void luago_pushinterface(lua_State* L,struct Interface ref){
	struct Interface* udata=(struct Interface*)lua_newuserdata(L,sizeof(ref));// ref
	*udata=ref;
	luaL_setmetatable(L,"__gointerface");
}

struct Interface* luago_tointerface(lua_State* L,int idx){
	struct Interface* udata=(struct Interface*)lua_touserdata(L,idx);
	if(udata){
		return udata;
	}
	return NULL;
}

void luago_getinterfacemetatable(lua_State* L){
	luaL_getmetatable(L,"__gointerface");
}
