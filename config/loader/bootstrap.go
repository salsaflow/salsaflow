package loader

var bootstrapSpecs []ConfigSpec

// RegisterBootstrapConfigSpec can be used to register a config spec
// that is run during `repo bootstrap`.
//
// This function is exported for internal use by SalsaFlow packages,
// it is not supposed to be used by modules.
func RegisterBootstrapConfigSpec(spec ConfigSpec) {
	key := spec.ConfigKey()
	for _, s := range bootstrapSpecs {
		if s.ConfigKey() == key {
			return
		}
	}
	bootstrapSpecs = append(bootstrapSpecs, spec)
}
