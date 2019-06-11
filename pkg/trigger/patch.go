package trigger

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Record will be added to annotation of pod template of workload to trigger rolling update
type Record struct {
	LastUpdateTime int64    `json:"lastUpdateTime,omitempty"`
	Sources        []Source `json:"sources,omitempty"`
}

type Source struct {
	Name            string `json:"name,omitempty"`
	Namespace       string `json:"namespace,omitempty"`
	Kind            string `json:"kind,omitempty"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
}

const (
	RecordKeyPrefix = "trigger.app.example.com/"
)

func GetRecordKey(name, namespace string) string {
	return RecordKeyPrefix + namespace + "." + name
}

// escape JSON Pointer value per https://tools.ietf.org/html/rfc6901
func escapeJSONPointerValue(in string) string {
	step := strings.Replace(in, "~", "~0", -1)
	return strings.Replace(step, "/", "~1", -1)
}

// FIXME: annotations may not exist, in this case we should patch to add entire annotations field.
func generatePatch(rec *Record, key string) ([]byte, error) {
	val, err := json.Marshal(rec)
	if err != nil {
		return nil, fmt.Errorf("err encode %#v: %v", rec, err)
	}

	pt, err := json.Marshal([]interface{}{
		map[string]interface{}{
			"op":    "add",
			"path":  fmt.Sprintf("/spec/template/metadata/annotations/%s", escapeJSONPointerValue(key)),
			"value": string(val),
		},
	})

	return pt, nil
}
