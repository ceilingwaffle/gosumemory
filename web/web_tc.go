package web

import (
	"log"
	"net/http"

	"github.com/l3lackShark/gosumemory/config"
	"github.com/l3lackShark/gosumemory/memory"
	"github.com/spf13/cast"
)

type wsTourneyClientMessage struct {
	MID  string `json:"mid"`  // message ID (e.g. connected)
	TCID int    `json:"tcid"` // tourney client ID
	PID  int    `json:"pid"`  // tourney client process ID
}

func wsEndpointTourneyClients(w http.ResponseWriter, r *http.Request) {
	if cast.ToBool(config.Config["cors"]) {
		enableCors(&w)
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	defer ws.Close()
	for {
		m := wsTourneyClientMessage{}
		err = ws.ReadJSON(&m)
		if err != nil {
			log.Println("Error reading json.", err)
		}
		log.Printf("Received TC message: %#v\n", m)
		if m.MID == "connected" {
			// respond to 'connected' message with another message containing the tcid
			var latestInjectedTourneyProc = memory.GetLatestInjectedTourneyProc()
			tcdata := wsTourneyClientMessage{
				MID:  "tid-assigned",
				PID:  latestInjectedTourneyProc.PROC.Pid(),
				TCID: latestInjectedTourneyProc.TCID,
			}
			err = ws.WriteJSON(tcdata)
			if err != nil {
				log.Println(err.Error())
			}

			// TODO: replace this with: if memory.TourneyProcs.contains(connectedTourneyClients.PID)
			if len(memory.InjectedTourneyProcs) >= len(memory.TourneyProcs) {
				// all TC's now have overlays injected; do not attempt to inject more
				continue
			}
			_, _, err = memory.InjectNextTourneyProc()
			if err != nil {
				log.Println("Error injecting tourney proc.", err)
			}
		}
	}
}
