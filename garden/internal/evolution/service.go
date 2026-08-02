package evolution

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	Provider EvolverProvider
	Store    *Store
	Events   *EventStore
	Hub      HubPolicy
}

func (s *Service) StartRun(ctx context.Context, bundle EvolutionEvidenceBundle, actor string) (EvolutionRun, error) {
	if s.Provider == nil {
		return EvolutionRun{}, ErrProviderUnavailable
	}
	if err := ValidateBundleInput(bundle); err != nil {
		return EvolutionRun{}, err
	}
	bundle.BundleID = "bundle_" + uuid.NewString()[:8]
	bundle.CreatedAt = time.Now().UTC()

	runID, err := s.Provider.StartRun(ctx, bundle)
	if err != nil {
		s.emitEvent(ctx, EvolutionEvent{Type: "run_failed", Actor: actor, Detail: "provider start failed: " + err.Error()})
		return EvolutionRun{}, err
	}

	run := EvolutionRun{
		RunID:     runID,
		Status:    "running",
		BundleID:  bundle.BundleID,
		Provider:  s.Provider.Name(),
		StartedAt: time.Now().UTC(),
	}
	_ = s.Store.SaveRun(ctx, run)
	s.emitEvent(ctx, EvolutionEvent{RunID: runID, Type: "run_started", Actor: actor, Detail: "provider=" + s.Provider.Name()})
	return run, nil
}

func (s *Service) GetRun(ctx context.Context, runID string) (EvolutionRun, error) {
	run, err := s.Store.GetRun(ctx, runID)
	if err != nil {
		return EvolutionRun{}, err
	}
	if run.Status == "running" && s.Provider != nil {
		status, pollErr := s.Provider.PollRun(ctx, runID)
		if pollErr == nil && status.Status != "running" {
			_ = s.Store.UpdateRunStatus(ctx, runID, status.Status, status.Error)
			run.Status = status.Status
			run.Error = status.Error
			if status.Status == "completed" || status.Status == "failed" {
				now := time.Now().UTC()
				run.CompletedAt = &now
			}
		}
	}
	return run, nil
}

func (s *Service) GetCandidate(ctx context.Context, candidateID string) (GeneCandidate, error) {
	return s.Store.GetCandidate(ctx, candidateID)
}

func (s *Service) CreateProposal(ctx context.Context, runID, candidateID, actor string) (EvolutionProposal, error) {
	candidate, err := s.Store.GetCandidate(ctx, candidateID)
	if err != nil {
		return EvolutionProposal{}, err
	}

	run, err := s.Store.GetRun(ctx, runID)
	if err != nil {
		return EvolutionProposal{}, err
	}

	bundle := EvolutionEvidenceBundle{BundleID: run.BundleID}
	report, normErr := NormalizeCandidate(candidate, bundle)

	proposal := EvolutionProposal{
		ProposalID:    "prop_" + uuid.NewString()[:8],
		RunID:         runID,
		CandidateID:   candidateID,
		Kind:          candidate.Kind,
		Status:        "pending",
		Summary:       candidate.Name + ": " + candidate.Description,
		LeakageReport: report,
		CreatedAt:     time.Now().UTC(),
	}
	if normErr != nil {
		proposal.Status = "rejected"
		proposal.ReviewNote = "automatic rejection: " + normErr.Error()
	}

	_ = s.Store.SaveProposal(ctx, proposal)
	s.emitEvent(ctx, EvolutionEvent{RunID: runID, ProposalID: proposal.ProposalID, Type: "proposal_created", Actor: actor, Detail: "kind=" + candidate.Kind})
	return proposal, nil
}

func (s *Service) GetProposal(ctx context.Context, proposalID string) (EvolutionProposal, error) {
	return s.Store.GetProposal(ctx, proposalID)
}

func (s *Service) ReviewProposal(ctx context.Context, proposalID, decision, reviewer, note string) (EvolutionProposal, error) {
	proposal, err := s.Store.GetProposal(ctx, proposalID)
	if err != nil {
		return EvolutionProposal{}, err
	}
	if decision != "approved" && decision != "rejected" {
		return EvolutionProposal{}, ErrInvalidDecision
	}

	_ = s.Store.UpdateProposalReview(ctx, proposalID, decision, reviewer, note)
	proposal.Status = decision
	proposal.Reviewer = reviewer
	proposal.ReviewNote = note
	now := time.Now().UTC()
	proposal.ReviewedAt = &now

	eventType := "proposal_approved"
	if decision == "rejected" {
		eventType = "proposal_rejected"
	}
	s.emitEvent(ctx, EvolutionEvent{RunID: proposal.RunID, ProposalID: proposalID, Type: eventType, Actor: reviewer, Detail: note})
	return proposal, nil
}

func (s *Service) GetEvent(ctx context.Context, eventID string) (EvolutionEvent, error) {
	return s.Events.Get(ctx, eventID)
}

func (s *Service) emitEvent(ctx context.Context, ev EvolutionEvent) {
	if s.Events != nil {
		_ = s.Events.Append(ctx, ev)
	}
}
