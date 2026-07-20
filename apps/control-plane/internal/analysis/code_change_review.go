package analysis

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

const minimumReviewConfidence = 0.80

type declaredRequirement struct {
	ID                    string   `json:"id"`
	Text                  string   `json:"text"`
	Critical              bool     `json:"critical"`
	RequiredEvidenceTypes []string `json:"required_evidence_types"`
}

type requirementsPayload struct {
	Request      string                `json:"request"`
	Requirements []declaredRequirement `json:"requirements"`
}

type requirementAssessment struct {
	RequirementID string   `json:"requirement_id"`
	Status        string   `json:"status"`
	Confidence    float64  `json:"confidence"`
	Rationale     string   `json:"rationale"`
	EvidenceRefs  []string `json:"evidence_refs"`
	Reviewer      string   `json:"reviewer"`
}

type changeArtifact struct {
	ID string `json:"id"`
}

type reviewObservation struct {
	assessment requirementAssessment
	evidence   domain.Evidence
}

// analyzeCodeChangeReview turns a written request into one independently
// inspectable claim per acceptance requirement. Model confidence is recorded,
// but cannot support a critical claim without an explicit evidence reference.
func analyzeCodeChangeReview(run domain.Run, decision domain.Decision) (domain.Analysis, error) {
	evidence := evidenceFromEvents(run)
	var declaration requirementsPayload
	declarationEvidence := domain.Evidence{}
	assessments := map[string][]reviewObservation{}
	artifacts := map[string]domain.Evidence{}

	for index, event := range run.Events {
		switch strings.ToLower(event.Type) {
		case "requirements.declared":
			if err := json.Unmarshal(event.Payload, &declaration); err != nil {
				return domain.Analysis{}, fmt.Errorf("decode requirements.declared: %w", err)
			}
			declarationEvidence = evidence[index]
		case "requirement.assessed":
			var assessment requirementAssessment
			if err := json.Unmarshal(event.Payload, &assessment); err != nil {
				return domain.Analysis{}, fmt.Errorf("decode requirement.assessed: %w", err)
			}
			assessments[assessment.RequirementID] = append(assessments[assessment.RequirementID], reviewObservation{assessment: assessment, evidence: evidence[index]})
		case "change.artifact.observed":
			var artifact changeArtifact
			if err := json.Unmarshal(event.Payload, &artifact); err != nil {
				return domain.Analysis{}, fmt.Errorf("decode change.artifact.observed: %w", err)
			}
			if strings.TrimSpace(artifact.ID) != "" {
				artifacts[artifact.ID] = evidence[index]
			}
		}
	}

	result := domain.Analysis{Evidence: evidence}
	for index, requirement := range declaration.Requirements {
		if strings.TrimSpace(requirement.ID) == "" || strings.TrimSpace(requirement.Text) == "" {
			return domain.Analysis{}, fmt.Errorf("requirement %d must contain id and text", index+1)
		}
		requiredTypes := requirement.RequiredEvidenceTypes
		if len(requiredTypes) == 0 {
			requiredTypes = []string{"code_diff", "test_result"}
		}
		observations := assessments[requirement.ID]
		referencedArtifacts := []domain.Evidence{}
		for _, observation := range observations {
			for _, reference := range observation.assessment.EvidenceRefs {
				if artifact, ok := artifacts[reference]; ok {
					referencedArtifacts = append(referencedArtifacts, artifact)
				}
			}
		}
		referencedArtifacts = dedupeEvidence(referencedArtifacts)
		state, justification := reviewState(observations, referencedArtifacts, requiredTypes)
		claimID := fmt.Sprintf("change_requirement_%d", index+1)
		bound := []domain.Evidence{declarationEvidence}
		for _, observation := range observations {
			bound = append(bound, observation.evidence)
		}
		bound = append(bound, referencedArtifacts...)
		bound = dedupeEvidence(bound)
		claim := claim(claimID, requirement.Text, "request requirement "+requirement.ID, requirement.Critical, requiredTypes, state, justification)
		claim.Support = supportFor(bound, state)
		for _, item := range bound {
			claim.EvidenceIDs = append(claim.EvidenceIDs, item.ID)
			relation := domain.RelationDerivedFrom
			if item.ID != declarationEvidence.ID {
				switch state {
				case domain.ClaimContradicted:
					relation = domain.RelationContradicts
				case domain.ClaimPartiallySupported, domain.ClaimVerificationPending:
					relation = domain.RelationQualifies
				default:
					relation = domain.RelationSupports
				}
			}
			result.Relations = append(result.Relations, domain.Relation{FromID: item.ID, ToID: claimID, Type: relation, Rationale: justification})
		}
		result.Claims = append(result.Claims, claim)

		if requirement.Critical && state == domain.ClaimVerificationPending {
			result.Assumptions = append(result.Assumptions, domain.Assumption{ClaimID: claimID, Statement: "The PR is assumed to cover requirement " + requirement.ID + ", but direct evidence is missing or below the confidence gate.", Critical: true, Status: "open", Impact: "blocks_merge"})
			result.Unknowns = append(result.Unknowns, domain.Unknown{Question: "What direct PR change or test verifies requirement " + requirement.ID + ": " + requirement.Text, Critical: true})
		}
	}

	return result, nil
}

func reviewState(observations []reviewObservation, referencedArtifacts []domain.Evidence, requiredTypes []string) (domain.ClaimState, string) {
	if len(observations) == 0 {
		return domain.ClaimVerificationPending, "No reviewer assessment is bound to this requirement."
	}
	best := observations[len(observations)-1].assessment
	for _, observation := range observations {
		status := strings.ToLower(observation.assessment.Status)
		if status == "failed" || status == "contradicted" {
			best = observation.assessment
			return domain.ClaimContradicted, reviewJustification(best, "The submitted change contradicts this requirement")
		}
	}
	switch strings.ToLower(best.Status) {
	case "passed", "covered":
		if best.Confidence < minimumReviewConfidence || len(referencedArtifacts) == 0 {
			return domain.ClaimVerificationPending, reviewJustification(best, "The reviewer reported coverage, but the confidence or evidence-reference gate was not met")
		}
		if missing := missingEvidenceTypes(referencedArtifacts, requiredTypes); len(missing) > 0 {
			return domain.ClaimPartiallySupported, reviewJustification(best, "The reviewer reported coverage, but direct evidence is missing for: "+strings.Join(missing, ", "))
		}
		return domain.ClaimSupported, reviewJustification(best, "The reviewer found direct coverage")
	case "partial", "partially_covered":
		return domain.ClaimPartiallySupported, reviewJustification(best, "The reviewer found only partial coverage")
	default:
		return domain.ClaimVerificationPending, reviewJustification(best, "The reviewer did not provide a decisive supported result")
	}
}

func missingEvidenceTypes(artifacts []domain.Evidence, required []string) []string {
	present := map[string]bool{}
	for _, artifact := range artifacts {
		present[artifact.Kind] = true
	}
	missing := []string{}
	for _, kind := range required {
		if !present[kind] {
			missing = append(missing, kind)
		}
	}
	return missing
}

func reviewJustification(assessment requirementAssessment, prefix string) string {
	reviewer := assessment.Reviewer
	if strings.TrimSpace(reviewer) == "" {
		reviewer = "unspecified reviewer"
	}
	detail := strings.TrimSpace(assessment.Rationale)
	if detail == "" {
		detail = "No rationale supplied."
	}
	return fmt.Sprintf("%s (%s, confidence %.0f%%). %s", prefix, reviewer, assessment.Confidence*100, detail)
}
