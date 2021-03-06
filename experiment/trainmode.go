// Copyright (C) 2016-2018 vLife Systems Ltd <http://vlifesystems.com>
// Licensed under an MIT licence.  Please see LICENSE.md for details.

package experiment

import (
	"fmt"

	"github.com/lawrencewoodman/ddataset"
	"github.com/lawrencewoodman/dexpr"
	"github.com/vlifesystems/rhkit/aggregator"
	"github.com/vlifesystems/rhkit/assessment"
	"github.com/vlifesystems/rhkit/description"
	"github.com/vlifesystems/rhkit/goal"
	"github.com/vlifesystems/rhkit/rule"
	"github.com/vlifesystems/rulehunter/config"
	"github.com/vlifesystems/rulehunter/progress"
	"github.com/vlifesystems/rulehunter/quitter"
	"github.com/vlifesystems/rulehunter/report"
)

type TrainMode struct {
	dataset        ddataset.Dataset
	when           *dexpr.Expr
	ruleGeneration ruleGeneration
}

type ruleGenerationDesc struct {
	Fields            []string `yaml:"fields"`
	Arithmetic        bool     `yaml:"arithmetic"`
	CombinationLength int      `yaml:"combinationLength"`
}

type ruleGeneration struct {
	fields            []string
	arithmetic        bool
	combinationLength int
}

func (rg ruleGeneration) Fields() []string {
	return rg.fields
}

func (rg ruleGeneration) Arithmetic() bool {
	return rg.arithmetic
}

type trainModeDesc struct {
	Dataset *datasetDesc `yaml:"dataset"`
	// An expression that works out whether to run the experiment for this mode
	When           string             `yaml:"when"`
	RuleGeneration ruleGenerationDesc `yaml:"ruleGeneration"`
}

func newTrainMode(
	cfg *config.Config,
	desc *trainModeDesc,
	experimentFilename string,
	aggregators []aggregator.Spec,
	goals []*goal.Goal,
	sortOrder []assessment.SortOrder,
) (*TrainMode, error) {
	d, err := makeDataset(cfg, desc.Dataset)
	if err != nil {
		return nil, fmt.Errorf("dataset: %s", err)
	}
	when, err := makeWhenExpr(desc.When)
	if err != nil {
		return nil, InvalidWhenExprError(desc.When)
	}
	return &TrainMode{
		dataset: d,
		when:    when,
		ruleGeneration: ruleGeneration{
			fields:            desc.RuleGeneration.Fields,
			arithmetic:        desc.RuleGeneration.Arithmetic,
			combinationLength: desc.RuleGeneration.CombinationLength,
		},
	}, nil
}

func (m *TrainMode) Kind() report.ModeKind {
	return report.Train
}

func (m *TrainMode) Release() error {
	if m == nil {
		return nil
	}
	return m.dataset.Release()
}

func (m *TrainMode) Dataset() ddataset.Dataset {
	return m.dataset
}

func (m *TrainMode) NumAssessRulesStages() int {
	return 4 + m.ruleGeneration.combinationLength
}

func (m *TrainMode) Process(
	e *Experiment,
	cfg *config.Config,
	pm *progress.Monitor,
	q *quitter.Quitter,
	rules []rule.Rule,
) ([]rule.Rule, error) {
	reportProgress := func(msg string, percent float64) error {
		return pm.ReportProgress(e.File.Name(), report.Train, msg, percent)
	}
	quitReceived := func() bool {
		select {
		case <-q.C:
			return true
		default:
			return false
		}
	}
	noRules := []rule.Rule{}
	rt := newRuleTracker()

	if err := reportProgress("Describing train dataset", 0); err != nil {
		return noRules, err
	}

	if quitReceived() {
		return noRules, ErrQuitReceived
	}
	desc, err := description.DescribeDataset(m.Dataset())
	if err != nil {
		return noRules, fmt.Errorf("Couldn't describe train dataset: %s", err)
	}

	if quitReceived() {
		return noRules, ErrQuitReceived
	}
	rt.track(rules)
	userRules := append(rules, rule.NewTrue())
	ass, err := assessRules(e, m, 1, userRules, pm, q, cfg)
	if err != nil {
		return noRules, fmt.Errorf("Couldn't assess rules: %s", err)
	}

	assessRules := func(
		stage int,
		rules []rule.Rule,
	) (*assessment.Assessment, error) {
		newRules := rt.track(rules)
		newAss, err :=
			assessRules(e, m, stage, newRules, pm, q, cfg)
		if err != nil {
			return nil, fmt.Errorf("Couldn't assess rules: %s", err)
		}
		ass, err = ass.Merge(newAss)
		if err != nil {
			return nil, fmt.Errorf("Couldn't merge assessments: %s", err)
		}
		ass.Sort(e.SortOrder)
		ass.Refine()
		ass = ass.TruncateRuleAssessments(10000)
		return newAss, nil
	}

	if err := reportProgress("Generating rules", 0); err != nil {
		return noRules, err
	}
	generatedRules, err := rule.Generate(desc, m.ruleGeneration)
	if err != nil {
		return noRules, fmt.Errorf("Couldn't generate rules: %s", err)
	}

	if quitReceived() {
		return noRules, ErrQuitReceived
	}

	if _, err := assessRules(2, generatedRules); err != nil {
		return noRules, err
	}

	if quitReceived() {
		return noRules, ErrQuitReceived
	}

	if err := reportProgress("Tweaking rules", 0); err != nil {
		return noRules, err
	}
	tweakableRules := rule.Tweak(1, ass.Rules(), desc)

	if _, err := assessRules(3, tweakableRules); err != nil {
		return noRules, err
	}

	if quitReceived() {
		return noRules, ErrQuitReceived
	}

	if err := reportProgress("Reduce DP of rules", 0); err != nil {
		return noRules, err
	}
	reducedDPRules := rule.ReduceDP(ass.Rules())

	if _, err := assessRules(4, reducedDPRules); err != nil {
		return noRules, err
	}

	if quitReceived() {
		return noRules, ErrQuitReceived
	}

	ruleAssessments := []*assessment.RuleAssessment{ass.RuleAssessments[0]}
	for i := 0; i < m.ruleGeneration.combinationLength; i++ {
		if err := reportProgress("Combining rules", 0); err != nil {
			return noRules, err
		}
		combinedRules := rule.Combine(ass.Rules(), 10000)
		combinedAss, err := assessRules(5+i, combinedRules)
		if err != nil {
			return noRules, err
		}
		if quitReceived() {
			return noRules, ErrQuitReceived
		}
		combinedAss.Sort(e.SortOrder)
		combinedAss.Refine()

		// Add ruleAssessment for each combinationLength
		for _, ra := range combinedAss.RuleAssessments {
			if _, isTrueRule := ra.Rule.(rule.True); !isTrueRule {
				ruleAssessments = append(ruleAssessments, ra)
				break
			}
		}
	}

	// Add the true rule assessment if missing
	trueRuleAssessment := getTrueRuleAssessment(ruleAssessments)
	if trueRuleAssessment == nil {
		trueRuleAssessment = getTrueRuleAssessment(ass.RuleAssessments)
		if trueRuleAssessment == nil {
			panic("true rule missing from assessment")
		}
		ruleAssessments = append(ruleAssessments, trueRuleAssessment)
	}
	ass.RuleAssessments = ruleAssessments
	ass.Sort(e.SortOrder)
	ass.Refine()
	// TODO: Remove ruleAssessments that have longer combinationLength than
	// previous ruleAssessment?

	r := report.New(
		report.Train,
		e.Title,
		desc,
		ass,
		e.Aggregators,
		e.SortOrder,
		e.File.Name(),
		e.Tags,
		e.Category,
	)
	if err := r.WriteJSON(cfg); err != nil {
		return noRules, fmt.Errorf("Couldn't write JSON train report: %s", err)
	}
	return ass.Rules(), nil
}

func getTrueRuleAssessment(
	ruleAssessments []*assessment.RuleAssessment,
) *assessment.RuleAssessment {
	for _, ra := range ruleAssessments {
		if _, isTrueRule := ra.Rule.(rule.True); isTrueRule {
			return ra
		}
	}
	return nil
}
