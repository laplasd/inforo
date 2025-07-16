package controllers

import (
	"bytes"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

/*
	================
	kuber-controller
	================
*/

type KuberController struct {
	Logger *logrus.Logger
}

func (i *KuberController) RunTask(taskMeta map[string]string, componentMeta map[string]string) error {
	taskID := taskMeta["id"] // предполагаем, что ID есть в метаданных
	taskType := taskMeta["Type"]

	i.Logger.Infof("KuberController running task %s of type %s with component metadata: %+v", taskID, taskType, componentMeta)

	// Здесь может быть логика запуска kubectl, apply, check и т.д.
	time.Sleep(1 * time.Second)

	i.Logger.Infof("Task %s completed", taskID)
	return nil
}

func (i *KuberController) ValideTask(TaskMeta map[string]string) error {
	return nil
}

func (i *KuberController) ValideComponent(ComponentMeta map[string]string) error {
	return nil
}

func (i *KuberController) CheckComponent(ComponentMeta map[string]string) error {
	return nil
}

/*
	================
	ssh-controller
	================
*/

type SSHController struct {
	Logger *logrus.Logger
}

func (s *SSHController) RunTask(taskMeta map[string]string, componentMeta map[string]string) error {
	cmd := taskMeta["command"]
	taskID := taskMeta["id"]
	taskType := taskMeta["type"]

	host := componentMeta["host"]
	user := componentMeta["user"]
	password := componentMeta["password"]
	port := componentMeta["port"]
	if port == "" {
		port = "22"
	}

	if host == "" || user == "" || cmd == "" {
		return fmt.Errorf("missing required metadata (host, user, command)")
	}

	s.Logger.Infof("SSHController running task %s (%s) on %s@%s:%s: %s", taskID, taskType, user, host, port, cmd)

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // ⚠️ заменить на безопасный при боевом использовании
		Timeout:         5 * time.Second,
	}

	address := fmt.Sprintf("%s:%s", host, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("failed to dial SSH: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		s.Logger.Errorf("SSH command failed: %s", stderr.String())
		return fmt.Errorf("ssh command error: %w", err)
	}

	s.Logger.Infof("SSH task %s output:\n%s", taskID, stdout.String())
	return nil
}

func (s *SSHController) ValideTask(taskMeta map[string]string) error {
	if taskMeta["id"] == "" {
		return fmt.Errorf("task id is required")
	}
	if taskMeta["type"] == "" {
		return fmt.Errorf("task type is required")
	}
	if taskMeta["command"] == "" {
		return fmt.Errorf("task command is required")
	}
	return nil
}

func (s *SSHController) ValideComponent(componentMeta map[string]string) error {
	if componentMeta["host"] == "" {
		return fmt.Errorf("component host is required")
	}
	if componentMeta["user"] == "" {
		return fmt.Errorf("component user is required")
	}
	if componentMeta["password"] == "" {
		return fmt.Errorf("component password is required")
	}
	return nil
}

func (s *SSHController) CheckComponent(componentMeta map[string]string) error {
	host := componentMeta["host"]
	user := componentMeta["user"]
	password := componentMeta["password"]
	port := componentMeta["port"]
	if port == "" {
		port = "22"
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // ⚠️ заменить в проде
		Timeout:         5 * time.Second,
	}

	address := fmt.Sprintf("%s:%s", host, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("failed to dial SSH for check: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	if err := session.Run("echo ok"); err != nil {
		return fmt.Errorf("SSH check command failed: %w", err)
	}

	return nil
}
