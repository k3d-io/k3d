package hostsfile

import "sync"

func newLookup() lookup {
	return lookup{l: make(map[string][]int)}
}

// lookup a rw mutex with a hashmap to keep track of keys (ips/hosts) and their position in the hostsfile
type lookup struct {
	sync.RWMutex
	l map[string][]int
}

func (lo *lookup) add(key string, pos int) {
	lo.Lock()
	defer lo.Unlock()
	lo.l[key] = append(lo.l[key], pos)
}

func (lo *lookup) remove(key string, pos int) {
	lo.Lock()
	defer lo.Unlock()
	// remove one entry from the lookup because we add one at a time
	lo.l[key] = removeOneFromSliceInt(pos, lo.l[key])
}

func (lo *lookup) get(key string) []int {
	lo.RLock()
	defer lo.RUnlock()
	if i, ok := lo.l[key]; ok {
		return i
	}

	return []int{}
}

func (lo *lookup) reset() {
	lo.Lock()
	defer lo.Unlock()
	lo.l = make(map[string][]int)
}
