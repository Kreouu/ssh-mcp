package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
)

type UploadInput struct {
	Host         string `json:"host" jsonschema:"Host name from config"`
	LocalPath    string `json:"local_path" jsonschema:"Local file or directory to upload"`
	RemotePath   string `json:"remote_path" jsonschema:"Remote destination path"`
	Overwrite    bool   `json:"overwrite,omitempty" jsonschema:"Overwrite existing files"`
	PreserveMode bool   `json:"preserve_mode,omitempty" jsonschema:"Preserve file permissions"`
}

type UploadOutput struct {
	Host        string `json:"host"`
	LocalPath   string `json:"local_path"`
	RemotePath  string `json:"remote_path"`
	Files       int    `json:"files"`
	Bytes       int64  `json:"bytes"`
	DurationMs  int64  `json:"duration_ms"`
	Directories int    `json:"directories"`
}

type DownloadInput struct {
	Host         string `json:"host" jsonschema:"Host name from config"`
	RemotePath   string `json:"remote_path" jsonschema:"Remote file or directory to download"`
	LocalPath    string `json:"local_path" jsonschema:"Local destination path"`
	Overwrite    bool   `json:"overwrite,omitempty" jsonschema:"Overwrite existing files"`
	PreserveMode bool   `json:"preserve_mode,omitempty" jsonschema:"Preserve file permissions"`
}

type DownloadOutput struct {
	Host        string `json:"host"`
	RemotePath  string `json:"remote_path"`
	LocalPath   string `json:"local_path"`
	Files       int    `json:"files"`
	Bytes       int64  `json:"bytes"`
	DurationMs  int64  `json:"duration_ms"`
	Directories int    `json:"directories"`
}

func (a *App) upload(ctx context.Context, host *HostConfig, input UploadInput) (UploadOutput, error) {
	start := time.Now()
	localPath := expandUserPath(input.LocalPath)
	info, err := os.Stat(localPath)
	if err != nil {
		return UploadOutput{}, fmt.Errorf("stat local path: %w", err)
	}

	client, err := a.dialSSH(host, a.cfg.Server.ConnectTimeoutSec)
	if err != nil {
		return UploadOutput{}, err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return UploadOutput{}, fmt.Errorf("sftp connect: %w", err)
	}
	defer sftpClient.Close()

	remotePath := input.RemotePath
	if !info.IsDir() {
		remotePath, err = resolveRemoteFileTarget(sftpClient, remotePath, filepath.Base(localPath))
		if err != nil {
			return UploadOutput{}, err
		}
	}

	var out UploadOutput
	out.Host = host.Name
	out.LocalPath = input.LocalPath
	out.RemotePath = remotePath

	if info.IsDir() {
		err = uploadDir(ctx, sftpClient, localPath, remotePath, input.Overwrite, input.PreserveMode, &out)
	} else {
		out.Files = 1
		out.Bytes, err = uploadFile(ctx, sftpClient, localPath, remotePath, info.Mode(), input.Overwrite, input.PreserveMode)
	}
	if err != nil {
		return UploadOutput{}, err
	}

	out.DurationMs = time.Since(start).Milliseconds()
	return out, nil
}

func (a *App) download(ctx context.Context, host *HostConfig, input DownloadInput) (DownloadOutput, error) {
	start := time.Now()
	localPath := expandUserPath(input.LocalPath)

	client, err := a.dialSSH(host, a.cfg.Server.ConnectTimeoutSec)
	if err != nil {
		return DownloadOutput{}, err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return DownloadOutput{}, fmt.Errorf("sftp connect: %w", err)
	}
	defer sftpClient.Close()

	info, err := sftpClient.Stat(input.RemotePath)
	if err != nil {
		return DownloadOutput{}, fmt.Errorf("stat remote path: %w", err)
	}

	var out DownloadOutput
	out.Host = host.Name
	out.RemotePath = input.RemotePath
	out.LocalPath = input.LocalPath

	if info.IsDir() {
		err = downloadDir(ctx, sftpClient, input.RemotePath, localPath, input.Overwrite, input.PreserveMode, &out)
	} else {
		target, err := resolveLocalFileTarget(localPath, filepath.Base(input.RemotePath))
		if err != nil {
			return DownloadOutput{}, err
		}
		out.LocalPath = target
		out.Files = 1
		out.Bytes, err = downloadFile(ctx, sftpClient, input.RemotePath, target, info.Mode(), input.Overwrite, input.PreserveMode)
	}
	if err != nil {
		return DownloadOutput{}, err
	}

	out.DurationMs = time.Since(start).Milliseconds()
	return out, nil
}

func uploadDir(ctx context.Context, client *sftp.Client, localRoot, remoteRoot string, overwrite, preserveMode bool, out *UploadOutput) error {
	return filepath.WalkDir(localRoot, func(pathname string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rel, err := filepath.Rel(localRoot, pathname)
		if err != nil {
			return err
		}
		if rel == "." {
			rel = ""
		}
		remotePath := joinRemotePath(remoteRoot, filepath.ToSlash(rel))
		if entry.IsDir() {
			if err := client.MkdirAll(remotePath); err != nil {
				return err
			}
			out.Directories++
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		bytes, err := uploadFile(ctx, client, pathname, remotePath, info.Mode(), overwrite, preserveMode)
		if err != nil {
			return err
		}
		out.Files++
		out.Bytes += bytes
		return nil
	})
}

func uploadFile(ctx context.Context, client *sftp.Client, localPath, remotePath string, mode os.FileMode, overwrite, preserveMode bool) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	if !overwrite {
		if _, err := client.Stat(remotePath); err == nil {
			return 0, fmt.Errorf("remote file exists: %s", remotePath)
		}
	}

	if err := client.MkdirAll(path.Dir(remotePath)); err != nil {
		return 0, err
	}

	src, err := os.Open(localPath)
	if err != nil {
		return 0, err
	}
	defer src.Close()

	flags := os.O_CREATE | os.O_WRONLY
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	dst, err := client.OpenFile(remotePath, flags)
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		return n, err
	}

	if preserveMode {
		_ = client.Chmod(remotePath, mode.Perm())
	}
	return n, nil
}

func downloadDir(ctx context.Context, client *sftp.Client, remoteRoot, localRoot string, overwrite, preserveMode bool, out *DownloadOutput) error {
	walker := client.Walk(remoteRoot)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rel := strings.TrimPrefix(walker.Path(), remoteRoot)
		rel = strings.TrimPrefix(rel, "/")
		localPath := filepath.Join(localRoot, filepath.FromSlash(rel))
		if walker.Stat().IsDir() {
			if err := os.MkdirAll(localPath, 0o755); err != nil {
				return err
			}
			out.Directories++
			continue
		}
		bytes, err := downloadFile(ctx, client, walker.Path(), localPath, walker.Stat().Mode(), overwrite, preserveMode)
		if err != nil {
			return err
		}
		out.Files++
		out.Bytes += bytes
	}
	return nil
}

func downloadFile(ctx context.Context, client *sftp.Client, remotePath, localPath string, mode os.FileMode, overwrite, preserveMode bool) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	if !overwrite {
		if _, err := os.Stat(localPath); err == nil {
			return 0, fmt.Errorf("local file exists: %s", localPath)
		}
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return 0, err
	}

	src, err := client.Open(remotePath)
	if err != nil {
		return 0, err
	}
	defer src.Close()

	flags := os.O_CREATE | os.O_WRONLY
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	dst, err := os.OpenFile(localPath, flags, 0o644)
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		return n, err
	}
	if preserveMode {
		_ = os.Chmod(localPath, mode.Perm())
	}
	return n, nil
}

func resolveRemoteFileTarget(client *sftp.Client, remotePath, localBase string) (string, error) {
	if strings.HasSuffix(remotePath, "/") {
		return joinRemotePath(remotePath, localBase), nil
	}
	info, err := client.Stat(remotePath)
	if err == nil && info.IsDir() {
		return joinRemotePath(remotePath, localBase), nil
	}
	return remotePath, nil
}

func resolveLocalFileTarget(localPath, remoteBase string) (string, error) {
	if strings.HasSuffix(localPath, string(os.PathSeparator)) {
		return filepath.Join(localPath, remoteBase), nil
	}
	info, err := os.Stat(localPath)
	if err == nil && info.IsDir() {
		return filepath.Join(localPath, remoteBase), nil
	}
	return localPath, nil
}
