package reset

import (
	"log"
)

var (
	isRunning = false
	// HadExecError to exit with statusFailedExec
	HadExecError = false
	// To not post-stop after stop
	wasStopped = false
)

func (act *ReqDoReset) exec(mnk *Monkey) (err error) {
	mnk.progress.state("🙉")

	var nxt Action
	if nxt, err = ExecuteScript(mnk.Cfg, act.GetKind()); err != nil {
		return
	}

	if err = mnk.ws.cast(nxt); err != nil {
		log.Println("[ERR]", err)
	}
	return
}
