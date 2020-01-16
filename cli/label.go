package run

import (
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

// mapNodesToLabelSpecs maps nodes to labelSpecs
func mapNodesToLabelSpecs(specs []string, createdNodes []string) (map[string][]string, error) {
	// check node-specifier possibilitites
	possibleNodeSpecifiers := []string{"all", "workers", "agents", "server", "master"}
	possibleNodeSpecifiers = append(possibleNodeSpecifiers, createdNodes...)

	nodeToLabelSpecMap := make(map[string][]string)

	for _, spec := range specs {
		labelSpec, node := extractLabelNode(spec)

		// check if node-specifier is valid (either a role or a name) and append to list if matches
		nodeFound := false
		for _, name := range possibleNodeSpecifiers {
			if node == name {
				nodeFound = true
				nodeToLabelSpecMap[node] = append(nodeToLabelSpecMap[node], labelSpec)
				break
			}
		}

		// node extraction was a false positive, use full spec with default node
		if !nodeFound {
			nodeToLabelSpecMap[defaultLabelNodes] = append(nodeToLabelSpecMap[defaultLabelNodes], spec)
		}
	}

	return nodeToLabelSpecMap, nil
}

// extractLabelNode separates the node specification from the actual label specs
func extractLabelNode(spec string) (string, string) {
	// label defaults to full spec
	labelSpec := spec

	// node defaults to "all"
	node := defaultLabelNodes

	// only split at the last "@"
	re := regexp.MustCompile(`^(.*)@([^@]+)$`)
	match := re.FindStringSubmatch(spec)

	if len(match) > 0 {
		labelSpec = match[1]
		node = match[2]
	}

	return labelSpec, node
}

// splitLabel separates the label key from the label value
func splitLabel(label string) (string, string) {
	// split only on first '=' sign (like `docker run` do)
	labelSlice := strings.SplitN(label, "=", 2)

	if len(labelSlice) > 1 {
		return labelSlice[0], labelSlice[1]
	}

	// defaults to label key with empty value (like `docker run` do)
	return label, ""
}

// MergeLabelSpecs merges labels for a given node
func MergeLabelSpecs(nodeToLabelSpecMap map[string][]string, role, name string) ([]string, error) {
	labelSpecs := []string{}

	// add portSpecs according to node role
	for _, group := range nodeRuleGroupsMap[role] {
		for _, v := range nodeToLabelSpecMap[group] {
			exists := false
			for _, i := range labelSpecs {
				if v == i {
					exists = true
				}
			}
			if !exists {
				labelSpecs = append(labelSpecs, v)
			}
		}
	}

	// add portSpecs according to node name
	for _, v := range nodeToLabelSpecMap[name] {
		exists := false
		for _, i := range labelSpecs {
			if v == i {
				exists = true
			}
		}
		if !exists {
			labelSpecs = append(labelSpecs, v)
		}
	}

	return labelSpecs, nil
}

// MergeLabels merges list of labels into a label map
func MergeLabels(labelMap map[string]string, labels []string) map[string]string {
	for _, label := range labels {
		labelKey, labelValue := splitLabel(label)

		if _, found := labelMap[labelKey]; found {
			log.Warningf("Overriding already existing label [%s]", labelKey)
		}

		labelMap[labelKey] = labelValue
	}

	return labelMap
}
