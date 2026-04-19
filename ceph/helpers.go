package ceph

// sliceDiff computes elements to add (in desired but not existing) and
// elements to remove (in existing but not desired).
func sliceDiff(existing, desired []string) (toAdd, toRemove []string) {
	existingSet := make(map[string]bool, len(existing))
	for _, s := range existing {
		existingSet[s] = true
	}
	desiredSet := make(map[string]bool, len(desired))
	for _, s := range desired {
		desiredSet[s] = true
	}
	for _, s := range desired {
		if !existingSet[s] {
			toAdd = append(toAdd, s)
		}
	}
	for _, s := range existing {
		if !desiredSet[s] {
			toRemove = append(toRemove, s)
		}
	}
	return
}
