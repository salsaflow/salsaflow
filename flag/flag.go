package flag

import (
	"fmt"
	"regexp"
)

type RegexpSetFlag struct {
	Values []*regexp.Regexp
}

func NewRegexpSetFlag() *RegexpSetFlag {
	return &RegexpSetFlag{make([]*regexp.Regexp, 0)}
}

func (set *RegexpSetFlag) String() string {
	return fmt.Sprint(set.Values)
}

func (set *RegexpSetFlag) Set(value string) error {
	for _, existing := range set.Values {
		if existing.String() == value {
			return nil
		}
	}
	re, err := regexp.Compile(value)
	if err != nil {
		return err
	}
	set.Values = append(set.Values, re)
	return nil
}
