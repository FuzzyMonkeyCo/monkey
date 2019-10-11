package runtime

import (
	"context"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

func (rt *runtime) reset(ctx context.Context) error {
	if err := rt.clt.Send(&fm.Clt{
		Msg: &fm.Clt_Msg{
			Msg: &fm.Clt_Msg_ResetProgress_{
				ResetProgress: &fm.Clt_Msg_ResetProgress{
					Status: fm.Clt_Msg_ResetProgress_started,
				}}}}); err != nil {
		log.Println("[ERR]", err)
		return err
	}

	resetter := rt.modelers[0].GetSUTResetter()
	start := time.Now()
	err := resetter.ExecReset(ctx, rt.client)
	elapsed := uint64(time.Since(start))

	if err != nil {
		var reason []string
		if resetErr, ok := err.(*Error); ok {
			reason = resetErr.reason()
		} else {
			reason = strings.Split(err.Error(), "\n")
		}

		if err2 := clt.Send(&fm.Clt{
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

	if err = clt.Send(&fm.Clt{
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
