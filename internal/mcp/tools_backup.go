package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (s *Server) toolCreateBackup(_ context.Context, args map[string]any) (ToolResult, error) {
	r, err := s.router(strOpt(args, "router", ""))
	if err != nil {
		return ToolResult{}, err
	}

	dryRun := boolOpt(args, "dry_run", true)
	uploadS3 := boolOpt(args, "upload_s3", true)

	// Default backup name: routername-YYYYMMDD-HHMMSS
	backupName := strOpt(args, "name", "")
	if backupName == "" {
		backupName = fmt.Sprintf("%s-%s", r.Name(), time.Now().UTC().Format("20060102-150405"))
	}
	filename := backupName + ".backup"

	s3Available := s.s3 != nil && uploadS3

	if dryRun {
		var sb strings.Builder
		sb.WriteString("⚠️  DRY RUN — no changes made\n")
		sb.WriteString(strings.Repeat("═", 40) + "\n\n")
		fmt.Fprintf(&sb, "Would create backup on %s:\n\n", r.Name())
		fmt.Fprintf(&sb, "  Filename:  %s\n", filename)
		fmt.Fprintf(&sb, "  Upload S3: %v\n", s3Available)
		if s3Available {
			fmt.Fprintf(&sb, "  S3 path:   %s\n", s.s3.S3URL(filename))
		} else if uploadS3 && s.s3 == nil {
			fmt.Fprintf(&sb, "  ⚠ S3 not configured — backup will remain on router only\n")
		}
		fmt.Fprintf(&sb, "\nTo apply: call create_backup again with dry_run=false\n")
		return textResult(sb.String()), nil
	}

	// Step 1: Create backup on router
	_, err = r.Post("/system/backup/save", map[string]string{
		"name":         backupName,
		"dont-encrypt": "yes",
	})
	if err != nil {
		return ToolResult{}, fmt.Errorf("create backup on %s: %w", r.Name(), err)
	}

	// Wait for file to be written
	time.Sleep(3 * time.Second)

	// Step 2: Confirm file exists
	rawFiles, err := r.Get("/file")
	if err != nil {
		return ToolResult{}, fmt.Errorf("list files on %s: %w", r.Name(), err)
	}

	var files []map[string]string
	if err := json.Unmarshal(rawFiles, &files); err != nil {
		return ToolResult{}, fmt.Errorf("parse file list: %w", err)
	}

	var backupFile map[string]string
	for _, f := range files {
		if f["name"] == filename {
			backupFile = f
			break
		}
	}
	if backupFile == nil {
		return ToolResult{}, fmt.Errorf("backup file %q not found on router after creation — check router logs", filename)
	}

	var sb strings.Builder
	sb.WriteString("✓ Backup created on router\n")
	sb.WriteString(strings.Repeat("═", 40) + "\n\n")
	fmt.Fprintf(&sb, "Router:   %s\n", r.Name())
	fmt.Fprintf(&sb, "File:     %s (%s bytes)\n", filename, backupFile["size"])
	fmt.Fprintf(&sb, "Created:  %s\n", backupFile["creation-time"])

	// Step 3: Download and upload to S3
	if s3Available {
		data, err := r.DownloadFile(filename)
		if err != nil {
			fmt.Fprintf(&sb, "\n⚠ Download failed: %v\n", err)
			fmt.Fprintf(&sb, "  Backup remains on router at: /%s\n", filename)
			return textResult(sb.String()), nil
		}

		s3Key := filename
		if err := s.s3.PutObject(s3Key, data, "application/octet-stream"); err != nil {
			fmt.Fprintf(&sb, "\n⚠ S3 upload failed: %v\n", err)
			fmt.Fprintf(&sb, "  Backup remains on router at: /%s\n", filename)
		} else {
			s3Path := s.s3.S3URL(s3Key)
			fmt.Fprintf(&sb, "\n✓ Uploaded to S3\n")
			fmt.Fprintf(&sb, "  Path:     %s\n", s3Path)
			fmt.Fprintf(&sb, "  Size:     %d bytes\n", len(data))
		}
	} else {
		fmt.Fprintf(&sb, "\nBackup stored on router only. Configure AWS_* env vars to enable S3 upload.\n")
		fmt.Fprintf(&sb, "To download manually: access https://<router>/%s\n", filename)
	}

	return textResult(sb.String()), nil
}
