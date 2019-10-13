package runtime

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter/shell"
	"go.starlark.net/starlark"
)

func (rt *runtime) reset(ctx context.Context, msg int) error {
	if err := rt.client.Send(&fm.Clt{
		Msg: &fm.Clt_Msg{
			Msg: &fm.Clt_Msg_ResetProgress_{
				ResetProgress: &fm.Clt_Msg_ResetProgress{
					Status: fm.Clt_Msg_ResetProgress_started,
				}}}}); err != nil {
		log.Println("[ERR]", err)
		return err
	}

	rsttr := rt.models[0].GetResetter()
	start := time.Now()
	err := rsttr.ExecReset(ctx, rt.client)
	elapsed := uint64(time.Since(start))

	if err != nil {
		var reason []string
		if resetErr, ok := err.(*resetter.Error); ok {
			reason = resetErr.Reason()
		} else {
			reason = strings.Split(err.Error(), "\n")
		}

		if err2 := rt.client.Send(&fm.Clt{
			Msg: &fm.Clt_Msg{
				Msg: &fm.Clt_Msg_ResetProgress_{
					ResetProgress: &fm.Clt_Msg_ResetProgress{
						Status: fm.Clt_Msg_ResetProgress_failed,
						TsDiff: elapsed,
						Reason: reason,
					}}}}); err != nil {
			log.Println("[ERR]", err2)
			// nothing to continue on
		}
		return err
	}

	if err = rt.client.Send(&fm.Clt{
		Msg: &fm.Clt_Msg{
			Msg: &fm.Clt_Msg_ResetProgress_{
				ResetProgress: &fm.Clt_Msg_ResetProgress{
					Status: fm.Clt_Msg_ResetProgress_ended,
					TsDiff: elapsed,
				}}}}); err != nil {
		log.Println("[ERR]", err)
	}
	return err
}

func newFromKwargs(modelerName string, r starlark.StringDict) (resetter.Resetter, error) {
	const (
		tExecReset = "ExecReset"
		tExecStart = "ExecStart"
		tExecStop  = "ExecStop"
	)
	var (
		ok bool
		v  starlark.Value
		vv starlark.String
		t  string
		// TODO: other Resetter.s
		rsttr = &resetter_shell.Shell{}
	)
	t = tExecStart
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		rsttr.Start = vv.GoString()
	}
	t = tExecReset
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		rsttr.Rst = vv.GoString()
	}
	t = tExecStop
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		rsttr.Stop = vv.GoString()
	}
	if len(r) != 0 {
		return nil, fmt.Errorf("unexpected arguments to %s(): %s", modelerName, strings.Join(r.Keys(), ", "))
	}
	return rsttr, nil
}
