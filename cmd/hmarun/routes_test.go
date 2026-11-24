package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
)

const validRoutes = `{"schema":"hmarun.routes/v1","implementation":{"agent":"jax","model":"gpt-6.1-terra","provider":"openai","provider_family":"openai","model_family":"gpt-family","effort":"medium"},"review":{"agent":"ming","model":"claude-opus-5-5","provider":"anthropic","provider_family":"anthropic","model_family":"claude-family","effort":"medium"}}`

func TestLoadRoutes(t *testing.T) {
	write := func(t *testing.T, data []byte) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "routes.json")
		if err := os.WriteFile(p, data, 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	got, err := loadRoutes(write(t, []byte(validRoutes)))
	if err != nil {
		t.Fatal(err)
	}
	if got[hoststate.RoleImplementation].Agent != "jax" || got[hoststate.RoleReview].Model != "claude-opus-5-5" {
		t.Fatalf("unexpected routes: %+v", got)
	}
	for _, role := range []string{"implementation", "review"} {
		for _, field := range []string{"agent", "model", "provider", "provider_family", "model_family", "effort"} {
			t.Run(role+"/missing_"+field, func(t *testing.T) {
				var doc map[string]any
				if err := json.Unmarshal([]byte(validRoutes), &doc); err != nil {
					t.Fatal(err)
				}
				delete(doc[role].(map[string]any), field)
				data, _ := json.Marshal(doc)
				_, err := loadRoutes(write(t, data))
				if err == nil || !strings.Contains(err.Error(), role+"."+field) {
					t.Fatalf("error = %v", err)
				}
			})
		}
	}
	for _, role := range []string{"implementation", "review"} {
		for _, field := range []string{"agent", "model", "provider", "provider_family", "model_family", "effort"} {
			values := []string{"   "}
			if field == "agent" {
				values = append(values, "../agent", "Agent", "user@host")
			} else if field == "effort" {
				values = append(values, "extreme")
			} else {
				values = append(values, "-value", "a b", "a\tb", "é", strings.Repeat("a", 129))
			}
			for _, value := range values {
				t.Run(role+"/"+field+"/"+value, func(t *testing.T) {
					var doc map[string]any
					if err := json.Unmarshal([]byte(validRoutes), &doc); err != nil {
						t.Fatal(err)
					}
					doc[role].(map[string]any)[field] = value
					data, err := json.Marshal(doc)
					if err != nil {
						t.Fatal(err)
					}
					_, err = loadRoutes(write(t, data))
					if err == nil || !strings.Contains(err.Error(), role+"."+field) {
						t.Fatalf("error = %v", err)
					}
				})
			}
		}
	}
	for _, tc := range []struct{ name, data, want string }{
		{"schema", strings.Replace(validRoutes, "hmarun.routes/v1", "wrong", 1), "schema"},
		{"missing schema", strings.Replace(validRoutes, `"schema":"hmarun.routes/v1",`, "", 1), "schema"},
		{"unknown", strings.Replace(validRoutes, `"agent":"jax"`, `"unexpected":1,"agent":"jax"`, 1), "unknown field"},
		{"same agent", strings.Replace(validRoutes, `"agent":"ming"`, `"agent":"jax"`, 1), "must differ"},
		{"bad agent", strings.Replace(validRoutes, `"agent":"jax"`, `"agent":"../jax"`, 1), "implementation.agent"},
		{"bad effort", strings.Replace(validRoutes, `"effort":"medium"`, `"effort":"extreme"`, 1), "implementation.effort"},
		{"trailing", validRoutes + ` {}`, "trailing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := write(t, []byte(tc.data))
			_, err := loadRoutes(path)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %s", err, tc.want)
			}
			if strings.Contains(tc.name, "schema") && !strings.Contains(err.Error(), path) {
				t.Fatalf("error = %v, want path %s", err, path)
			}
		})
	}
}

func TestRunnerDefaultRoutesAndStubIgnore(t *testing.T) {
	contract, plan := hmarunTestDocs(t, t.TempDir())
	cfg := runConfig{storeDir: t.TempDir(), repoRoot: hmarunTestRepo(t), runID: "routes-test", unit: "U1", contractPath: contract, planPath: plan, hmaBin: buildHMABinary(t), agentDispatchBin: "unused", autoApprove: true}
	r, err := newRunner(cfg, bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.dispatchConfigs, dispatchConfigFor) {
		t.Fatalf("default routes = %+v", r.dispatchConfigs)
	}
	routesPath := filepath.Join(t.TempDir(), "routes.json")
	if err := os.WriteFile(routesPath, []byte(validRoutes), 0600); err != nil {
		t.Fatal(err)
	}
	cfg.routesPath = routesPath
	r, err = newRunner(cfg, bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	want, err := loadRoutes(routesPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.dispatchConfigs, want) {
		t.Fatalf("configured routes = %+v, want %+v", r.dispatchConfigs, want)
	}
	impl, review := want[hoststate.RoleImplementation], want[hoststate.RoleReview]
	wantImpl := hmarunRoute(impl.Agent, impl.Provider, impl.ProviderFamily, impl.Model, impl.ModelFamily, impl.Effort, false)
	wantReviewer := hmarunRoute(review.Agent, review.Provider, review.ProviderFamily, review.Model, review.ModelFamily, review.Effort, true)
	if !reflect.DeepEqual(r.impl, wantImpl) || !reflect.DeepEqual(r.reviewer, wantReviewer) {
		t.Fatalf("runner routes = %+v, %+v; want %+v, %+v", r.impl, r.reviewer, wantImpl, wantReviewer)
	}
	cfg.stubDispatch = true
	cfg.routesPath = filepath.Join(t.TempDir(), "missing")
	if _, err := newRunner(cfg, bytes.NewReader(nil), &bytes.Buffer{}); err != nil {
		t.Fatalf("stub should ignore routes: %v", err)
	}
}
