package runtime

func (rt *Runtime) ProgressCampaignSummary() bool {
	return rt.progress.CampaignSummary()
}
