package main

import (
	"github.com/yuin/gopher-lua"
)

func LUA_P(L *lua.LState, fun string, nret int) (p lua.P) {
	p.Fn = L.GetGlobal(fun)
	p.NRet = nret
	p.Protect = true
	return p
}
