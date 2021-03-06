// Code generated by counterfeiter. DO NOT EDIT.
package internalfakes

import (
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal"
)

type FakeContainerRegistryInterface struct {
	CheckStub        func(string, name.Reference) (bool, error)
	checkMutex       sync.RWMutex
	checkArgsForCall []struct {
		arg1 string
		arg2 name.Reference
	}
	checkReturns struct {
		result1 bool
		result2 error
	}
	checkReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	PullStub        func(name.Reference) (v1.Image, string, error)
	pullMutex       sync.RWMutex
	pullArgsForCall []struct {
		arg1 name.Reference
	}
	pullReturns struct {
		result1 v1.Image
		result2 string
		result3 error
	}
	pullReturnsOnCall map[int]struct {
		result1 v1.Image
		result2 string
		result3 error
	}
	PushStub        func(v1.Image, name.Reference) error
	pushMutex       sync.RWMutex
	pushArgsForCall []struct {
		arg1 v1.Image
		arg2 name.Reference
	}
	pushReturns struct {
		result1 error
	}
	pushReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeContainerRegistryInterface) Check(arg1 string, arg2 name.Reference) (bool, error) {
	fake.checkMutex.Lock()
	ret, specificReturn := fake.checkReturnsOnCall[len(fake.checkArgsForCall)]
	fake.checkArgsForCall = append(fake.checkArgsForCall, struct {
		arg1 string
		arg2 name.Reference
	}{arg1, arg2})
	stub := fake.CheckStub
	fakeReturns := fake.checkReturns
	fake.recordInvocation("Check", []interface{}{arg1, arg2})
	fake.checkMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeContainerRegistryInterface) CheckCallCount() int {
	fake.checkMutex.RLock()
	defer fake.checkMutex.RUnlock()
	return len(fake.checkArgsForCall)
}

func (fake *FakeContainerRegistryInterface) CheckCalls(stub func(string, name.Reference) (bool, error)) {
	fake.checkMutex.Lock()
	defer fake.checkMutex.Unlock()
	fake.CheckStub = stub
}

func (fake *FakeContainerRegistryInterface) CheckArgsForCall(i int) (string, name.Reference) {
	fake.checkMutex.RLock()
	defer fake.checkMutex.RUnlock()
	argsForCall := fake.checkArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeContainerRegistryInterface) CheckReturns(result1 bool, result2 error) {
	fake.checkMutex.Lock()
	defer fake.checkMutex.Unlock()
	fake.CheckStub = nil
	fake.checkReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeContainerRegistryInterface) CheckReturnsOnCall(i int, result1 bool, result2 error) {
	fake.checkMutex.Lock()
	defer fake.checkMutex.Unlock()
	fake.CheckStub = nil
	if fake.checkReturnsOnCall == nil {
		fake.checkReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.checkReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeContainerRegistryInterface) Pull(arg1 name.Reference) (v1.Image, string, error) {
	fake.pullMutex.Lock()
	ret, specificReturn := fake.pullReturnsOnCall[len(fake.pullArgsForCall)]
	fake.pullArgsForCall = append(fake.pullArgsForCall, struct {
		arg1 name.Reference
	}{arg1})
	stub := fake.PullStub
	fakeReturns := fake.pullReturns
	fake.recordInvocation("Pull", []interface{}{arg1})
	fake.pullMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeContainerRegistryInterface) PullCallCount() int {
	fake.pullMutex.RLock()
	defer fake.pullMutex.RUnlock()
	return len(fake.pullArgsForCall)
}

func (fake *FakeContainerRegistryInterface) PullCalls(stub func(name.Reference) (v1.Image, string, error)) {
	fake.pullMutex.Lock()
	defer fake.pullMutex.Unlock()
	fake.PullStub = stub
}

func (fake *FakeContainerRegistryInterface) PullArgsForCall(i int) name.Reference {
	fake.pullMutex.RLock()
	defer fake.pullMutex.RUnlock()
	argsForCall := fake.pullArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeContainerRegistryInterface) PullReturns(result1 v1.Image, result2 string, result3 error) {
	fake.pullMutex.Lock()
	defer fake.pullMutex.Unlock()
	fake.PullStub = nil
	fake.pullReturns = struct {
		result1 v1.Image
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeContainerRegistryInterface) PullReturnsOnCall(i int, result1 v1.Image, result2 string, result3 error) {
	fake.pullMutex.Lock()
	defer fake.pullMutex.Unlock()
	fake.PullStub = nil
	if fake.pullReturnsOnCall == nil {
		fake.pullReturnsOnCall = make(map[int]struct {
			result1 v1.Image
			result2 string
			result3 error
		})
	}
	fake.pullReturnsOnCall[i] = struct {
		result1 v1.Image
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeContainerRegistryInterface) Push(arg1 v1.Image, arg2 name.Reference) error {
	fake.pushMutex.Lock()
	ret, specificReturn := fake.pushReturnsOnCall[len(fake.pushArgsForCall)]
	fake.pushArgsForCall = append(fake.pushArgsForCall, struct {
		arg1 v1.Image
		arg2 name.Reference
	}{arg1, arg2})
	stub := fake.PushStub
	fakeReturns := fake.pushReturns
	fake.recordInvocation("Push", []interface{}{arg1, arg2})
	fake.pushMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeContainerRegistryInterface) PushCallCount() int {
	fake.pushMutex.RLock()
	defer fake.pushMutex.RUnlock()
	return len(fake.pushArgsForCall)
}

func (fake *FakeContainerRegistryInterface) PushCalls(stub func(v1.Image, name.Reference) error) {
	fake.pushMutex.Lock()
	defer fake.pushMutex.Unlock()
	fake.PushStub = stub
}

func (fake *FakeContainerRegistryInterface) PushArgsForCall(i int) (v1.Image, name.Reference) {
	fake.pushMutex.RLock()
	defer fake.pushMutex.RUnlock()
	argsForCall := fake.pushArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeContainerRegistryInterface) PushReturns(result1 error) {
	fake.pushMutex.Lock()
	defer fake.pushMutex.Unlock()
	fake.PushStub = nil
	fake.pushReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeContainerRegistryInterface) PushReturnsOnCall(i int, result1 error) {
	fake.pushMutex.Lock()
	defer fake.pushMutex.Unlock()
	fake.PushStub = nil
	if fake.pushReturnsOnCall == nil {
		fake.pushReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.pushReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeContainerRegistryInterface) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.checkMutex.RLock()
	defer fake.checkMutex.RUnlock()
	fake.pullMutex.RLock()
	defer fake.pullMutex.RUnlock()
	fake.pushMutex.RLock()
	defer fake.pushMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeContainerRegistryInterface) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ internal.ContainerRegistryInterface = new(FakeContainerRegistryInterface)
