package pkg

import (
	"errors"
	"sort"
	"strings"
)

var ErrInvalidPayload = errors.New("invalid JSON payload")
var ErrNoSuchRef = errors.New("no such $ref")

type eid = uint32
type sid = uint32
type sids []sid
type schemaJSON = map[string]interface{}
type schemasJSON = map[string]schemaJSON

func (f sids) Len() int           { return len(f) }
func (f sids) Less(i, j int) bool { return f[i] < f[j] }
func (f sids) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }

func (vald *Validator) refsFromSIDs(SIDs sids) []string {
	sort.Sort(SIDs)
	refs := make([]string, 0, len(SIDs))
	for _, SID := range SIDs {
		schemaPtr := vald.Spec.Schemas.Json[SID].GetPtr()
		if ref := schemaPtr.GetRef(); ref != "" {
			ref = strings.TrimPrefix(ref, oa3ComponentsSchemas)
			refs = append(refs, ref)
		}
	}
	if len(refs) == 0 {
		return []string{"_"}
	}
	return refs
}
