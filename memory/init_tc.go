package memory

import (
	"log"

	"github.com/l3lackShark/gosumemory/injctr"
	"github.com/l3lackShark/gosumemory/mem"
)

type injectedTourneyProc struct {
	PROC mem.Process
	TCID int
}

var InjectedTourneyProcs []injectedTourneyProc
var TourneyProcs []mem.Process

func initTourneyClientOverlayInjection(tourneyProcs []mem.Process) {
	TourneyProcs = make([]mem.Process, len(tourneyProcs))
	InjectNextTourneyProc()
}

func InjectNextTourneyProc() (int, int, error) {
	// if len(InjectedTourneyProcs) >= len(TourneyProcs)-1 {
	// 	return 0, 0, nil
	// 	//return 0, 0, errors.New("maximum tourney client procs already injected")
	// }
	var tcid = len(InjectedTourneyProcs)
	var proc = tourneyProcs[tcid]
	var pid, err = injectTourneyProc(proc, tcid)
	return pid, tcid, err
}

func injectTourneyProc(proc mem.Process, tcid int) (int, error) {
	var pid = proc.Pid()
	var err error = injctr.Injct(pid)
	if err != nil {
		log.Printf("Failed to inject into osu's process, in game overlay will be unavailabe. %e\n", err)
	} else {
		var injectedTourneyProc = injectedTourneyProc{}
		injectedTourneyProc.PROC = proc
		injectedTourneyProc.TCID = tcid
		InjectedTourneyProcs = append(InjectedTourneyProcs, injectedTourneyProc)
	}
	return pid, err
}

func GetLatestInjectedTourneyProc() injectedTourneyProc {
	return InjectedTourneyProcs[len(InjectedTourneyProcs)-1]
}
