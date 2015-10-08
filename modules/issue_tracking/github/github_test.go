package github

import (
	// Stdlib
	"testing"

	// Vendor - testing framework
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var (
	BeforeEach     = ginkgo.BeforeEach
	Context        = ginkgo.Context
	Describe       = ginkgo.Describe
	It             = ginkgo.It
	JustBeforeEach = ginkgo.JustBeforeEach

	BeEmpty = gomega.BeEmpty
	BeNil   = gomega.BeNil
	BeTrue  = gomega.BeTrue
	BeZero  = gomega.BeZero
	Equal   = gomega.Equal
	Expect  = gomega.Expect
)

func TestGitHubUtilities(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "GitHub Issues")
}
