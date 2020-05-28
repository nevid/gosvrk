package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

//%x = 0xrrcbmmcc (rr-rack c-bus b-crate mm-mod cc-chan
type sv3w_input_id struct {
	//Chnum    int8
	//Modnum   int8
	//CrateBus int8
	//Rack     int8
	Id uint32
}

func (t *sv3w_input_id) Get() (iid sv3w_input_id_el) {
	var i uint8
	tb := new(bytes.Buffer)
	binary.Write(tb, binary.LittleEndian, t.Id)
	binary.Read(tb, binary.LittleEndian, &i)
	iid.Chnum = i
	binary.Read(tb, binary.LittleEndian, &i)
	iid.Modnum = i
	binary.Read(tb, binary.LittleEndian, &i)
	iid.Crate = i & 0x0f
	iid.Bus = i >> 4
	binary.Read(tb, binary.LittleEndian, &i)
	iid.Rack = i
	return iid
}

type sv3w_input_id_el struct {
	Chnum  uint8
	Modnum uint8
	Crate  uint8
	Bus    uint8
	Rack   uint8
}

func (t *sv3w_input_id_el) Str() (s string) {
	s = fmt.Sprintf("rk=%d b=%d cr=%d md=%d ch=%d", t.Rack, t.Bus, t.Crate, t.Modnum, t.Chnum)
	return s
}
func (t *sv3w_input_id_el) ToInt() (iid sv3w_input_id) {
	//crbus := t.Bus*16 + t.Crate
	//iid.Id = uint32(t.Rack*2 ^ 24 + crbus*2 ^ 16 + t.Modnum*2 ^ 8 + t.Chnum)
	//print("iid",inputid,string.format("%x",inputid))
	var br uint8
	br = (t.Bus << 4) | t.Crate
	iid.Id = uint32(t.Rack)<<24 | uint32(br)<<16 | uint32(t.Modnum)<<8 | uint32(t.Chnum)

	return iid

}

type sv3w_SVTime struct {
	Year         int16
	Month        int8
	Day          int8
	Hour         int8
	Minute       int8
	Second       int8
	Milliseconds int16
}

func (t *sv3w_SVTime) ToLnxMs() (ltm int64) {
	if t.Year == 0 {
		return 0
	}
	nanosec := int(t.Milliseconds) * 1000000
	var m time.Month
	m = time.Month(t.Month)
	tm := time.Date(int(t.Year), m, int(t.Day), int(t.Hour), int(t.Minute), int(t.Second),
		nanosec, time.UTC)
	ltm = tm.UnixNano() / 1000000
	return ltm
}
func (t *sv3w_SVTime) ToStr() (st string) {
	nanosec := int(t.Milliseconds) * 1000000
	tm := time.Date(int(t.Year), time.Month(t.Month), int(t.Day), int(t.Hour), int(t.Minute), int(t.Second),
		nanosec, time.UTC)
	st = tm.Format(time.StampMilli)
	return st
}
func (t *sv3w_SVTime) ToTime() (tm time.Time) {
	nanosec := int(t.Milliseconds) * 1000000
	tm = time.Date(int(t.Year), time.Month(t.Month), int(t.Day), int(t.Hour), int(t.Minute), int(t.Second),
		nanosec, time.UTC)
	return tm
}

type sv3w_UNOHead struct {
	Apnum            int8
	Racknum          int8
	Busnum           int8
	Basetime         sv3w_SVTime
	Unotime          sv3w_SVTime
	Crateplacescount int8
	Modsplacescount  int8
	Cyclescount      int8
}

func (t *sv3w_UNOHead) Rd(bf *bytes.Buffer) error {
	return binary.Read(bf, binary.LittleEndian, t)
}

type sv3w_UNODryRaw struct {
	Pos      uint8
	Databyte uint8
}

func (t *sv3w_UNODryRaw) Rd(bf *bytes.Buffer) error {
	return binary.Read(bf, binary.LittleEndian, t)
}

type sv3w_CrateHead struct {
	Racknum       uint8
	Busnum        uint8
	Cratenum      uint8
	Protocol_ver  uint16
	Struct_ver    uint16
	Crate_type    uint16      //1-одномаг. 2-двумаг.
	Cyclemode     uint8       //режим: 10=десять измерений в сек (в этом режиме, возможна передача, 10 измерений от модулей МДПЗ и одного от остальных модулей); 1=одно
	Basetime      sv3w_SVTime //базовое время, все смещения в пакете данного каркаса от него (фактически является временем начала сбора очередного пакета)
	Cyclecount    uint8       // не определена
	Modplacecount uint8       //кол-во посадоч. мест модулей
	Tme_pc104     sv3w_SVTime
}

func (t *sv3w_CrateHead) Rd(bf *bytes.Buffer) error {
	return binary.Read(bf, binary.LittleEndian, t)
}

//------
type sv3w_CrateSDiag struct {
	Timeoff  int16
	Trueflag uint8
	Inputid  sv3w_input_id
	Name     [11]byte
	Val4     [4]byte
} //22 байт
func (t *sv3w_CrateSDiag) Rd(bf *bytes.Buffer) error {
	return binary.Read(bf, binary.LittleEndian, t)
}

//-----

type sv3w_ModuleStatus_Uni2 struct {
	TipM   uint8 //тип модуля (считанный из него) при =0 - модуль отсутствует
	ManNum int16 //заводской номер модуля  (считанный из него) default=0

	Cable        uint8 // -1=отключен, 0=не определен, 1=включен
	Work_lamp    int8
	Control_1    int8
	Control_2    int8
	Test_lamp    int8
	Error_lamp   int8 //Горит индикатор "ОТКАЗ"
	Module_good  int8 //Исправность / неисправность модуля
	Module_ready int8 //Готовность модуля
	Watchdog     int8 //1 -  Сработал сторожевой таймер, обслуживаемый
	//дублирующей системной магистралью (нормально, когда=-1)
	L int8
	//MRS
	Adj_watchdog   int8      // состояние сторожевого таймера по чужой магистрали =1 сработал
	Self_watchdog  int8      //сигнал сторожевого таймера по своей магистрали =1 сработал
	Adjmodule_good int8      //состояние сигнала исравность по чужой магистрали =1 исправен
	Mrstime        [2]uint16 //два счетчика текущ. времени МРС
	Mrsrle         uint16    //состояние контактных групп реле

	//MVC
	Mvcmask uint16 //состояние масок инициатив МВЦ
	Mvctime uint32 //счетчик времени МВЦ

	Mic [11]byte
} // 39 байт

type sv3w_ModsDiag struct {
	Cyclenum uint8         //Номер измер. цикла (1б.)
	Timeoff  int16         //время (2б. знак) (смещ. от базы в мс)
	Trueflag uint8         //Признак достоверности (1 байт)
	Inputid  sv3w_input_id //Номер входа (4 б.))
	Modpos   uint8         //Позиция модуля (1 байт)
	Dg       sv3w_ModuleStatus_Uni2
}

func (t *sv3w_ModsDiag) Rd(bf *bytes.Buffer) error {
	return binary.Read(bf, binary.LittleEndian, t)
}

//------------------------

/*
func sv3w_Rd(bf *bytes.Buffer, t *interface{}) {

	binary.Read(bf, binary.LittleEndian, *t)
}
*/
