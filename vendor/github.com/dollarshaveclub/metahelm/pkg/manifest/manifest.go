/*
Package manifest is (modified) copypasta from https://github.com/helm/helm/tree/master/pkg/manifest
The Helm manifest package was broken out from pkg/tiller for 2.11.0, but in order to stay on an earlier
Helm version (which allows us to keep k8s 1.10.x and related dependencies) we copy it here.

This copied code is licensed under the Apache 2.0 license (see LICENSE in this directory)
*/
package manifest

import (
	"log"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/releaseutil"
)

// K8sManifest models the k8s object metadata we care about
type K8sManifest struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
}

// SplitManifests takes a map of rendered templates and splits them into the
// detected manifests.
func SplitManifests(templates map[string]string) []releaseutil.Manifest {
	var listManifests []releaseutil.Manifest
	// extract kind and name
	re := regexp.MustCompile("kind:(.*)\n")
	for k, v := range templates {
		match := re.FindStringSubmatch(v)
		h := "Unknown"
		if len(match) == 2 {
			h = strings.TrimSpace(match[1])
		}
		// we have to parse the yaml to get the metadata name
		km := K8sManifest{}
		if err := yaml.Unmarshal([]byte(v), &km); err != nil {
			log.Printf("error unmarshaling template: %v; ignoring", err)
			continue
		}
		m := releaseutil.Manifest{
			Name:    k,
			Content: v,
			Head: &releaseutil.SimpleHead{
				Kind: h,
				Metadata: &struct {
					Name        string            `json:"name"`
					Annotations map[string]string `json:"annotations"`
				}{
					Name: km.Metadata.Name,
				},
			},
		}
		listManifests = append(listManifests, m)
	}
	return listManifests
}
