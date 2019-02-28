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

package mesos

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	mesos "github.com/uber/peloton/.gen/mesos/v1"
	sched "github.com/uber/peloton/.gen/mesos/v1/scheduler"

	"github.com/uber/peloton/storage"
	"github.com/uber/peloton/util"
	"github.com/uber/peloton/yarpc/encoding/mpb"
	"github.com/uber/peloton/yarpc/transport/mhttp"
)

const (
	// ServiceName for mesos scheduler
	ServiceName = "Scheduler"

	// Schema and path for Mesos service URL.
	serviceSchema = "http"
	servicePath   = "/api/v1/scheduler"

	// A magical framework ID, generated by md5('peloton') + "-9999".
	pelotonFrameworkID = "3dcc744f-016c-6579-9b82-6325424502d2-9999"
)

// SchedulerDriver extends the Mesos HTTP Driver API.
type SchedulerDriver interface {
	mhttp.MesosDriver
	FrameworkInfoProvider
}

// FrameworkInfoProvider can be used to retrieve mesosStreamID and frameworkID.
type FrameworkInfoProvider interface {
	GetMesosStreamID(ctx context.Context) string
	GetFrameworkID(ctx context.Context) *mesos.FrameworkID
}

// schedulerDriver implements the Mesos Driver API
type schedulerDriver struct {
	store         storage.FrameworkInfoStore
	frameworkID   *mesos.FrameworkID
	mesosStreamID string
	cfg           *FrameworkConfig
	encoding      string

	defaultHeaders http.Header
}

var instance *schedulerDriver

// InitSchedulerDriver initialize Mesos scheduler driver for Mesos scheduler
// HTTP API.
func InitSchedulerDriver(
	cfg *Config,
	store storage.FrameworkInfoStore,
	defaultHeaders http.Header) SchedulerDriver {
	// TODO: load framework ID from ZK or DB
	instance = &schedulerDriver{
		store:         store,
		frameworkID:   nil,
		mesosStreamID: "",
		cfg:           cfg.Framework,
		encoding:      cfg.Encoding,

		defaultHeaders: defaultHeaders,
	}
	return instance
}

// GetSchedulerDriver return the interface to SchedulerDriver.
func GetSchedulerDriver() SchedulerDriver {
	return instance
}

// GetFrameworkID returns the frameworkID.
// Implements FrameworkInfoProvider.GetFrameworkID().
func (d *schedulerDriver) GetFrameworkID(ctx context.Context) *mesos.FrameworkID {
	if d.frameworkID != nil {
		return d.frameworkID
	}
	frameworkIDVal, err := d.store.GetFrameworkID(ctx, d.cfg.Name)
	if err != nil {
		log.WithError(err).
			WithField("framework_name", d.cfg.Name).
			Error("Failed to GetframeworkID from db for framework")
		return nil
	}
	if frameworkIDVal == "" {
		log.WithField("framework_name", d.cfg.Name).
			Error("GetframeworkID from db is empty")
		return nil
	}
	log.WithFields(log.Fields{
		"framework_id":   frameworkIDVal,
		"framework_name": d.cfg.Name,
	}).Debug("Loaded frameworkID")
	d.frameworkID = &mesos.FrameworkID{
		Value: &frameworkIDVal,
	}
	return d.frameworkID
}

// GetMesosStreamID reads DB for the Mesos stream ID.
// Implements FrameworkInfoProvider.GetMesosStreamID().
func (d *schedulerDriver) GetMesosStreamID(ctx context.Context) string {
	id, err := d.store.GetMesosStreamID(ctx, d.cfg.Name)
	if err != nil {
		log.WithError(err).
			WithField("framework_name", d.cfg.Name).
			Error("Failed to GetmesosStreamID from db")
		return ""
	}
	log.WithFields(log.Fields{
		"stream_id": id,
		"framework": d.cfg.Name,
	}).Debug("Loaded Mesos stream id")

	// TODO: This cache variable was never used?
	d.mesosStreamID = id
	return id
}

// Returns the name of Scheduler driver.
// Implements mhttp.MesosDriver.Name().
func (d *schedulerDriver) Name() string {
	return ServiceName
}

// Returns the Mesos endpoint to be connected to.
// Implements mhttp.MesosDriver.Endpoint().
func (d *schedulerDriver) Endpoint() url.URL {
	return url.URL{
		Scheme: serviceSchema,
		Path:   servicePath,
	}
}

// Returns the Type of Mesos event message such as
// mesos.v1.scheduler.Event or mesos.v1.executor.Event
// Implements mhttp.MesosDriver.EventDataType().
func (d *schedulerDriver) EventDataType() reflect.Type {
	return reflect.TypeOf(sched.Event{})
}

func (d *schedulerDriver) prepareSubscribe(ctx context.Context) (*sched.Call, error) {
	var capabilities []*mesos.FrameworkInfo_Capability
	if d.cfg.GPUSupported {
		log.Info("GPU capability is supported")
		gpuSupported := mesos.FrameworkInfo_Capability_GPU_RESOURCES
		gpuCapability := &mesos.FrameworkInfo_Capability{
			Type: &gpuSupported,
		}
		capabilities = append(capabilities, gpuCapability)
	}

	if d.cfg.TaskKillingStateSupported {
		log.Info("Task_Killing_State capability is supported")
		taskKillingStateSupported := mesos.FrameworkInfo_Capability_TASK_KILLING_STATE
		taskKillingStateCapability := &mesos.FrameworkInfo_Capability{
			Type: &taskKillingStateSupported,
		}
		capabilities = append(capabilities, taskKillingStateCapability)
	}

	if d.cfg.PartitionAwareSupported {
		log.Info("Partition Aware capability is supported")
		partitionAwareSupported := mesos.FrameworkInfo_Capability_PARTITION_AWARE
		partitionAwareCapability := &mesos.FrameworkInfo_Capability{
			Type: &partitionAwareSupported,
		}
		capabilities = append(capabilities, partitionAwareCapability)
	}

	if d.cfg.RevocableResourcesSupported {
		log.Info("Revocable resources capability is supported")
		revocableResourcesSupported := mesos.FrameworkInfo_Capability_REVOCABLE_RESOURCES
		revocableResourcesCapability := &mesos.FrameworkInfo_Capability{
			Type: &revocableResourcesSupported,
		}
		capabilities = append(capabilities, revocableResourcesCapability)
	}

	host, err := os.Hostname()
	if err != nil {
		msg := "Failed to get host name"
		log.WithError(err).Error(msg)
		return nil, errors.Wrap(err, msg)
	}

	// Peloton has no reason to run as non-checkpoint framework.
	checkpoint := true

	info := &mesos.FrameworkInfo{
		User:            &d.cfg.User,
		Name:            &d.cfg.Name,
		FailoverTimeout: &d.cfg.FailoverTimeout,
		Checkpoint:      &checkpoint,
		Capabilities:    capabilities,
		Hostname:        &host,
		Principal:       &d.cfg.Principal,
	}

	// To make peloton consistent, if we are not able to load a valid frameworkId
	// from storage driver, we will generate our own framework id.
	// This ensures that we always uses the same framework id in any cluster.
	frameworkID := d.GetFrameworkID(ctx)
	if v := frameworkID.GetValue(); len(v) == 0 {
		frameworkID = &mesos.FrameworkID{
			Value: util.PtrPrintf(pelotonFrameworkID),
		}
	} else if v != pelotonFrameworkID {
		// TODO: Require consistent framework once all clusters are rebuilt.
		log.WithField("framework_id", v).Warn("Framework id is not consistent")
	}

	callType := sched.Call_SUBSCRIBE
	msg := &sched.Call{
		FrameworkId: frameworkID,
		Type:        &callType,
		Subscribe:   &sched.Call_Subscribe{FrameworkInfo: info},
	}

	info.Id = frameworkID
	msg.FrameworkId = frameworkID
	log.WithFields(log.Fields{
		"framework_id": frameworkID,
		"timeout":      d.cfg.FailoverTimeout,
	}).Info("Reregister to Mesos master with previous framework ID")

	if d.cfg.Role != "" {
		info.Role = &d.cfg.Role
	}

	return msg, nil
}

// PrepareSubscribeRequest returns a HTTP post request that can be used to
// initiate subscription to mesos master.
// Implements mhttp.MesosDriver.PrepareSubscribeRequest().
func (d *schedulerDriver) PrepareSubscribeRequest(ctx context.Context, mesosMasterHostPort string) (
	*http.Request, error) {

	if len(mesosMasterHostPort) == 0 {
		return nil, errors.New("No active leader detected")
	}

	subscribe, err := d.prepareSubscribe(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Failed prepareSubscribe")
	}

	body, err := mpb.MarshalPbMessage(subscribe, d.encoding)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal subscribe call")
	}

	url := d.Endpoint()
	url.Host = mesosMasterHostPort
	var req *http.Request
	req, err = http.NewRequest("POST", url.String(), strings.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "Failed HTTP request")
	}

	for k, v := range d.defaultHeaders {
		for _, vv := range v {
			req.Header.Set(k, vv)
		}
	}

	req.Header.Set("Content-Type", fmt.Sprintf("application/%s", d.encoding))
	req.Header.Set("Accept", fmt.Sprintf("application/%s", d.encoding))
	return req, nil
}

// Invoked after the subscription to Mesos is done
// Implements mhttp.MesosDriver.PostSubscribe().
func (d *schedulerDriver) PostSubscribe(ctx context.Context, mesosStreamID string) {
	err := d.store.SetMesosStreamID(ctx, d.cfg.Name, mesosStreamID)
	if err != nil {
		log.WithError(err).
			WithFields(log.Fields{
				"framework_name": d.cfg.Name,
				"stream_id":      mesosStreamID,
			}).Error("Failed to save Mesos stream ID")
	}
}

// GetContentEncoding returns the http content encoding of the Mesos
// HTTP traffic.
// Implements mhttp.MesosDriver.GetContentEncoding().
func (d *schedulerDriver) GetContentEncoding() string {
	return d.encoding
}

// GetAuthHeader returns necessary auth header used for HTTP request.
func GetAuthHeader(config *Config, secretPath string) (http.Header, error) {
	header := http.Header{}
	username := config.Framework.Principal
	if len(username) == 0 {
		log.Info("No Mesos princpial is provided to framework")
		return header, nil
	}

	if len(secretPath) == 0 {
		log.Info("No secret file is provided to framework")
		return header, nil
	}

	log.WithFields(log.Fields{
		"secret_path": secretPath,
		"principal":   username,
	}).Info("Loading Mesos Authorization header from secret file")

	buf, err := ioutil.ReadFile(secretPath)
	if err != nil {
		return nil, err
	}
	password := strings.TrimSpace(string(buf))
	auth := username + ":" + password
	basicAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	header.Add("Authorization", "Basic "+basicAuth)

	log.WithFields(log.Fields{
		"secret_path": secretPath,
		"principal":   username,
	}).Info("Mesos Authorization header loaded for principal")
	return header, nil
}