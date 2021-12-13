package metahelm

import (
	"strings"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/strvals"
)

// ValueOverridesMap represents a set of chart YAML overrides, a map of YAML path to value,
// in the same format as accepted by the helm CLI: "foo.bar=value" except in a map.
type ValueOverridesMap map[string]string

// ToYAMLStream takes overrides and serializes into a raw YAML stream that is assigned to c.ValueOverrides
func (c *Chart) ToYAMLStream(overrides ValueOverridesMap) error {
	sl := []string{}
	for k, v := range overrides {
		sl = append(sl, k+"="+v)
	}
	vo, err := strvals.ToYAML(strings.Join(sl, ","))
	if err != nil {
		return errors.Wrap(err, "error encoding values to YAML")
	}
	c.ValueOverrides = []byte(vo)
	return nil
}
