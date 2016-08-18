package porto

import (
	"encoding/json"
	"io"
	"sort"
	"sync"

	"github.com/pborman/uuid"
)

type layersMap map[string]string

type journal struct {
	mu     sync.RWMutex
	UUID   string    `json:"uuid"`
	Layers layersMap `json:"layers"`
}

func newJournal() *journal {
	j := &journal{
		UUID:   uuid.New(),
		Layers: make(layersMap),
	}
	return j
}

func (j *journal) Dump(w io.Writer) error {
	j.mu.RLock()
	defer j.mu.RUnlock()
	enc := json.NewEncoder(w)
	return enc.Encode(j)
}

func (j *journal) Load(r io.Reader) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := json.NewDecoder(r).Decode(j); err != nil {
		return err
	}
	return nil
}

func (j *journal) Insert(layer string, digest string) *journal {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Layers[layer] = digest
	return j
}

func (j *journal) In(layer string, digest string) bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	v, ok := j.Layers[layer]
	return ok && v == digest
}

func (j *journal) UpdateFromPorto(layers []string) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if len(layers) == 0 {
		j.Layers = make(layersMap)
		return
	}

	if !sort.StringsAreSorted(layers) {
		sort.Strings(layers)
	}
	for k := range j.Layers {
		if !in(layers, k) {
			delete(j.Layers, k)
		}
	}
}

func (j *journal) String() string {
	j.mu.RLock()
	defer j.mu.RUnlock()
	body, err := json.Marshal(j)
	if err != nil {
		return err.Error()
	}
	return string(body)
}

func in(a []string, x string) bool {
	i := sort.SearchStrings(a, x)
	return i < len(a) && a[i] == x
}
