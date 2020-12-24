package helm

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const indexYaml = `
apiVersion: v1
entries:
  testchart:
  - apiVersion: v2
    appVersion: 1.10.13
    created: "2020-11-27T18:10:15.83673504Z"
    name: testchart
    version: 1.1.1
  - apiVersion: v2
    appVersion: 1.10.13
    created: "2020-11-27T18:10:15.83673504Z"
    name: testchart
    version: 2.1.1
  - apiVersion: v2
    appVersion: 1.10.13
    created: "2020-11-27T18:10:15.83673504Z"
    name: testchart
    version: 0.0.1
generated: "2020-12-18T22:41:15.405046581Z"
`

func createServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/index.yaml" {
			fmt.Fprintln(w, indexYaml)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestMissingChart(t *testing.T) {

	ts := createServer()
	defer ts.Close()

	// fetch non-existing chart
	cfg := fmt.Sprintf("chart: nosuch\nrepository: %s", ts.URL)
	h, err := New("", cfg)
	assert.NoError(t, err)
	assert.NoError(t, h.Init())

	_, err = h.Probe("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "was not found")
}

func TestExistingChart(t *testing.T) {

	ts := createServer()
	defer ts.Close()

	// fetch non-existing chart
	cfg := fmt.Sprintf("chart: testchart\nrepository: %s", ts.URL)
	h, e := New("", cfg)
	assert.NoError(t, e)
	assert.NoError(t, h.Init())

	var v string
	var err error

	v, err = h.Probe("")
	assert.NoError(t, err)
	assert.Equal(t, "2.1.1", v, "should return latest version if no versions specified")

	v, err = h.Probe("2.0.0")
	assert.NoError(t, err)
	assert.Equal(t, "2.1.1", v, "should return latest version compared to specified")

	v, err = h.Probe("100.0.0")
	assert.NoError(t, err)
	assert.Equal(t, "", v, "should return empty version if current one greater than existing")

	v, err = h.Probe("2.1.1")
	assert.NoError(t, err)
	assert.Equal(t, "", v, "should return empty version if current one is equal to existing")

	cfg2 := fmt.Sprintf("chart: testchart\nrepository: %s\nconstraint: 1.*.*", ts.URL)
	h2, e2 := New("", cfg2)
	assert.NoError(t, e2)
	assert.NoError(t, h2.Init())
	v, err = h2.Probe("1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, "1.1.1", v, "should respect specified constraint")

}
