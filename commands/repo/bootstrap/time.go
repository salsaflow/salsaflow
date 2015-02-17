package bootstrapCmd

import "time"

// The whole point of this type is to wrap time.Time to make it work
// with go-yaml out of the box. go-yaml uses Time.String method to marshal
// time.Time while using Time.UnmarshalText for unmarshalling,
// which is unfortunately not symmetrical.
type Time time.Time

func (t Time) String() string {
	raw, err := time.Time(t).MarshalText()
	if err != nil {
		panic(err)
	}
	return string(raw)
}
