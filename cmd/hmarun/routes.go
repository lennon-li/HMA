package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	hoststate "github.com/lennon-li/HMA/internal/host"
)

type routeJSON struct {
	Agent          string `json:"agent"`
	Model          string `json:"model"`
	Provider       string `json:"provider"`
	ProviderFamily string `json:"provider_family"`
	ModelFamily    string `json:"model_family"`
	Effort         string `json:"effort"`
}

type routesJSON struct {
	Schema         string    `json:"schema"`
	Implementation routeJSON `json:"implementation"`
	Review         routeJSON `json:"review"`
}

var agentNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
var routeLabelPattern = regexp.MustCompile(`^[!-~]+$`)

func defaultDispatchConfigs() map[hoststate.DispatchRole]dispatchAgentConfig {
	configs := make(map[hoststate.DispatchRole]dispatchAgentConfig, len(dispatchConfigFor))
	for role, config := range dispatchConfigFor {
		configs[role] = config
	}
	return configs
}

func loadRoutes(path string) (map[hoststate.DispatchRole]dispatchAgentConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("routes %s: %w", path, err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var routes routesJSON
	if err := dec.Decode(&routes); err != nil {
		return nil, fmt.Errorf("routes %s: %w", path, err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("routes %s: unexpected trailing JSON value", path)
		}
		return nil, fmt.Errorf("routes %s: trailing data: %w", path, err)
	}
	if routes.Schema != "hmarun.routes/v1" {
		return nil, fmt.Errorf("routes %s: schema must be hmarun.routes/v1", path)
	}
	result := make(map[hoststate.DispatchRole]dispatchAgentConfig, 2)
	for _, item := range []struct {
		name  string
		role  hoststate.DispatchRole
		route routeJSON
	}{
		{"implementation", hoststate.RoleImplementation, routes.Implementation}, {"review", hoststate.RoleReview, routes.Review},
	} {
		r := item.route
		for _, field := range []struct{ name, value string }{
			{"agent", r.Agent}, {"model", r.Model}, {"provider", r.Provider}, {"provider_family", r.ProviderFamily}, {"model_family", r.ModelFamily}, {"effort", r.Effort},
		} {
			if strings.TrimSpace(field.value) == "" {
				return nil, fmt.Errorf("routes: %s.%s is required and must be non-empty", item.name, field.name)
			}
		}
		if !agentNamePattern.MatchString(r.Agent) {
			return nil, fmt.Errorf("routes: %s.agent must match %s", item.name, agentNamePattern)
		}
		for _, field := range []struct{ name, value string }{
			{"model", r.Model}, {"provider", r.Provider}, {"provider_family", r.ProviderFamily}, {"model_family", r.ModelFamily},
		} {
			if len(field.value) > 128 || field.value[0] == '-' || !routeLabelPattern.MatchString(field.value) {
				return nil, fmt.Errorf("routes: %s.%s must be at most 128 printable ASCII characters without whitespace and must not start with '-'", item.name, field.name)
			}
		}
		switch r.Effort {
		case "low", "medium", "high", "xhigh", "max":
		default:
			return nil, fmt.Errorf("routes: %s.effort must be low, medium, high, xhigh, or max", item.name)
		}
		result[item.role] = dispatchAgentConfig{Agent: r.Agent, Model: r.Model, Provider: r.Provider, ProviderFamily: r.ProviderFamily, ModelFamily: r.ModelFamily, Effort: r.Effort}
	}
	if routes.Implementation.Agent == routes.Review.Agent {
		return nil, fmt.Errorf("routes: implementation.agent and review.agent must differ")
	}
	return result, nil
}
