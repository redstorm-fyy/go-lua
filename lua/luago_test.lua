
print("lua begin")

test={}
function test.CallMe(...)
	print("lua CallMe ",...)
	return true,...
end

function test.Register(...)
	return gofunc.test.luaFunc(...)
end

function test.Method(t,...)
	--t:noexist()
	return t:mymethod(...)
end

RefTable={
	{["b"]=1,3,"abc",{[5]=44,["haha"]="a"}},
	["kk"]={},
}
