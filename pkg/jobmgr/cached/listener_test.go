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

package cached

import (
	pbjob "github.com/uber/peloton/.gen/peloton/api/v0/job"
	"github.com/uber/peloton/.gen/peloton/api/v0/peloton"
	pbtask "github.com/uber/peloton/.gen/peloton/api/v0/task"
)

type FakeJobListener struct {
	jobID      *peloton.JobID
	jobType    pbjob.JobType
	jobRuntime *pbjob.RuntimeInfo
}

func (l *FakeJobListener) Name() string {
	return "fake_job_listener"
}

func (l *FakeJobListener) JobRuntimeChanged(
	jobID *peloton.JobID,
	jobType pbjob.JobType,
	runtime *pbjob.RuntimeInfo) {
	l.jobID = jobID
	l.jobType = jobType
	l.jobRuntime = runtime
}

func (l *FakeJobListener) TaskRuntimeChanged(
	jobID *peloton.JobID,
	instanceID uint32,
	jobType pbjob.JobType,
	runtime *pbtask.RuntimeInfo,
	labels []*peloton.Label) {
}

func (l *FakeJobListener) Reset() {
	l.jobID = nil
	l.jobRuntime = nil
}

type FakeTaskListener struct {
	jobID       *peloton.JobID
	jobType     pbjob.JobType
	instanceID  uint32
	taskRuntime *pbtask.RuntimeInfo
	labels      []*peloton.Label
}

func (l *FakeTaskListener) Name() string {
	return "fake_task_listener"
}

func (l *FakeTaskListener) JobRuntimeChanged(
	jobID *peloton.JobID,
	jobType pbjob.JobType,
	runtime *pbjob.RuntimeInfo) {
}

func (l *FakeTaskListener) TaskRuntimeChanged(
	jobID *peloton.JobID,
	instanceID uint32,
	jobType pbjob.JobType,
	runtime *pbtask.RuntimeInfo,
	labels []*peloton.Label) {
	l.jobID = jobID
	l.instanceID = instanceID
	l.jobType = jobType
	l.taskRuntime = runtime
	l.labels = labels
}
