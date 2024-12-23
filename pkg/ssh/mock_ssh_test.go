/*
Copyright 2024 The Forge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ssh

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"
)

func TestMockSSHClient_GetSSHPassword(t *testing.T) {
	// Create a new instance of MockSSHClient
	c := &MockSSHClient{}

	// Test case 1: MockGetSSHPassword is nil
	password := c.GetSSHPassword()
	if password != "" {
		t.Errorf("Expected empty password, got %s", password)
	}

	// Test case 2: MockGetSSHPassword is defined
	expectedPassword := "test123"
	c.MockGetSSHPassword = func() string {
		return expectedPassword
	}
	password = c.GetSSHPassword()
	if password != expectedPassword {
		t.Errorf("Expected password %s, got %s", expectedPassword, password)
	}
}

func TestMockSSHClient_SetSSHPassword(t *testing.T) {
	// Create a new instance of MockSSHClient
	c := &MockSSHClient{}

	// Test case 1: MockSetSSHPassword is nil
	c.SetSSHPassword("test123")
	// Verify that the password is not set
	if c.MockSetSSHPassword != nil {
		t.Errorf("Expected MockSetSSHPassword to be nil")
	}

	// Test case 2: MockSetSSHPassword is defined
	expectedPassword := "test123"
	c.MockSetSSHPassword = func(s string) {
		if s != expectedPassword {
			t.Errorf("Expected password %s, got %s", expectedPassword, s)
		}
	}
	c.SetSSHPassword(expectedPassword)
	// Verify that the password is set correctly
	if c.MockSetSSHPassword == nil {
		t.Errorf("Expected MockSetSSHPassword to be defined")
	}
}

func TestMockSSHClient_Connect(t *testing.T) {
	// Test case 1: MockConnect is nil
	c := &MockSSHClient{}
	err := c.Connect()
	if err != ErrNotImplemented {
		t.Errorf("Expected error %v, got %v", ErrNotImplemented, err)
	}

	// Test case 2: MockConnect is defined
	expectedError := errors.New("connection error")
	c.MockConnect = func() error {
		return expectedError
	}
	err = c.Connect()
	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}
}
func TestMockSSHClient_Disconnect(t *testing.T) {
	// Test case 1: MockDisconnect is nil
	c := &MockSSHClient{}
	c.Disconnect() // Call the Disconnect method
	// No assertion is needed as the method does nothing when MockDisconnect is nil

	// Test case 2: MockDisconnect is defined
	disconnectCalled := false
	c.MockDisconnect = func() {
		disconnectCalled = true
	}
	c.Disconnect() // Call the Disconnect method
	// Verify that the MockDisconnect function is called
	if !disconnectCalled {
		t.Errorf("Expected MockDisconnect to be called")
	}
}

func TestMockSSHClient_Run(t *testing.T) {
	// Test case 1: MockRun is nil
	c := &MockSSHClient{}
	command := "ls"
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := c.Run(command, stdout, stderr)
	if err != ErrNotImplemented {
		t.Errorf("Expected error %v, got %v", ErrNotImplemented, err)
	}

	// Test case 2: MockRun is defined
	expectedError := errors.New("run error")
	c.MockRun = func(cmd string, out io.Writer, err io.Writer) error {
		if cmd != command || out != stdout || err != stderr {
			if cmd != command || out != stdout || err != stderr {
				t.Errorf("Expected command %s, stdout %v, stderr %v, got command %s, stdout %v, stderr %v",
					command, stdout, stderr, cmd, out, err)
			}
		}
		return expectedError
	}
	err = c.Run(command, stdout, stderr)
	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}
}
func TestMockSSHClient_Validate(t *testing.T) {
	// Test case 1: MockValidate is nil
	c := &MockSSHClient{}
	err := c.Validate()
	if err != ErrNotImplemented {
		t.Errorf("Expected error %v, got %v", ErrNotImplemented, err)
	}

	// Test case 2: MockValidate is defined
	expectedError := errors.New("validation error")
	c.MockValidate = func() error {
		return expectedError
	}
	err = c.Validate()
	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}
}
func TestMockSSHClient_WaitForSSH(t *testing.T) {
	// Test case 1: MockWaitForSSH is nil
	c := &MockSSHClient{}
	maxWait := time.Second
	err := c.WaitForSSH(maxWait)
	if err != ErrNotImplemented {
		t.Errorf("Expected error %v, got %v", ErrNotImplemented, err)
	}

	// Test case 2: MockWaitForSSH is defined
	expectedError := errors.New("wait for SSH error")
	c.MockWaitForSSH = func(maxWait time.Duration) error {
		return expectedError
	}
	err = c.WaitForSSH(maxWait)
	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}
}
func TestMockSSHClient_SetSSHPrivateKey(t *testing.T) {
	// Test case 1: MockSetSSHPrivateKey is nil
	c := &MockSSHClient{}
	c.SetSSHPrivateKey("private_key")
	// No assertion is needed as the method does nothing when MockSetSSHPrivateKey is nil

	// Test case 2: MockSetSSHPrivateKey is defined
	expectedPrivateKey := "private_key"
	setPrivateKeyCalled := false
	c.MockSetSSHPrivateKey = func(s string) {
		if s != expectedPrivateKey {
			t.Errorf("Expected private key %s, got %s", expectedPrivateKey, s)
		}
		setPrivateKeyCalled = true
	}
	c.SetSSHPrivateKey(expectedPrivateKey)
	// Verify that the MockSetSSHPrivateKey function is called
	if !setPrivateKeyCalled {
		t.Errorf("Expected MockSetSSHPrivateKey to be called")
	}
}
func TestMockSSHClient_GetSSHPrivateKey(t *testing.T) {
	// Test case 1: MockGetSSHPrivateKey is nil
	c := &MockSSHClient{}
	privateKey := c.GetSSHPrivateKey()
	if privateKey != "" {
		t.Errorf("Expected empty private key, got %s", privateKey)
	}

	// Test case 2: MockGetSSHPrivateKey is defined
	expectedPrivateKey := "test_private_key"
	c.MockGetSSHPrivateKey = func() string {
		return expectedPrivateKey
	}
	privateKey = c.GetSSHPrivateKey()
	if privateKey != expectedPrivateKey {
		t.Errorf("Expected private key %s, got %s", expectedPrivateKey, privateKey)
	}
}
func TestMockSSHClient_Download(t *testing.T) {
	// Test case 1: MockDownload is nil
	c := &MockSSHClient{}
	src := &mockWriteCloser{}
	dst := "test.txt"
	err := c.Download(src, dst)
	if err != ErrNotImplemented {
		t.Errorf("Expected error %v, got %v", ErrNotImplemented, err)
	}

	// Test case 2: MockDownload is defined
	expectedError := errors.New("download error")
	c.MockDownload = func(src io.WriteCloser, dst string) error {
		return expectedError
	}
	err = c.Download(src, dst)
	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}
}

type mockWriteCloser struct{}

func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockWriteCloser) Close() error {
	return nil
}
func TestMockSSHClient_Upload(t *testing.T) {
	// Test case 1: MockUpload is nil
	c := &MockSSHClient{}
	src := &mockReader{}
	dst := "test.txt"
	mode := uint32(0644)
	err := c.Upload(src, dst, mode)
	if err != ErrNotImplemented {
		t.Errorf("Expected error %v, got %v", ErrNotImplemented, err)
	}

	// Test case 2: MockUpload is defined
	expectedError := errors.New("upload error")
	c.MockUpload = func(src io.Reader, dst string, mode uint32) error {
		return expectedError
	}
	err = c.Upload(src, dst, mode)
	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}
}

type mockReader struct{}

func (m *mockReader) Read(p []byte) (n int, err error) {
	return len(p), nil
}
