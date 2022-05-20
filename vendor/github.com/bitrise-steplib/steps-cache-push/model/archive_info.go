package model

import "fmt"

type ArchiveInfo struct {
	Version      uint64 `json:"version,omitempty"`
	StackID      string `json:"stack_id,omitempty"`
	Architecture string `json:"architecture,omitempty"`
}

func (a ArchiveInfo) String() string {
	return fmt.Sprintf("%s (%s)", a.StackID, a.Architecture)
}
