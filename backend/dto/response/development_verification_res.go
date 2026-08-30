package response

type DevelopmentVerificationDetailRes struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type DevelopmentVerificationStatusRes struct {
	ScenarioId string                             `json:"scenario_id"`
	State      string                             `json:"state"`
	Available  bool                               `json:"available"`
	ItemCount  int                                `json:"item_count"`
	Summary    string                             `json:"summary"`
	Details    []DevelopmentVerificationDetailRes `json:"details"`
}

type DevelopmentVerificationAccountRes struct {
	UserName string `json:"user_name"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Expected string `json:"expected"`
}

type DevelopmentVerificationPrepareRes struct {
	Status   DevelopmentVerificationStatusRes    `json:"status"`
	Accounts []DevelopmentVerificationAccountRes `json:"accounts"`
}
