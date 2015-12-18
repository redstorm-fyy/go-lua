#ifndef _LUAGO_H_
#define _LUAGO_H_


int luago_upvalueindex(int idx);

int luago_finishpcallk(lua_State* L,int status,lua_KContext ctx);
void luago_initstate(lua_State* L);
void luago_pushgoclosure(lua_State* L, int ref);
int luago_togoclosure(lua_State* L,int idx);
struct Interface{
	int reft;
	int refv;
	int refgc;
};
void luago_pushinterface(lua_State* L,struct Interface ref);
struct Interface* luago_tointerface(lua_State* L,int idx);
void luago_getinterfacemetatable(lua_State* L);

#endif//_LUAGO_H_
