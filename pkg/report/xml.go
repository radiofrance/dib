package report

import "encoding/xml"

type Testsuite struct {
	XMLName   xml.Name   `xml:"testsuite"`
	Name      string     `xml:"name,attr"`
	Errors    string     `xml:"errors,attr"`
	Tests     string     `xml:"tests,attr"`
	Failures  string     `xml:"failures,attr"`
	Skipped   string     `xml:"skipped,attr"`
	Time      string     `xml:"time,attr"`
	Timestamp string     `xml:"timestamp,attr"`
	TestCases []TestCase `xml:"testcase"`
}

type TestCase struct {
	XMLName   xml.Name `xml:"testcase"`
	ClassName string   `xml:"classname,attr"`
	File      string   `xml:"file,attr"`
	Name      string   `xml:"name,attr"`
	Time      string   `xml:"time,attr"`
	SystemOut string   `xml:"system-out"`
	Failure   string   `xml:"failure"`
}

func convertJunitReportXMLToHumanReadableFormat(rawDgossTestLogs []byte) (Testsuite, error) {
	testSuite := Testsuite{}
	err := xml.Unmarshal(rawDgossTestLogs, &testSuite)
	if err != nil {
		return testSuite, err
	}

	return testSuite, nil
}
