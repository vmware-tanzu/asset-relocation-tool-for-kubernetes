package goerkin

import (
	"fmt"
	"os"
	"reflect"
	"regexp"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type Steps struct {
	definitions definitions
	used        definitions

	Fail func(message string, callerSkip ...int)
}

func (s *Steps) UnusedSteps() []string {
	var unused []string

	for stepRE := range s.definitions {
		if _, used := s.used[stepRE]; !used {
			unused = append(unused, stepRE.String())
		}
	}

	return unused
}

type defineBodyFn func(Definitions)
type bodyFn func()

func NewSteps() *Steps {
	steps := &Steps{
		definitions: definitions{},
		used:        definitions{},
		Fail:        ginkgo.Fail,
	}

	if _, set := os.LookupEnv("UNUSED_FAIL"); set {
		ginkgo.AfterEach(func() {
			Expect(steps.UnusedSteps()).To(BeEmpty(), "This array represents unused step definitions")
		})
	}

	return steps
}

func Define(body ...interface{}) *Steps {
	steps := NewSteps()

	if len(body) > 0 {
		steps.Define(body[0].(func(Definitions)))
	}

	return steps
}

func (s *Steps) Define(bodies ...defineBodyFn) {
	for _, body := range bodies {
		body(s.definitions)
	}
}

type matchT struct {
	body   interface{}
	params []string
	re     *regexp.Regexp
}

func (s *Steps) run(method, text string, override []bodyFn) {
	if len(override) > 0 {
		ginkgo.By(text, override[0])
		return
	}

	var matches []string
	match := matchT{}

	for re, body := range s.definitions {
		stringMatches := re.FindStringSubmatch(text)
		if stringMatches == nil {
			continue
		}

		matches = append(matches, re.String())

		match.body = body
		match.params = stringMatches[1:]
		match.re = re
	}

	if len(matches) > 1 {
		faultMessage := fmt.Sprintf("Too many matches for `%s`:\n", text)
		for i, expression := range matches {
			faultMessage = fmt.Sprintf("%s\t%d: %s\n", faultMessage, i, expression)
		}
		s.Fail(faultMessage)
		return // not necessary but makes it clear that this does not continue
	}

	if match.body == nil {
		templateBacktick := fmt.Sprintf("define.%s(`^%s$`, func() {})", method, text)
		templateDouble := fmt.Sprintf("define.%s(\"^%s$\", func() {})", method, text)

		// Skip 2 skips this line and the alias below, resulting in the code location being where the step was called
		// in the user's feature file
		s.Fail(fmt.Sprintf("No match for `%s`, try adding:\n%s\nor:\n%s\n", text, templateBacktick, templateDouble), 2)
		return // not necessary but makes it clear that this does not continue
	}

	s.used[match.re] = true

	ginkgo.By(text, func() {
		switch match.body.(type) {
		case func():
			match.body.(func())()
		default:
			matchValue := reflect.ValueOf(match.body)

			in := make([]reflect.Value, len(match.params))

			for paramIndex := range in {
				in[paramIndex] = reflect.ValueOf(match.params[paramIndex])
			}

			matchValue.Call(in)

			//ginkgo.Fail(fmt.Sprintf("Could not match function call for \"%s\"\nlooking for:%v", text, reflect.TypeOf(match)))
		}
	})
}

func (s *Steps) Given(text string, body ...bodyFn) { s.run("Given", text, body) }
func (s *Steps) When(text string, body ...bodyFn)  { s.run("When", text, body) }
func (s *Steps) Then(text string, body ...bodyFn)  { s.run("Then", text, body) }
func (s *Steps) And(text string, body ...bodyFn)   { s.run("And", text, body) }

func (s *Steps) Run(text string) { s.run("Step", text, nil) }
