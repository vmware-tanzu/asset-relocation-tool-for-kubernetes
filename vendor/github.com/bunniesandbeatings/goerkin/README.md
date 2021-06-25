* Travis Tests [![Build Status](https://travis-ci.com/bunniesandbeatings/goerkin.svg?branch=master)](https://travis-ci.com/bunniesandbeatings/goerkin)
* Covenant: [![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v1.4%20adopted-ff69b4.svg)](CODE_OF_CONDUCT.md)

# Goerkin
A Gherkin DSL for Ginkgo

Pronounced just like Gherkin OR Go-erk-in. Your choice.

Inspired by [Robbie Clutton's simple_bdd](https://github.com/robb1e/simple_bdd)

Goerkin is great for feature tests. Let us know if you use it, and what for! We'd love to hear feedback.

* Designed as an extension to [Ginkgo](https://github.com/onsi/ginkgo). 
* We recommend [Gomega](https://github.com/onsi/gomega) matchers.
* For testing web apps, try combining Goerkin with [Agouti](https://github.com/sclevine/agouti).
* Use it even if your app isn't written in Go. It's a nice expressive way to build tests for JS, Node and other kinds of apps.  

# Goals
* Provide the gherkin format for stories
    * without a special `*.feature` format
* Local step definitions instead of shared steps which often drives developers toward [the wrong abstraction](https://www.sandimetz.com/blog/2016/1/20/the-wrong-abstraction)
    * of course you can still [share steps](#shared-steps)
* Lean on Ginkgo so as not to create a whole other BDD system that needs extensive design and testing
* Promote imperative style tests
    * Dissuade the use of BeforeEach/AfterEach

# Samples

You can find most of these use cases as [actual tests in the documentation test](https://github.com/bunniesandbeatings/goerkin/blob/master/features/documentation_test.go)

## Simple usage
```go
    import (
        . "github.com/onsi/ginkgo"
        . "github.com/onsi/gomega"
        . "github.com/bunniesandbeatings/goerkin"
    )

    var _ = Describe("running a total", func() {
        var (
            total int
        )
    
        steps := Define(func(define Definitions) {
            define.Given("The current total is cleared", func() {
            	total = 0
            })
    
            define.When("^I add 5$", func() {
            	total = total + 5
            })
    
            define.When("^I add 3$", func() {
                total = total + 3
            })
    
            define.Then("^The total is 8$", func() {
                Expect(total).To(Equal(8))
            })
        })
    
        Scenario("Adding", func() {
            steps.Given("The current total is cleared")
            
            steps.When("I add 5")
            steps.And("I add 3")
            
            steps.Then("The total is 8")
        })

        Scenario("Subtracting with inline definitions", func() {
            steps.Given("The current total is cleared")
            
            steps.When("I add 5")
            steps.And("I subtract 3", func() {
            	total = total - 3
            })
            
            steps.Then("The total is 2", func() {
            	Expect(total).To(Equal(2))
            })
        })
    })
```

## Calling steps from within other steps
```go
    var _ = Describe("running a total", func() {
        var (
            total int
            steps *Steps
        )
    
        steps = Define(func(define Definitions) {
            define.Given("The current total is cleared", func() {
            	total = 0
            })
    
            define.When("^I add 5$", func() {
            	total = total + 5
            })
    
            define.When("^I add 3$", func() {
                total = total + 3
            })

            define.When("^I add 5 and 3 to the total$", func() {
                steps.Run("I add 5")
                steps.Run("I add 3")
            })
            
            define.Then("^The total is 8$", func() {
                Expect(total).To(Equal(8))
            })
        })
    
        Scenario("Adding", func() {
            steps.Given("The current total is cleared")
            
            steps.When("I add 5 and 3 to the total")
            
            steps.Then("The total is 8")
        })
    })
```
## Features first
I like my features at the top of the file. You can do that:

```go
    var _ = Describe("running a total", func() {
        var (
            total int
        )
    
        steps := NewSteps()

        Scenario("Adding", func() {
            steps.Given("The current total is cleared")
            
            steps.When("I add 5")
            steps.And("I add 3")
            
            steps.Then("The total is 8")
        })

        Scenario("Subtracting with inline definitions", func() {
            steps.Given("The current total is cleared")
            
            steps.When("I add 5")
            steps.And("I subtract 3", func() {
            	total = total - 3
            })
            
            steps.Then("The total is 2", func() {
            	Expect(total).To(Equal(2))
            })
        })
        
        
        steps.Define(func(define Definitions) {
            define.Given("The current total is cleared", func() {
            	total = 0
            })
    
            define.When("^I add 5$", func() {
            	total = total + 5
            })
    
            define.When("^I add 3$", func() {
                total = total + 3
            })
    
            define.Then("^The total is 8$", func() {
                Expect(total).To(Equal(8))
            })
        })
    
    })
```

## Cleanup Steps

`Givens` and `Whens` support cleanup methods

```go
    var _ = Describe("Daemonize works", func() {
        var (
            app *exec.Cmd
        )
    
        steps := NewSteps()

        Scenario("Running", func() {
            steps.Given("My server is running")
            
            steps.When("I visit it's url")
            
            steps.Then("It responds")
        })

        
        
        steps.Define(func(define Definitions) {
            define.Given("My server is running",
            	func() {
            	    app := startMyServer()
                },
                func() {
                	// this is a cleanup step
                	stopMyServer(app)
                }
            )
    
            ... blah, blah blah blablah ...
        })
        
            
    })
```

## Shared Steps

You can define shared steps and re-use them. Within the same
package, package level var's are great for sharing state. Because
we tend to put all our feature tests in one `features` package
this has not been an issue. Let us know if you need to share 
step definitions across packages, and if you have a solution 
you can share.


```go
// your_test.go:
package features_test

var _ = Describe("Shared Steps with the framework", func() {
	steps := NewSteps()

	Scenario("Use a shared step", func() {
		steps.Given("I am a shared step")
		steps.Then("I can depend upon it")
	})

	steps.Define(
		sharedSteps, // framework addition
		func(define Definitions) {
			define.Then(`^I can depend upon it$`, func() {
				Expect(sharedValue).To(Equal("shared step called"))
			})
		},
	)
})
```

```go
// shared_steps_test.go:
package features_test

var sharedValue string

var sharedSteps = func(define Definitions) {
	define.Given(`^I am a shared step$`, func() {
		sharedValue = "shared step called"
	}, func() {
		sharedValue = "" // remember to clean up broadly scoped variables
	})
}
```

## Unused steps

If you want to find all your unused steps, run the entire suite with env var `UNUSED_FAIL` set:

```bash
    UNUSED_FAIL=true && ginkgo -r .
```

For _shared steps_ this will fail fast (first Describe block that doesn't use all of the shared steps),
so **tread carefully** with shared steps. Make sure you run all your tests again with UNUSED_FAIL unset.

# Guidelines

**Please note** that this project is released with a Contributor Code of Conduct. 
By participating in this project you agree to abide by its terms.
