package goerkin

import (
	"fmt"
	"regexp"

	"github.com/onsi/ginkgo"
)

type Definitions interface {
	Given(re string, given interface{}, after ...func())
	When(re string, when interface{}, after ...func())
	Then(re string, then interface{}, after ...func())
}

type definitions map[*regexp.Regexp]interface{}

func (defs definitions) add(text string, body interface{}, after []func()) {
	if len(after) > 0 {
		ginkgo.AfterEach(after[0])
	}

	re, err := regexp.Compile(text)
	if err != nil {
		panic(fmt.Sprintf("Could not compile %s with error %s", text, err))
	}

	defs[re] = body
}

func (defs definitions) Given(re string, given interface{}, after ...func()) {
	defs.add(re, given, after)
}
func (defs definitions) When(re string, when interface{}, after ...func()) { defs.add(re, when, after) }
func (defs definitions) Then(re string, then interface{}, after ...func()) { defs.add(re, then, after) }
