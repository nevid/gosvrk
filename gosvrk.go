package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	//"reflect"
	//"io/ioutil"
	"os"
	//"strconv"
	"html"
	"net/http"
	"strings"
	"sync"
	"time"

	//"runtime"

	"github.com/yuin/gopher-lua"

	"encoding/json"
)

//var LuaSt *lua.LState

var LogPkStat int
var MaxMsgFileLen int
var PkDmpFlSz int

func init() {
	fmt.Println("GO forever! pkg: main")
}

func Lua_GetMsg(LuaSt *lua.LState, n int) (kind string, iid uint32, msg string) {
	LuaSt.CallByParam(LUA_P(LuaSt, "GetMsg", 3), lua.LNumber(n))

	//fmt.Println("getmsg", t1, t2, t3)
	kind = LuaSt.ToString(-3)
	iid = uint32(LuaSt.ToInt(-2))
	msg = LuaSt.ToString(-1)

	if kind == "" {
		return "", 0, ""
	}

	//LuaSt.Pop(3)    //
	LuaSt.SetTop(0) //на всякий случай

	//lua.LVAsNumber()

	//lua.Lv
	return kind, iid, msg
}

type sv3w_stat struct {
	allcnt int64
	df     int64
	df50   int64
	df200  int64
}

type sv3w_pack struct {
	uh    sv3w_UNOHead
	dry   sv3w_UNODryRaw
	md    sv3w_ModsDiag
	crhd  sv3w_CrateHead
	sdiag sv3w_CrateSDiag

	pk  *bytes.Buffer
	bdt *bytes.Buffer

	nmstr string
	szstr uint32

	racknum uint8
	busnum  uint8

	iocon *sv3w_io
	//syn1  *sync.Mutex

	ptmpack time.Time
	ctmpack time.Time
	st1fl   *os.File
	st2fl   *os.File //unotime
	stflag  int      //1-отменяет запись файлов статист (not use now)
	pkdmpfl *os.File

	packnum int64

	pkstat       sv3w_stat
	stmsg_tm     *time.Timer //timer for stat msg
	stmsg_tm_int time.Duration

	extcrstat ExtCrStats
	rkdrycont RkDryCont
}

func (t *sv3w_pack) Init() {
	t.stmsg_tm_int = time.Second * 30
	t.stmsg_tm = time.NewTimer(t.stmsg_tm_int)

	t.stflag = 1

}

func (t *sv3w_pack) OpPkRd(bf *[]byte) {
	t.pk = bytes.NewBuffer(*bf)
}

func (t *sv3w_pack) ReadStruct() int {
	bnm := make([]byte, 16)
	_, err := t.pk.Read(bnm)
	if err != nil {
		return 1
	}
	bsz := make([]byte, 4)
	_, err = t.pk.Read(bsz)
	if err != nil {
		return 2
	}

	t.nmstr = string(bnm)
	//t.nmstr = strings.TrimRight(t.nmstr, "\x00")
	s := strings.Split(t.nmstr, "\x00")
	t.nmstr = s[0]

	t.szstr = binary.LittleEndian.Uint32(bsz)

	//read data struct
	b := make([]byte, t.szstr)
	_, err = t.pk.Read(b)
	if err != nil {
		return 3
	}
	t.bdt = bytes.NewBuffer(b)

	return 0
}

//call this!!! bf- full packet
func (t *sv3w_pack) ParsePk(bf *[]byte) {
	t.OpPkRd(bf)

	if PkDmpFlSz > 0 {
		l := int32(len(*bf))
		binary.Write(t.pkdmpfl, binary.LittleEndian, l)
		t.pkdmpfl.Write(*bf)
	}

	for {
		r := t.ReadStruct() //after t.bdt = buffer of struct
		if r != 0 {
			break
		} else {
			//fmt.Println("nmstr", t.nmstr)
			if t.nmstr == "UNOHead" {
				t.packnum++
				t.ctmpack = time.Now()
				var puh sv3w_UNOHead
				puh.Basetime = t.uh.Basetime
				puh.Unotime = t.uh.Unotime
				t.uh.Rd(t.bdt)

				//fmt.Println("rk,bus,min,sec", t.uh.Racknum, t.uh.Busnum, t.uh.Unotime.Minute, t.uh.Unotime.Second)

				t.iocon.syn1.Lock()
				t.iocon.luaSt.CallByParam(LUA_P(t.iocon.luaSt, "Add_UNOHead", 0))
				t.iocon.syn1.Unlock()

				t.PackStat(&puh)

				t.ptmpack = t.ctmpack

			}

			if t.nmstr == "UNODryRaw" {
				for {
					err := t.dry.Rd(t.bdt)
					if err != nil {
						break
					}
					if int(t.dry.Pos) < len(t.rkdrycont.RawIpa) {
						t.rkdrycont.RawIpa[t.dry.Pos] = t.dry.Databyte
					}
				}
				//var bdry []byte
				bdry := UnpackDry(t.rkdrycont.RawIpa)
				//fmt.Println(bdry)

				t.rkdrycont.Nm = make(map[string]uint8)
				t.iocon.syn1.Lock()
				//t.Lua_Add_CrateHead(&t.crhd)...
				for n := 0; n < len(bdry); n++ {
					t.iocon.luaSt.CallByParam(LUA_P(t.iocon.luaSt, "Ipa_GetDigDataNm", 1),
						lua.LNumber(n))

					nm, _ := t.iocon.luaSt.Get(-1).(lua.LString)
					t.iocon.luaSt.Pop(1)
					if nm != "" {
						//fmt.Println("dryyy", nm, bdry[n])
						t.rkdrycont.Nm[nm.String()] = bdry[n]

					}
				}

				t.iocon.syn1.Unlock()

				t.EndOfUNO()
			}

			if t.nmstr == "ModsDiag" {
				for {
					err := t.md.Rd(t.bdt)
					if err != nil {
						break
					}

					//fmt.Println("RD", "type", t.md.Dg.TipM)

					if t.md.Dg.TipM != 0 {

						t.iocon.syn1.Lock()
						/*
							LuaSt.CallByParam(LUA_P(LuaSt, "Add_ModDiag", 0),
								lua.LNumber(t.md.Inputid.Id), lua.LNumber(t.md.Dg.Test_lamp),
								lua.LNumber(t.md.Dg.Error_lamp), lua.LNumber(t.md.Dg.Module_good))
						*/
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "_start_", 0)
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "type", int(t.md.Dg.TipM))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "testl", int(t.md.Dg.Test_lamp))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "errl", int(t.md.Dg.Error_lamp))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "modgood", int(t.md.Dg.Module_good))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "wrkl", int(t.md.Dg.Work_lamp))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "cable", int(t.md.Dg.Cable))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "cntrl_1", int(t.md.Dg.Control_1))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "cntrl_2", int(t.md.Dg.Control_2))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "mannum", int(t.md.Dg.ManNum))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "modready", int(t.md.Dg.Module_ready))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "mvcmask", int(t.md.Dg.Mvcmask))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "watchdog", int(t.md.Dg.Watchdog))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "self_watchdog", int(t.md.Dg.Self_watchdog))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "adjmod_good", int(t.md.Dg.Adjmodule_good))
						t.Lua_Add_ModDiag2(t.md.Inputid.Id, "adj_watcdog", int(t.md.Dg.Adj_watchdog))

						//t.Lua_Add_ModDiag2(t.md.Inputid.Id, "timeoff", int(t.md.Timeoff))

						//DEBUG for change, must be crate
						//t.Lua_Add_ModDiag2(t.md.Inputid.Id, "testl", int(1))

						t.iocon.syn1.Unlock()

					}

					//fmt.Println("modpos=", t.md.Modpos)

				}
			}
			if t.nmstr == "CrateHead" {
				for {
					err := t.crhd.Rd(t.bdt)
					if err != nil {
						break
					}

					if (t.crhd.Busnum != t.busnum) || (t.crhd.Racknum != t.racknum) {
						fmt.Printf("channel confirm error  %s: recv: rk=%d, bus=%d\n", t.iocon.ipport,
							t.crhd.Racknum, t.crhd.Busnum)
						break
					}

					t.iocon.syn1.Lock()
					t.Lua_Add_CrateHead(&t.crhd)
					t.iocon.syn1.Unlock()
					//fmt.Printf("crate=%d\n", t.crhd.Cratenum)

					//ведение своей статистики
					/*
						var iid sv3w_input_id_el
						iid.Bus = t.crhd.Busnum
						iid.Chnum = 0
						iid.Modnum = 0
						iid.Crate = t.crhd.Cratenum
						iid.Rack = t.crhd.Racknum
						id := iid.ToInt()
					*/
					//cs, f := crstats[id.Id]
					//if f == false {
					//	cs = *new(CrStats)
					//}
					//cs.Crhd = t.crhd
					//cs.Cnt++
					//crstats[id.Id] = cs

					//!!!pubstats.PutCr(t.crhd.Racknum, t.crhd.Cratenum, &t.crhd)

					//el := cs.Crstats[t.crhd.Cratenum]
					//el.Cnt++
					//cs[t.crhd.Cratenum].Cnt++
				}
			}
			if t.nmstr == "CrateSDiag" {
				for {
					err := t.sdiag.Rd(t.bdt)
					if err != nil {
						break
					}

					nm := string(t.sdiag.Name[:])
					nm = strings.TrimRight(nm, "\x00")

					//fmt.Println(nm)

					if nm == "ATMPOVER" {
						t.extcrstat.Atmpover = binary.LittleEndian.Uint32(t.sdiag.Val4[:])
						//fmt.Println("1")
					}
					if nm == "MUPPOVER" {
						t.extcrstat.Muppover[0] = uint8(t.sdiag.Val4[0])
						t.extcrstat.Muppover[1] = uint8(t.sdiag.Val4[1])
						t.extcrstat.Muppover[2] = uint8(t.sdiag.Val4[2])
						t.extcrstat.Muppover[3] = uint8(t.sdiag.Val4[3])

						//for test only!
						//t.sdiag.Val4[3] = 100
						//t.sdiag.Val4[2] = 100
						//t.extcrstat.Mupvolt5 = float32(binary.LittleEndian.Uint32(t.sdiag.Val4[:]))
					}
					if nm == "MUPPLAT" {
						t.extcrstat.Mupplat = uint(binary.LittleEndian.Uint32(t.sdiag.Val4[:]))
					}
					if nm == "MUPTEMPC" {
						t.extcrstat.Muptemppc = uint(binary.LittleEndian.Uint32(t.sdiag.Val4[:]))
					}
					if nm == "MUPVOLT5" {
						t.extcrstat.Mupvolt5 = float32(binary.LittleEndian.Uint32(t.sdiag.Val4[:]))
					}
					if nm == "MUPVOLT12" {
						t.extcrstat.Mupvolt5 = float32(binary.LittleEndian.Uint32(t.sdiag.Val4[:]))
					}
					if nm == "MUPVOLT25" {
						t.extcrstat.Mupvolt5 = float32(binary.LittleEndian.Uint32(t.sdiag.Val4[:]))
					}
					if nm == "IPADDR" {
						t.extcrstat.Ipaddr[0] = uint8(t.sdiag.Val4[0])
						t.extcrstat.Ipaddr[1] = uint8(t.sdiag.Val4[1])
						t.extcrstat.Ipaddr[2] = uint8(t.sdiag.Val4[2])
						t.extcrstat.Ipaddr[3] = uint8(t.sdiag.Val4[3])
					}
				}
			}
			if t.nmstr == "CrateInits" {
				for {
					err := t.crhd.Rd(t.bdt)
					if err != nil {

						//fmt.Println("cr ebd!!!")
						t.EndOfCr()
						break
					}
				}
			}

		}
		//fmt.Println(t.nmstr, t.szstr)
	}

	t.iocon.syn1.Lock()
	t.iocon.luaSt.CallByParam(LUA_P(t.iocon.luaSt, "End_Pack", 0))
	t.iocon.syn1.Unlock()

	fmt.Println()
}

//вызывается в конце обработки каждого каркаса
func (t *sv3w_pack) EndOfCr() {
	pubstats.PutCr(t.crhd.Racknum, t.crhd.Cratenum, &t.crhd, &t.extcrstat)
}

//вызывается в конце обработки структур уно
func (t *sv3w_pack) EndOfUNO() {
	headstats.PutHead(uint8(t.uh.Racknum), &t.uh, &t.rkdrycont)
}

func (t *sv3w_pack) Lua_Add_ModDiag2(iid uint32, dgname string, val int) {
	t.iocon.luaSt.CallByParam(LUA_P(t.iocon.luaSt, "Add_ModDiag2", 0),
		lua.LNumber(iid), lua.LString(dgname), lua.LNumber(val))
}
func (t *sv3w_pack) Lua_Add_CrateHead(crhd *sv3w_CrateHead) {
	var criid sv3w_input_id_el
	//при включении может одноразово прийти 0 маг.
	criid.Bus = crhd.Busnum
	criid.Chnum = 0
	criid.Crate = crhd.Cratenum
	criid.Modnum = 0
	criid.Rack = crhd.Racknum
	icriid := criid.ToInt()

	t.iocon.luaSt.CallByParam(LUA_P(t.iocon.luaSt, "Add_CrateHead", 0),
		lua.LNumber(icriid.Id), lua.LNumber(crhd.Basetime.ToLnxMs()),
		lua.LString(crhd.Basetime.ToStr()))
}

func (t *sv3w_pack) PackStat(puh *sv3w_UNOHead) {
	if t.packnum < 2 {
		return
	}

	df := t.ctmpack.Sub(t.ptmpack)
	dfms := df.Nanoseconds() / 1000000
	st, _ := t.st2fl.Stat()
	if st.Size() < int64(LogPkStat) {
		t.WrPkStat(t.st1fl, t.ctmpack, dfms)
	}

	//in run statistic -----
	t.pkstat.allcnt++
	m := int64(1000)
	if (dfms > m+200) || (dfms < m-200) {
		t.pkstat.df200++
	} else {
		if (dfms > m+50) || (dfms < m-50) {
			t.pkstat.df50++
		} else {
			t.pkstat.df++
		}
	}
	p1 := float32(t.pkstat.df) / float32(t.pkstat.allcnt) * 100
	p2 := float32(t.pkstat.df50) / float32(t.pkstat.allcnt) * 100
	p3 := float32(t.pkstat.df200) / float32(t.pkstat.allcnt) * 100

	select {
	case <-t.stmsg_tm.C:
		var iid sv3w_input_id_el
		iid.Bus = t.busnum
		iid.Rack = t.racknum

		s := fmt.Sprintf("[<50]=%.2f  [50_200]=%.2f   [>200]=%.2f", p1, p2, p3)
		t.iocon.Msg("stat_pk", iid.ToInt().Id, s)
		t.stmsg_tm.Reset(t.stmsg_tm_int)
	default:
	}
	//-----------

	pt := puh.Unotime.ToLnxMs()
	ct := t.uh.Unotime.ToLnxMs()
	dfms = ct - pt
	//if t.stflag == 0 {
	if LogPkStat > 0 {
		st, _ := t.st2fl.Stat()
		if st.Size() < int64(LogPkStat) {
			t.WrPkStat(t.st2fl, t.uh.Unotime.ToTime(), dfms)
		}
	}

}

func (t *sv3w_pack) IId() (iid sv3w_input_id) {
	var id sv3w_input_id_el
	id.Bus = t.busnum
	id.Rack = t.racknum
	iid = id.ToInt()
	return iid
}

func (t *sv3w_pack) WrPkStat(fl *os.File, curtm time.Time, dfms int64) {
	if t.packnum < 3 {
		return
	}
	s := fmt.Sprintf("%d;%s;%d\n", t.packnum, curtm.Format((time.StampMilli)), dfms)
	//fmt.Println(s)
	fmt.Fprint(fl, s)
}

//----------------------------

type sv3w_io struct {
	con    net.Conn
	ipport string

	pk sv3w_pack
	//hdses  string

	syn1   *sync.Mutex
	synmsg sync.Mutex
	//fmsg *os.File
	//string=kind of msg
	fmsg map[string]*os.File

	luaSt *lua.LState
}

//write msg to file
func (t *sv3w_io) Msg(kind string, iid uint32, msg string) {
	//t.synmsg.Lock()

	var id sv3w_input_id

	st := time.Now().Format(time.StampMilli)
	id.Id = iid
	sid := id.Get()
	//fmt.Fprintf(t.fmsg[kind], "%s:%x:%s\n", kind, iid, msg)
	//fmt.Println("msg", kind, t.fmsg[kind], t.fmsg)

	fst, _ := t.fmsg[kind].Stat()
	if fst.Size() > int64(MaxMsgFileLen) {
		t.fmsg[kind].Truncate(0)
		t.fmsg[kind].Seek(0, 0)
	}

	fmt.Fprintf(t.fmsg[kind], "%s:%s:%s:%s\n", st, kind, sid.Str(), msg)

	//t.synmsg.Unlock()

}

//flush msg to disk   (так лучше работает с браузером)
/*
func (t *sv3w_io) MsgFlush() {
	return

	for _, v := range t.fmsg {
		v.Sync()
	}
}
*/

func (t *sv3w_io) ClIO() {
	var err error
	//t.con.SetDeadline()
	//conn.SetReadDeadline(time.Now().Add(timeoutDuration))

	//t.pk.syn1 = t.syn1

	/*
		t.luaSt = lua.NewState()
		defer t.luaSt.Close()
		if err := t.luaSt.DoFile("gosvrk.lua"); err != nil {
			panic(err)
		}
	*/

	fn := fmt.Sprintf("./Stat/pk_rk%d_bus%d.csv", t.pk.racknum, t.pk.busnum)
	t.pk.st1fl, _ = os.Create(fn)
	fn = fmt.Sprintf("./Stat/unotm_rk%d_bus%d.csv", t.pk.racknum, t.pk.busnum)
	t.pk.st2fl, _ = os.Create(fn)

	fn = fmt.Sprintf("./Stat/pkdmp_rk%d_bus%d.dat", t.pk.racknum, t.pk.busnum)
	if PkDmpFlSz > 0 {
		t.pk.pkdmpfl, _ = os.Create(fn)
	}
	//DEBUG only!!!
	/*
		for {
			t.Test()
			time.Sleep(time.Millisecond * 5)
		}
	*/

	for {

		fmt.Println("try connect")
		t.con, err = net.Dial("tcp", t.ipport)
		if err == nil {
			hdses := make([]byte, 16)
			copy(hdses, "SV3W")
			t.con.Write(hdses)

			for {
				t.con.SetDeadline(time.Now().Add(time.Second * 2))

				bsz := make([]byte, 4)
				_, err = t.con.Read(bsz)
				if err == nil {
					sz := binary.LittleEndian.Uint32(bsz)
					bfpk := make([]byte, sz)
					_, err = io.ReadFull(t.con, bfpk)
					if err == nil {
						start := time.Now()

						t.pk.ParsePk(&bfpk)

						endt := time.Now()
						elapsed := endt.Sub(start)
						fmt.Println("time=", elapsed.Seconds(), "\n")

						t.syn1.Lock()
						t.luaSt.CallByParam(LUA_P(t.luaSt, "TestDt", 0))

						for n := 1; ; n++ {
							k, iid, msg := Lua_GetMsg(t.luaSt, n)
							if k == "" {
								break
							}

							//fmt.Println("go msg", k, iid, msg)
							t.Msg(k, iid, msg)

						}

						t.syn1.Unlock()

						//t.MsgFlush()

					} else {
						break
					}
				} else {
					break
				}

				//end of recv pack
				//LuaSt.SetTop(0)
				//var nm runtime.MemStats
				//runtime.ReadMemStats(&nm)
				//fmt.Println(nm)
				//fmt.Println("mem:", nm.HeapAlloc)

				t.luaSt.SetTop(0) //иначе утечка

				//fmt.Println("lua top:", t.luaSt.GetTop())
			}

			t.Msg("errorpk", t.pk.IId().Id, "disconnect from head")

		} else {
			time.Sleep(time.Second * 10)
		}
	}

}

func (t *sv3w_io) Test() {

	t.syn1.Lock()

	t.luaSt.CallByParam(LUA_P(t.luaSt, "Test", 0))

	for n := 1; ; n++ {
		//k, iid, msg := Lua_GetMsg(t.luaSt, n)
		k, _, _ := Lua_GetMsg(t.luaSt, n)
		if k == "" {
			break
		}

		//fmt.Println("go msg", k, iid, msg)
		//t.Msg(k, iid, msg)

	}

	//var nm runtime.MemStats
	//runtime.ReadMemStats(&nm)
	//fmt.Println(nm)
	//fmt.Println("mem:", nm.HeapAlloc)

	fmt.Println("lua top:", t.luaSt.GetTop())

	t.syn1.Unlock()
}

// нельзя корректно принимать одной прогой с двух маг. !!!!
// связано с проблемой когда каркас включается у него может быть неправильн. маг. и
// он испортит таблицу
var httpservaddr = "0.0.0.0:8091"

func main() {
	//tst := []byte{2, 4, 0, 1}
	//otst := UnpackDry(tst)
	//fmt.Println("oo", otst)

	var iid sv3w_input_id
	iid.Id = 0x01230205
	iidel := iid.Get()
	fmt.Println("iid", iidel)

	L := lua.NewState()
	//LuaSt = L
	defer L.Close()
	if err := L.DoFile("gosvrk.lua"); err != nil {
		panic(err)
	}

	/*
		L.CallByParam(lua.P{
			Fn:      L.GetGlobal("TestFun"), // name of Lua function
			NRet:    1,                      // number of returned values
			Protect: true,                   // return err or panic
		}, lua.LString("Go"), lua.LString("Lua"))
	*/
	L.CallByParam(LUA_P(L, "TestFun", 2), lua.LString("Go"), lua.LString("Lua"))

	var i1 int
	i1 = L.ToInt(-1)
	str, _ := L.Get(-2).(lua.LString)
	L.Pop(2)
	fmt.Println("from lua:", str, i1)

	start := time.Now()
	for i := 0; i < 100000; i++ {
		L.CallByParam(lua.P{
			Fn:      L.GetGlobal("AddDt"), // name of Lua function
			NRet:    0,                    // number of returned values
			Protect: true,                 // return err or panic
		}, lua.LNumber(i), lua.LNumber(i+100))

	}
	t := time.Now()
	elapsed := t.Sub(start)
	fmt.Println("time=", elapsed.Seconds())

	start = time.Now()
	for i := 0; i < 10000; i++ {
		L.CallByParam(lua.P{
			Fn:      L.GetGlobal("GetDt"), // name of Lua function
			NRet:    1,                    // number of returned values
			Protect: true,                 // return err or panic
		}, lua.LNumber(i))
		dt, _ := L.Get(-1).(lua.LNumber)
		L.Pop(1)

		if i == 1000 {
			fmt.Println("dt", dt)
		}
	}
	t = time.Now()
	elapsed = t.Sub(start)
	fmt.Println("time=", elapsed.Seconds())

	//-------

	var io *sv3w_io
	var syn1 sync.Mutex

	//crstats = make(map[uint32]CrStats)
	pubstats.Init()
	headstats.Init()

	var fmsg map[string]*os.File
	fmsg = make(map[string]*os.File)

	//syn1 := make(chan int)

	httpservaddr = string(L.GetGlobal("HttpServ").(lua.LString))

	go RunHttp() //http serv thread

	//fmsg_diagchg, _ := os.Create("./Msg/diagchg.txt")

	//var clips []string

	CreateMsg(L, fmsg)

	ft := int(L.GetGlobal("MsgFlush_Sec").(lua.LNumber))
	go MsgFlush(fmsg, ft) //flush thread

	LogPkStat = int(L.GetGlobal("LogPkStat").(lua.LNumber) * 1024)

	MaxMsgFileLen = int(L.GetGlobal("MaxMsgFileLen").(lua.LNumber) * 1024)

	PkDmpFlSz = int(L.GetGlobal("PkDmpFlSz").(lua.LNumber) * 1024)

	var lvl *lua.LTable
	var lvl2 *lua.LTable
	lvl = L.GetGlobal("ClIP").(*lua.LTable)
	//fmt.Println("lua type=", lvl.Len())
	for i := 1; i <= lvl.Len(); i++ {
		lvl2 = lvl.RawGetInt(i).(*lua.LTable)

		//if lvl2 == lua.LNil {
		//	break
		//}
		v := lvl2.RawGetInt(1)
		if v == lua.LNil {
			continue
		}
		rk := lvl2.RawGetInt(2)
		if rk == lua.LNil {
			continue
		}
		bus := lvl2.RawGetInt(3)
		if bus == lua.LNil {
			continue
		}

		//fmt.Println("lua type2=", v.String(), v, rk, bus)
		//clips = append(clips, v.String())

		io = new(sv3w_io)
		//io.fmsg = make(map[string]*os.File)
		//io.fmsg["diagchg"] = fmsg_diagchg
		//CreateMsg(io)
		io.luaSt = L
		io.fmsg = fmsg
		io.syn1 = &syn1
		//io.ipport = clips[icl] //"127.0.0.1:19747"
		io.ipport = v.String()
		io.pk.racknum = uint8(rk.(lua.LNumber))
		io.pk.busnum = uint8(bus.(lua.LNumber))
		io.pk.iocon = io
		io.pk.Init()
		fmt.Println("chan:", i, io.ipport, io.pk.racknum, io.pk.busnum)
		go io.ClIO()

	}

	for {
		time.Sleep(time.Millisecond * 5000)
	}

}

func RunHttp() {
	fmt.Println("Run http server at: ", httpservaddr)
	http.HandleFunc("/QF", HttpHandler)
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.ListenAndServe(httpservaddr, nil)
}

//func CreateMsg(io *sv3w_io) {
func CreateMsg(LuaSt *lua.LState, fm map[string]*os.File) {
	var mt *lua.LTable
	mt = LuaSt.GetGlobal("MsgKind").(*lua.LTable)
	//io.fmsg = make(map[string]*os.File)
	for i := 1; i <= mt.Len(); i++ {
		sk := mt.RawGetInt(i).String()
		fh, _ := os.Create("./Msg/" + sk + ".txt")
		//io.fmsg[sk] = fh
		fm[sk] = fh
		//fmt.Println("crmsg", sk, fm[sk])
	}
}

//поток - flush messages
func MsgFlush(fmsg map[string]*os.File, flper_sec int) {
	//fmt.Println("flash", flper_sec)
	msgticker := time.NewTicker(time.Second * time.Duration(flper_sec))
	for {

		select {
		case <-msgticker.C:
			for _, v := range fmsg {
				v.Sync()
				fmt.Println("Sync")
			}

		default:
			time.Sleep(time.Second * 2)
		}
	}

}

//---------
func HttpHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Fprintf(w, "Handler!\n")
	//fmt.Fprintf(w, html.EscapeString(r.URL.RequestURI()))
	//fmt.Fprintf(w, html.EscapeString(r.URL.RawQuery))
	//FRead(w, r.URL.RawQuery)

	if html.EscapeString(r.URL.RawQuery) == "crstats" {
		//map сортируется при json.Encode
		je := json.NewEncoder(w)
		je.Encode(pubstats.Rkstats)
	}
	if html.EscapeString(r.URL.RawQuery) == "headstats" {
		//map сортируется при json.Encode
		je := json.NewEncoder(w)
		je.Encode(headstats.Headstats)
	}

}

//---------
func UnpackDry(ipadt [12]uint8) (v []byte) {
	//var v []byte
	vn := 0
	v = make([]byte, len(ipadt)*8)
	var b uint
	for n := 0; n < len(ipadt); n++ {
		for b = 0; b < 8; b++ {
			v[vn] = (ipadt[n] >> b) & 1
			vn++
		}
	}
	return v
}
