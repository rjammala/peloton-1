// Copyright (c) 2019 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/uber/peloton/.gen/peloton/api/v0/peloton"
	"github.com/uber/peloton/.gen/peloton/api/v0/respool"

	"github.com/uber/peloton/pkg/common/stringset"
)

// LookupResourcePoolID returns the resource pool ID for a given path
func (c *Client) LookupResourcePoolID(resourcePoolPath string) (*peloton.ResourcePoolID, error) {
	request := &respool.LookupRequest{
		Path: &respool.ResourcePoolPath{
			Value: resourcePoolPath,
		},
	}

	response, err := c.resClient.LookupResourcePoolID(c.ctx, request)
	if err != nil {
		return nil, err
	}
	return response.Id, nil
}

// ExtractHostnames extracts a list of hosts from a comma-separated list
func (c *Client) ExtractHostnames(hosts string, hostSeparator string) ([]string, error) {
	hostSet := stringset.New()
	for _, host := range strings.Split(hosts, hostSeparator) {
		// removing leading and trailing white spaces
		host = strings.TrimSpace(host)
		if host == "" {
			return nil, fmt.Errorf("Host cannot be empty")
		}
		if hostSet.Contains(host) {
			return nil, fmt.Errorf("Invalid input. Duplicate entry for host %s found", host)
		}
		hostSet.Add(host)
	}
	hostSlice := hostSet.ToSlice()
	sort.Strings(hostSlice)
	return hostSlice, nil
}
