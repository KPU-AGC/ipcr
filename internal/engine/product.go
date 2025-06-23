package engine

import "ipcress-go/internal/primer"

type Product struct {
	ExperimentID string `json:"experiment_id"`
	SequenceID   string `json:"sequence_id"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
	Length       int    `json:"length"`
	Type         string `json:"type"`
	FwdMatch     primer.Match `json:"-"`
	RevMatch     primer.Match `json:"-"`
	Seq          string `json:"seq,omitempty"`
}
