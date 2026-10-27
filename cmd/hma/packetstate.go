package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/lennon-li/HMA/internal/engine"
	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/model"
)

// addContractState attaches the complete, approved contract and plan unit.
func addContractState(st *packetCoreState, storePath string, core []model.Record, unit, contractPath, policyPath, planPath string) error {
	_, _, err := addContractStateWithPlan(st, storePath, core, unit, contractPath, policyPath, planPath)
	return err
}

// addContractStateWithPlan validates the approved contract, policy and plan.
// A caller may retain the returned plan only after all packet refusal checks pass.
func addContractStateWithPlan(st *packetCoreState, storePath string, core []model.Record, unit, contractPath, policyPath, planPath string) (*model.Plan, string, error) {
	bound := engine.BoundContractDigest(core)
	if bound == "" {
		if contractPath != "" {
			return nil, "", errors.New(string(engine.ReasonContractMismatch))
		}
		return nil, "", nil
	}
	if contractPath == "" {
		return nil, "", errors.New("HOST_REVIEW_PACKET_INPUT_MISSING")
	}
	raw, err := os.ReadFile(contractPath)
	if err != nil {
		return nil, "", errors.New("HOST_REVIEW_PACKET_INPUT_MISSING")
	}
	contract, digest, err := model.ParseAcceptanceContract(raw)
	if err != nil || digest != bound {
		return nil, "", errors.New(string(engine.ReasonContractMismatch))
	}
	st.Contract = &packetContract{Digest: digest, AcceptanceContract: contract}
	if policyPath == "" {
		return nil, "", errors.New("HOST_REVIEW_PACKET_INPUT_MISSING")
	}
	var policy model.RoutePolicy
	if err := decodeStrictFile(policyPath, &policy); err != nil {
		return nil, "", errors.New("HOST_REVIEW_PACKET_INPUT_MISSING")
	}
	projection, err := model.NormalizeRoutePolicy(policy)
	policyDigest, digestErr := engine.RoutePolicyDigest(projection)
	if err != nil || digestErr != nil || policyDigest != contract.PolicyDigest {
		return nil, "", errors.New(hoststate.ReasonPolicyNotCurrent)
	}
	st.Policy = &projection
	st.PolicyDigest = contract.PolicyDigest
	st.RequestBasis = "contract.intent"
	st.Request = contract.Intent
	var suppliedPlan *model.Plan
	planDigest := ""
	if planPath != "" {
		planRaw, err := os.ReadFile(planPath)
		if err != nil {
			return nil, "", reviewPacketPlanMissingError()
		}
		plan, parsedDigest, err := model.ParsePlan(planRaw)
		if err != nil {
			return nil, "", reviewPacketPlanMissingError()
		}
		if parsedDigest != leasePlanDigest(core, unit) {
			return nil, "", reviewPacketPlanMissingError()
		}
		planDigest = parsedDigest
		suppliedPlan = &plan
	}
	u, err := currentPlanUnit(storePath, core, unit, suppliedPlan)
	if err != nil {
		return nil, "", reviewPacketPlanMissingError()
	}
	st.Unit = &packetUnit{PlanUnit: u}
	return suppliedPlan, planDigest, nil
}

func reviewPacketPlanMissingError() error {
	return fmt.Errorf("%s: supply plan_path for the lease issuer's accepted plan", hoststate.ReasonReviewPacketPlanMissing)
}

func leasePlanDigest(core []model.Record, unit string) string {
	d := ""
	for _, r := range core {
		if engine.IssuesLease(r, unit) {
			d = r.Approval.AcceptedPlanDigest
		}
	}
	return d
}

func leasePlanSourceRecord(core []model.Record, unit string) string {
	anchor := ""
	for _, r := range core {
		if engine.IssuesLease(r, unit) {
			anchor = r.Metadata.HeadAnchor
		}
	}
	return anchor
}
