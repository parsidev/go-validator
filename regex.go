package validation

import (
	"regexp"
	"sync"
)

const (
	mobileRegexString = "^09\\d{9}$"
)

func lazyRegexCompile(str string) func() *regexp.Regexp {
	var regex *regexp.Regexp
	var once sync.Once
	return func() *regexp.Regexp {
		once.Do(func() {
			regex = regexp.MustCompile(str)
		})
		return regex
	}
}

var (
	mobileRegex = lazyRegexCompile(mobileRegexString)
)
