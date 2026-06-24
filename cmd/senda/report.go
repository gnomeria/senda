// Run reporters: render []RunResult as a machine-readable report (JSON or
// JUnit XML) for CI pipelines, instead of the human text status lines.
package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"

	"senda/internal/model"
)

// renderReport returns the report bytes for the given format ("json" or
// "junit"). Unknown formats return an error.
func renderReport(format string, results []model.RunResult) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(results, "", "  ")
	case "junit":
		return renderJUnit(results), nil
	default:
		return nil, fmt.Errorf("unknown report format %q (want json or junit)", format)
	}
}

type junitSuites struct {
	XMLName  xml.Name     `xml:"testsuites"`
	Tests    int          `xml:"tests,attr"`
	Failures int          `xml:"failures,attr"`
	Suites   []junitSuite `xml:"testsuite"`
}

type junitSuite struct {
	Name      string      `xml:"name,attr"`
	Tests     int         `xml:"tests,attr"`
	Failures  int         `xml:"failures,attr"`
	TestCases []junitCase `xml:"testcase"`
}

type junitCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

func renderJUnit(results []model.RunResult) []byte {
	failures := 0
	cases := make([]junitCase, 0, len(results))
	for _, r := range results {
		c := junitCase{
			Name:      r.Name,
			ClassName: r.Path,
			Time:      float64(r.DurationMs) / 1000,
		}
		if !r.OK {
			failures++
			msg := r.Error
			if msg == "" {
				msg = fmt.Sprintf("%s %s %d, asserts %d/%d", r.Method, r.URL, r.Status, r.AssertPass, r.AssertPass+r.AssertFail)
			}
			c.Failure = &junitFailure{Message: msg, Body: msg}
		}
		cases = append(cases, c)
	}
	doc := junitSuites{
		Tests:    len(results),
		Failures: failures,
		Suites: []junitSuite{{
			Name:      "senda",
			Tests:     len(results),
			Failures:  failures,
			TestCases: cases,
		}},
	}
	out, _ := xml.MarshalIndent(doc, "", "  ")
	return append([]byte(xml.Header), out...)
}
