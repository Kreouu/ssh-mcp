package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type ExecInput struct {
	Host       string            `json:"host" jsonschema:"Host name from config"`
	Command    string            `json:"command" jsonschema:"Shell command to execute"`
	Workdir    string            `json:"workdir,omitempty" jsonschema:"Working directory on remote"`
	Env        map[string]string `json:"env,omitempty" jsonschema:"Environment variables"`
	TimeoutSec int               `json:"timeout_sec,omitempty" jsonschema:"Override timeout in seconds"`
}

type ExecOutput struct {
	Ok         bool   `json:"ok"`
	Host       string `json:"host"`
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	Truncated  bool   `json:"truncated"`
	DurationMs int64  `json:"duration_ms"`
}

func (a *App) runCommand(ctx context.Context, host *HostConfig, input ExecInput) (ExecOutput, error) {
	if err := a.policy.Check(input.Command); err != nil {
		return ExecOutput{}, err
	}

	timeoutSec := a.cfg.Server.DefaultTimeoutSec
	if input.TimeoutSec > 0 {
		timeoutSec = input.TimeoutSec
	}

	script, err := buildShellScript(input.Command, input.Workdir, input.Env)
	if err != nil {
		return ExecOutput{}, err
	}

	start := time.Now()
	client, err := a.dialSSH(host, a.cfg.Server.ConnectTimeoutSec)
	if err != nil {
		return ExecOutput{}, err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return ExecOutput{}, err
	}
	defer session.Close()

	stdoutBuf := newLimitedBuffer(a.cfg.Server.MaxOutputBytes)
	stderrBuf := newLimitedBuffer(a.cfg.Server.MaxOutputBytes)
	session.Stdout = stdoutBuf
	session.Stderr = stderrBuf

	runCtx := ctx
	var cancel context.CancelFunc
	if timeoutSec > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		defer cancel()
	}

	cmd := "bash -lc " + shellEscape(script)
	errCh := make(chan error, 1)
	go func() {
		errCh <- session.Run(cmd)
	}()

	var runErr error
	select {
	case <-runCtx.Done():
		_ = session.Close()
		runErr = runCtx.Err()
	case runErr = <-errCh:
	}

	exitCode := 0
	ok := true
	if runErr != nil {
		if exitErr, okStatus := runErr.(*ssh.ExitError); okStatus {
			exitCode = exitErr.ExitStatus()
			ok = false
		} else {
			return ExecOutput{}, runErr
		}
	}

	truncated := stdoutBuf.Truncated() || stderrBuf.Truncated()

	return ExecOutput{
		Ok:         ok,
		Host:       host.Name,
		Command:    input.Command,
		ExitCode:   exitCode,
		Stdout:     stdoutBuf.String(),
		Stderr:     stderrBuf.String(),
		Truncated:  truncated,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func buildShellScript(command, workdir string, env map[string]string) (string, error) {
	var sb strings.Builder
	if workdir != "" {
		sb.WriteString("cd ")
		sb.WriteString(shellEscape(workdir))
		sb.WriteString("\n")
	}
	if len(env) > 0 {
		for k, v := range env {
			if !isValidEnvKey(k) {
				return "", fmt.Errorf("invalid env key: %s", k)
			}
			sb.WriteString("export ")
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(shellEscape(v))
			sb.WriteString("\n")
		}
	}
	sb.WriteString(command)
	return sb.String(), nil
}
