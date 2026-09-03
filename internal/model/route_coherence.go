package model

// RouteProfile is a host-supplied profile snapshot used only as an input to
// coherence validation. It is not a portable repository artifact and its
// availability claim is not proof of capability or trustworthiness.
type RouteProfile struct {
	ProfileDigest       string           `json:"profile_digest"`
	ProviderFamily      string           `json:"provider_family"`
	ModelFamily         string           `json:"model_family"`
	ExecutionContext    ExecutionContext `json:"execution_context"`
	AccessServiceDigest string           `json:"access_service_digest"`
	CapabilityClasses   []string         `json:"capability_classes"`
	PermissionClasses   []string         `json:"permission_classes"`
	Eligible            bool             `json:"eligible"`
}

// RouteProposal is a portable claim that one profile is proposed for one
// route role. It does not select or execute the route.
type RouteProposal struct {
	ProfileDigest     string           `json:"profile_digest"`
	ProviderFamily    string           `json:"provider_family"`
	ModelFamily       string           `json:"model_family"`
	ExecutionContext  ExecutionContext `json:"execution_context"`
	CapabilityClasses []string         `json:"capability_classes"`
	PermissionClasses []string         `json:"permission_classes"`
}
