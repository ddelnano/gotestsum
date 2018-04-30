/*Package junitxml creates a JUnit XML report from a testjson.Execution.
 */
package junitxml

import (
	"encoding/xml"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"gotest.tools/gotestsum/testjson"
)

// JUnitTestSuites is a collection of JUnit test suites.
type JUnitTestSuites struct {
	XMLName xml.Name `xml:"testsuites"`
	Suites  []JUnitTestSuite
}

// JUnitTestSuite is a single JUnit test suite which may contain many
// testcases.
type JUnitTestSuite struct {
	XMLName    xml.Name        `xml:"testsuite"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	Time       string          `xml:"time,attr"`
	Name       string          `xml:"name,attr"`
	Properties []JUnitProperty `xml:"properties>property,omitempty"`
	TestCases  []JUnitTestCase
}

// JUnitTestCase is a single test case with its result.
type JUnitTestCase struct {
	XMLName     xml.Name          `xml:"testcase"`
	Classname   string            `xml:"classname,attr"`
	Name        string            `xml:"name,attr"`
	Time        string            `xml:"time,attr"`
	SkipMessage *JUnitSkipMessage `xml:"skipped,omitempty"`
	Failure     *JUnitFailure     `xml:"failure,omitempty"`
}

// JUnitSkipMessage contains the reason why a testcase was skipped.
type JUnitSkipMessage struct {
	Message string `xml:"message,attr"`
}

// JUnitProperty represents a key/value pair used to define properties.
type JUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// JUnitFailure contains data related to a failed test.
type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

// Write creates an XML document and writes it to out.
func Write(out io.Writer, exec *testjson.Execution) error {
	return errors.Wrap(write(out, generate(exec)), "failed to write JUnit XML")
}

func generate(exec *testjson.Execution) JUnitTestSuites {
	suites := JUnitTestSuites{}
	for _, pkgname := range exec.Packages() {
		pkg := exec.Package(pkgname)
		junitpkg := JUnitTestSuite{
			Name:       pkgname,
			Tests:      pkg.Total,
			Time:       testjson.FormatDurationAsSeconds(pkg.Elapsed(), 3),
			Properties: packageProperties(),
			TestCases:  packageTestCases(pkg),
			Failures:   len(pkg.Failed),
		}
		suites.Suites = append(suites.Suites, junitpkg)
	}
	return suites
}

func packageProperties() []JUnitProperty {
	return []JUnitProperty{
		{Name: "go.version", Value: runtime.Version()},
	}
}

func packageTestCases(pkg *testjson.Package) []JUnitTestCase {
	cases := []JUnitTestCase{}

	for _, tc := range pkg.Failed {
		jtc := newJUnitTestCase(tc)
		jtc.Failure = &JUnitFailure{
			Message:  "Failed",
			Contents: strings.Join(pkg.Output(tc.Test), ""),
		}
		cases = append(cases, jtc)
	}

	for _, tc := range pkg.Skipped {
		jtc := newJUnitTestCase(tc)
		jtc.SkipMessage = &JUnitSkipMessage{Message: strings.Join(pkg.Output(tc.Test), "")}
		cases = append(cases, jtc)
	}

	for _, tc := range pkg.Passed {
		jtc := newJUnitTestCase(tc)
		cases = append(cases, jtc)
	}
	return cases
}

func newJUnitTestCase(tc testjson.TestCase) JUnitTestCase {
	return JUnitTestCase{
		Classname: filepath.Base(tc.Package),
		Name:      tc.Test,
		Time:      testjson.FormatDurationAsSeconds(tc.Elapsed, 3),
	}
}

func write(out io.Writer, suites JUnitTestSuites) error {
	doc, err := xml.MarshalIndent(suites, "", "\t")
	if err != nil {
		return err
	}
	_, err = out.Write([]byte(xml.Header))
	if err != nil {
		return err
	}
	_, err = out.Write(doc)
	return err
}
