package flag

import (
	"errors"
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

type StringEnumFlag struct {
	choices []string
	value   string
}

func NewStringEnumFlag(choices []string, defaultChoice string) *StringEnumFlag {
	return &StringEnumFlag{choices, defaultChoice}
}

func (enum *StringEnumFlag) String() string {
	return enum.value
}

func (enum *StringEnumFlag) Set(value string) error {
	for _, c := range enum.choices {
		if value == c {
			enum.value = value
			return nil
		}
	}
	return errors.New("not one of the possible enum values: " + value)
}

func (enum *StringEnumFlag) Value() string {
	return enum.value
}
