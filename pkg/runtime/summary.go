package runtime

// TestingCampaignOutcomer describes a testing campaign's results
type TestingCampaignOutcomer interface {
	error
	isTestingCampaignOutcomer()
}

var _ TestingCampaignOutcomer = (*TestingCampaignSuccess)(nil)
var _ TestingCampaignOutcomer = (*TestingCampaignFailure)(nil)
var _ TestingCampaignOutcomer = (*TestingCampaignFailureDueToResetterError)(nil)

// TestingCampaignSuccess indicates no bug was found during fuzzing.
type TestingCampaignSuccess struct{}

// TestingCampaignFailure indicates a bug was found during fuzzing.
type TestingCampaignFailure struct{}

// TestingCampaignFailureDueToResetterError indicates a bug was found during reset.
type TestingCampaignFailureDueToResetterError struct{}

func (tc *TestingCampaignSuccess) Error() string { return "Found no bug" }
func (tc *TestingCampaignFailure) Error() string { return "Found a bug" }
func (tc *TestingCampaignFailureDueToResetterError) Error() string {
	return "Something went wrong while resetting the system to a neutral state."
}

func (tc *TestingCampaignSuccess) isTestingCampaignOutcomer()                   {}
func (tc *TestingCampaignFailure) isTestingCampaignOutcomer()                   {}
func (tc *TestingCampaignFailureDueToResetterError) isTestingCampaignOutcomer() {}

func plural(s string, n uint32) string {
	if n == 1 {
		return s
	}
	return s + "s"
}
