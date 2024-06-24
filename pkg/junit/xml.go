package junit

import (
	"encoding/xml"
)

type Testsuite struct {
	XMLName   xml.Name   `json:"-"                   xml:"testsuite"`
	Name      string     `json:"name,omitempty"      xml:"name,attr"`
	Errors    string     `json:"errors,omitempty"    xml:"errors,attr"`
	Tests     string     `json:"tests,omitempty"     xml:"tests,attr"`
	Failures  string     `json:"failures,omitempty"  xml:"failures,attr"`
	Skipped   string     `json:"skipped,omitempty"   xml:"skipped,attr"`
	Time      string     `json:"time,omitempty"      xml:"time,attr"`
	Timestamp string     `json:"timestamp,omitempty" xml:"timestamp,attr"`
	TestCases []TestCase `json:"testcases,omitempty" xml:"testcase"`
}

type TestCase struct {
	XMLName   xml.Name `json:"-"                    xml:"testcase"`
	ClassName string   `json:"class_name,omitempty" xml:"classname,attr"`
	File      string   `json:"file,omitempty"       xml:"file,attr"`
	Name      string   `json:"name,omitempty"       xml:"name,attr"`
	Time      string   `json:"time,omitempty"       xml:"time,attr"`
	SystemOut string   `json:"system_out,omitempty" xml:"system-out"`
	Failure   string   `json:"failure,omitempty"    xml:"failure"`
}

// ParseRawLogs cast a raw XML JunitReport (as byte) into a Testsuite structure.
func ParseRawLogs(testsuiteData []byte) (Testsuite, error) {
	testSuite := Testsuite{}
	err := xml.Unmarshal(testsuiteData, &testSuite)
	if err != nil {
		return testSuite, err
	}

	return testSuite, nil
}
