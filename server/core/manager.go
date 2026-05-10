package core

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Manager struct {
	path string
}

func NewManager(path string) *Manager {
	return &Manager{path: path}
}

func (m *Manager) Add(ip, domain string) error {
	f, err := os.Open(m.path)
	if err != nil {
		return err
	}
	defer f.Close()

	dir := filepath.Dir(m.path)
	temp, err := os.CreateTemp(dir, "tmp.*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()

	renamed := false
	defer func() {
		if !renamed {
			temp.Close()
			os.Remove(tempPath)
		}
	}()

	scanner := bufio.NewScanner(f)
	writer := bufio.NewWriter(temp)

	for scanner.Scan() {
		line := scanner.Text()
		l := strings.Fields(line)

		if len(l) >= 2 {
			if l[0] == ip {
				return ErrServerExists
			}
		}

		if _, err = writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	if _, err = writer.WriteString(fmt.Sprintf("%s %s\n", ip, domain)); err != nil {
		return err
	}

	if err = writer.Flush(); err != nil {
		return err
	}
	if err = temp.Sync(); err != nil {
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}

	if err = os.Rename(tempPath, m.path); err != nil {
		return err
	}
	renamed = true
	return nil
}

func (m *Manager) Remove(ip string) error {
	f, err := os.Open(m.path)
	if err != nil {
		return err
	}
	defer f.Close()

	dir := filepath.Dir(m.path)
	temp, err := os.CreateTemp(dir, "tmp.*")
	if err != nil {
		return err
	}

	renamed := false
	tempPath := temp.Name()
	defer func() {
		if !renamed {
			temp.Close()
			os.Remove(tempPath)
		}
	}()

	scanner := bufio.NewScanner(f)
	writer := bufio.NewWriter(temp)

	found := false
	for scanner.Scan() {
		line := scanner.Text()
		l := strings.Fields(line)

		if len(l) >= 2 {
			if l[0] == ip {
				found = true
				continue
			}
		}
		if _, err = writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	if err = scanner.Err(); err != nil {
		return err
	}
	if !found {
		return ErrServerNotFound
	}
	if err = writer.Flush(); err != nil {
		return err
	}
	if err = temp.Sync(); err != nil {
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}

	if err = os.Rename(tempPath, m.path); err != nil {
		return err
	}
	renamed = true

	return nil
}

func (m *Manager) List(w io.Writer) error {
	f, err := os.Open(m.path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}
