// stat_structs
package main

//import "fmt"
type ExtCrStats struct {
	Atmpover  uint32
	Muppover  [4]uint8
	Mupplat   uint //2-platform arm
	Muptemppc uint
	Mupvolt5  float32
	Mupvolt12 float32
	Mupvolt25 float32
	Ipaddr    [4]uint8
}

type CrStats struct {
	Crhd    sv3w_CrateHead
	ExtStat ExtCrStats
	Cnt     uint
	//prevcnt uint
	//livefl  int8
}

type RkStats struct {
	Crstats map[uint8]CrStats
}

type PubStats struct {
	Rkstats map[uint8]RkStats
}

func (t *PubStats) Init() {
	t.Rkstats = make(map[uint8]RkStats)

}
func (t *PubStats) PutCr(rkn uint8, crn uint8, hd *sv3w_CrateHead, ext *ExtCrStats) {

	rk, f := t.Rkstats[rkn]

	if f == false {
		rk = *new(RkStats)
		rk.Crstats = make(map[uint8]CrStats)
		//fmt.Println("new rk")
	}
	t.Rkstats[rkn] = rk

	cs, f1 := rk.Crstats[crn]
	if f1 == false {
		cs = *new(CrStats)
		//fmt.Println("new cr")
	}

	cs.Cnt++
	cs.Crhd = *hd
	cs.ExtStat = *ext

	rk.Crstats[crn] = cs

}

//--------
type RkDryCont struct {
	RawIpa [12]uint8
	Nm     map[string]uint8
}
type HeadStats struct {
	Headhd  sv3w_UNOHead
	Drycont RkDryCont
	Cnt     uint
}
type HeadStatsM struct {
	Headstats map[uint8]HeadStats //key-rack num
}

func (t *HeadStatsM) Init() {
	t.Headstats = make(map[uint8]HeadStats)
}
func (t *HeadStatsM) PutHead(rkn uint8, hd *sv3w_UNOHead, dry *RkDryCont) {
	hs := t.Headstats[rkn]
	hs.Cnt++
	hs.Headhd = *hd
	hs.Drycont = *dry
	t.Headstats[rkn] = hs

}

//--------

var pubstats PubStats
var headstats HeadStatsM
