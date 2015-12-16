
#include "_cgo_export.h"

int luago_finishpcallk(lua_State* L,int status,lua_KContext ctx){
	return finishpcallk(L,status,ctx);
}

static int luago_gofunccall(lua_State* L){
	return gofunccall(L);
}

static int luago_gofuncgc(lua_State* L){
	return gofuncgc(L);
}

void luago_initstate(lua_State* L){
	luaL_newmetatable(L,"__gofunc");		// meta
	lua_pushcclosure(L,luago_gofuncgc,0);	// meta gc
	lua_setfield(L,-2,"__gc");				// (meta.__gc=gc) meta
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

int luago_upvalueindex(int idx){
	return lua_upvalueindex(idx);
}
