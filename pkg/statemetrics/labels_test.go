package statemetrics

import (
	"strings"
	"testing"
	"testing/quick"

	"github.com/prometheus/common/model"
)

const reservedLabelPrefix = "__"

func TestLabelsTransformation(t *testing.T) {
	f := func(labels map[string]string) bool {
		newLabels := kubernetesLabelsToPrometheusLabels(labels)

		for k := range newLabels {
			if !checkLabelName(k) {
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func checkLabelName(l string) bool {
	return model.LabelName(l).IsValid() && !strings.HasPrefix(l, reservedLabelPrefix)
}
