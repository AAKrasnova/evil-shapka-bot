package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readFile(path string, dest any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewDecoder(f).Decode(&dest)
}

func Test_validateCMS(t *testing.T) {
	cms := map[string]map[string]string{}
	err := readFile("./cms.json", &cms)
	require.NoError(t, err)

	lens := make(map[int][]string)
	for loc, texts := range cms {
		lens[len(texts)] = append(lens[len(texts)], loc)
	}
	if len(lens) > 1 {
		t.Errorf("not equal texts count: %v", lens)
	}

	data, err := json.Marshal(texts{})
	require.NoError(t, err)

	ids := make(map[string]string)
	err = json.Unmarshal(data, &ids)
	require.NoError(t, err)

	for loc, texts := range cms {
		for id := range ids {
			_, ok := texts[id]
			assert.True(t, ok, "missing text %s for locale %s", id, loc)
		}
	}
}

func Test_cmdReload(t *testing.T) {
	copyFile := func(src, dst string) error {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, os.ModePerm)
	}
	const testFile = "testdata/cms.json"

	localies := newLocalies()
	err := localies.load(testFile)
	require.NoError(t, err)
	err = localies.initWatcher(testFile)
	require.NoError(t, err)

	_, ok := localies.cms["cn"] // it's ok to check in test without mutex
	require.False(t, ok)

	// add cn locale
	const (
		updatedTestFile = "testdata/updated-cms.json"
		bkTestFile      = "testdata/cms.bk.json"
	)
	err = copyFile(testFile, bkTestFile)
	require.NoError(t, err)
	err = copyFile(updatedTestFile, testFile)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // wait for watcher to reload cms

	_, ok = localies.cms["cn"] // it's ok to check in test without mutex
	require.True(t, ok)

	// remove cn locale
	err = copyFile(bkTestFile, testFile)
	require.NoError(t, err)
	err = os.Remove(bkTestFile)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // wait for watcher to reload cms

	_, ok = localies.cms["cn"] // it's ok to check in test without mutex
	require.False(t, ok)
}
