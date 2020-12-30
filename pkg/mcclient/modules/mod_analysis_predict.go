package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	AnalysisPredictManager *SAnalysisPredict
)

func init() {
	AnalysisPredictManager = NewAnalysisPredict()
	Register(AnalysisPredictManager)
}

type SAnalysisPredict struct {
	*modulebase.ResourceManager
}

func NewAnalysisPredict() *SAnalysisPredict {
	man := NewSuggestionManager("analysispredict", "analysispredicts", []string{}, []string{})
	return &SAnalysisPredict{
		ResourceManager: &man,
	}
}
