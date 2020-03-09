// Copyright 2016-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package node

import (
	"flag"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-node-termination-handler/pkg/config"
	h "github.com/aws/aws-node-termination-handler/pkg/test"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/kubectl/pkg/drain"
)

var nodeName = "NAME"
var testFile = "test.out"

func resetFlagsForTest() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd"}
	os.Setenv("NODE_NAME", nodeName)
}

func getTestDrainHelper(client *fake.Clientset) *drain.Helper {
	return &drain.Helper{
		Client:              client,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		DeleteLocalData:     true,
		Timeout:             time.Duration(120) * time.Second,
		Out:                 os.Stdout,
		ErrOut:              os.Stderr,
	}
}

func getNthConfig(t *testing.T) config.Config {
	nthConfig, err := config.ParseCliArgs()
	if err != nil {
		t.Error("failed to create nthConfig")
	}
	return nthConfig
}

func getNode(t *testing.T, drainHelper *drain.Helper) *Node {
	tNode, err := NewWithValues(getNthConfig(t), drainHelper)
	if err != nil {
		t.Error("failed to create node")
	}
	return tNode
}

func TestGetUptimeSuccess(t *testing.T) {
	d1 := []byte("350735.47 234388.90")
	ioutil.WriteFile(testFile, d1, 0644)

	value, err := getSystemUptime(testFile)
	os.Remove(testFile)
	h.Ok(t, err)
	h.Equals(t, 350735.47, value)
}

func TestGetUptimeFailure(t *testing.T) {
	d1 := []byte("Something not time")
	ioutil.WriteFile(testFile, d1, 0644)

	_, err := getSystemUptime(testFile)
	os.Remove(testFile)
	h.Assert(t, err != nil, "Failed to throw error for float64 parse")
}

func TestUncordonIfRebootedFileReadError(t *testing.T) {
	resetFlagsForTest()
	uptimeFile = testFile

	client := fake.NewSimpleClientset()
	client.CoreV1().Nodes().Create(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"aws-node-termination-handler/action":      "UncordonAfterReboot",
				"aws-node-termination-handler/action-time": strconv.FormatInt(time.Now().Unix(), 10),
			},
		},
	})
	tNode := getNode(t, getTestDrainHelper(client))
	err := tNode.UncordonIfRebooted()
	h.Assert(t, err != nil, "Failed to return error on UncordonIfReboted failure to read file")
}

func TestUncordonIfRebootedSystemNotRestarted(t *testing.T) {
	resetFlagsForTest()
	uptimeFile = testFile
	d1 := []byte("350735.47 234388.90")
	ioutil.WriteFile(testFile, d1, 0644)

	client := fake.NewSimpleClientset()
	client.CoreV1().Nodes().Create(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"aws-node-termination-handler/action":      "UncordonAfterReboot",
				"aws-node-termination-handler/action-time": strconv.FormatInt(time.Now().Unix(), 10),
			},
		},
	})
	tNode := getNode(t, getTestDrainHelper(client))
	err := tNode.UncordonIfRebooted()
	os.Remove(testFile)
	h.Ok(t, err)
}

func TestUncordonIfRebootedFailureToRemoveLabel(t *testing.T) {
	resetFlagsForTest()
	uptimeFile = testFile
	d1 := []byte("0 234388.90")
	ioutil.WriteFile(testFile, d1, 0644)

	client := fake.NewSimpleClientset()
	client.CoreV1().Nodes().Create(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"aws-node-termination-handler/action":      "UncordonAfterReboot",
				"aws-node-termination-handler/action-time": strconv.FormatInt(time.Now().Unix(), 10),
			},
		},
	})
	tNode := getNode(t, getTestDrainHelper(client))
	err := tNode.UncordonIfRebooted()
	os.Remove(testFile)
	h.Assert(t, err != nil, "Failed to return error on UncordonIfReboted failure remove NTH Label")
}

func TestUncordonIfRebootedFailureSuccess(t *testing.T) {
	resetFlagsForTest()
	uptimeFile = testFile
	d1 := []byte("0 234388.90")
	ioutil.WriteFile(testFile, d1, 0644)

	client := fake.NewSimpleClientset()
	client.CoreV1().Nodes().Create(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"aws-node-termination-handler/action":      "UncordonAfterReboot",
				"aws-node-termination-handler/action-time": strconv.FormatInt(time.Now().Unix(), 10),
				"aws-node-termination-handler/event-id":    "HELLO",
			},
		},
	})
	tNode := getNode(t, getTestDrainHelper(client))
	err := tNode.UncordonIfRebooted()
	os.Remove(testFile)
	h.Ok(t, err)
}
