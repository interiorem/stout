package porto

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJournalDumpLoad(t *testing.T) {
	assertT := require.New(t)

	var fixture = struct {
		UUID   string            `json:"uuid"`
		Layers map[string]string `json:"layers"`
	}{
		UUID: "uuid",
		Layers: map[string]string{
			"C": "abc",
			"D": "abc",
			"A": "abc",
			"E": "abc"},
	}

	fixtureBody, _ := json.Marshal(fixture)

	j := newJournal()
	assertT.Empty(j.Layers)

	assertT.NoError(j.Load(bytes.NewReader(fixtureBody)))
	assertT.Equal(fixture.UUID, j.UUID)

	// Dump and Load mechanism
	var buff = new(bytes.Buffer)
	assertT.NoError(j.Dump(buff))
	dumpedJ := newJournal()
	dumpedJ.Load(buff)
	assertT.Equal(j, dumpedJ)

	// test two different branches
	dumpedJ.UpdateFromPorto([]string{"B", "A", "E"})
	assertT.EqualValues(map[string]string{"A": "abc", "E": "abc"}, dumpedJ.Layers)
	dumpedJ.UpdateFromPorto([]string{})
	assertT.Empty(dumpedJ.Layers)
	dumpedJ.UpdateFromPorto([]string{})
	assertT.Empty(dumpedJ.Layers)
}

func TestJournalInsert(t *testing.T) {
	assertT := require.New(t)
	j := newJournal()
	_ = j.String()

	assertT.Empty(j.Layers)
	assertT.False(j.In("C", "c"))

	j.Insert("D", "d").Insert("B", "b").Insert("A", "a")
	assertT.True(j.In("B", "b"))
	assertT.False(j.In("B", "a"))
	assertT.False(j.In("C", "c"))
	j.Insert("C", "c")
	assertT.True(j.In("C", "c"))

	assertT.EqualValues(map[string]string{"A": "a", "B": "b", "C": "c", "D": "d"}, j.Layers)
}
