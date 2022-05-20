package main

import (
	"encoding/json"

	"github.com/bitrise-steplib/steps-cache-push/model"
)

//type archiveInfo struct {
//	Version      uint64 `json:"version,omitempty"`
//	StackID      string `json:"stack_id,omitempty"`
//	Architecture string `json:"architecture,omitempty"`
//}
//
//func (a archiveInfo) String() string {
//	return fmt.Sprintf("%s (%s)", a.StackID, a.Architecture)
//}

// parseArchiveInfo reads the stack id and architecture from the given json bytes.
func parseArchiveInfo(b []byte) (info model.ArchiveInfo, err error) {
	err = json.Unmarshal(b, &info)
	return
}
