package config

import (
	// Stdlib
	"encoding/json"
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
)

// ConfigurationsSection ---------------------------------------------------------------

type ConfigurationsSection struct {
	Records map[string]*json.RawMessage `json:"configuration"`
}

func newConfigurationsSection() *ConfigurationsSection {
	return &ConfigurationsSection{make(map[string]*json.RawMessage)}
}

func (section *ConfigurationsSection) ConfigRecord(configKey string) (*ConfigRecord, error) {
	task := fmt.Sprintf("Find the configuration record for key '%v'", configKey)
	rawMsgPtr, ok := section.Records[configKey]
	if !ok {
		return nil, errs.NewError(task, &ErrConfigRecordNotFound{configKey})
	}
	return newConfigRecord(fmt.Sprintf(`configuration["%v"]`, configKey), *rawMsgPtr), nil
}

func (section *ConfigurationsSection) SetConfigRecord(configKey string, rawConfig []byte) {
	content := make([]byte, len(rawConfig))
	copy(content, rawConfig)

	msg := json.RawMessage(content)
	section.Records[configKey] = &msg
}

// ConfigRecord ----------------------------------------------------------------

type ConfigRecord struct {
	sectionPath string
	RawConfig   []byte
}

func newConfigRecord(path string, raw json.RawMessage) *ConfigRecord {
	return &ConfigRecord{path, []byte(raw)}
}

func (ms *ConfigRecord) Path() string {
	return ms.sectionPath
}
