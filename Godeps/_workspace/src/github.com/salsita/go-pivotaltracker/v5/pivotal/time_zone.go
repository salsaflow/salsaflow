// Copyright (c) 2014 Salsita Software
// Use of this source code is governed by the MIT License.
// The license can be found in the LICENSE file.

package pivotal

type TimeZone struct {
	OlsonName string `json:"olson_name,omitempty"`
	Offset    string `json:"offset,omitempty"`
}
