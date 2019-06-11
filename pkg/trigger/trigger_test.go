package trigger

import (
	"encoding/json"
	"testing"
)

func TestGeneratePatch(t *testing.T) {
	rec := &Record{
		LastUpdateTime: 1560210953900081130,
		Sources: []Source{
			{
				Name:            "foo",
				Namespace:       "foo-ns",
				Kind:            "ConfigMap",
				ResourceVersion: "1",
			},
		},
	}
	key := GetRecordKey("foo", "foo-ns")
	pt, err := generatePatch(rec, key)
	if err != nil {
		t.Error(err)
	}

	any := AnySlice{}
	if err := json.Unmarshal(pt, &any); err != nil {
		t.Error(err)
	}
	if len(any) != 1 {
		t.Fatal("Expect have length 1")
	}
	if len(any[0]) != 3 {
		t.Fatal("Expect have length 3")
	}
}

type AnySlice []Any
type Any map[string]interface{}
