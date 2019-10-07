package fm

type cltDoer interface {
	isSrv_Msg_Msg()
	do() (err error)
}
