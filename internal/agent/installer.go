package agent

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Installer struct {
	logger *slog.Logger
	args   map[string]string
}

func NewInstaller(logger *slog.Logger, args map[string]string) *Installer {
	return &Installer{logger: logger, args: args}
}

// Install – главная функция установки, поддерживает режим и создание точки восстановления
func (i *Installer) Install(files []string, mode string) (string, error) {
	// Создаём точку восстановления (только Windows)
	if err := createRestorePoint(); err != nil {
		i.logger.Warn("Failed to create restore point", "error", err)
		// Продолжаем, это не критично
	}

	var output strings.Builder
	var lastErr error

	for _, file := range files {
		i.logger.Info("Installing file", "file", file, "mode", mode)
		out, err := i.installOne(file, mode)
		output.WriteString(out)
		if err != nil {
			lastErr = err
			output.WriteString("\nERROR: ")
			output.WriteString(err.Error())
			i.logger.Error("Install failed", "file", file, "error", err)
		} else {
			i.logger.Info("Install succeeded", "file", file)
		}
	}
	return output.String(), lastErr
}

// installOne – выбор метода по расширению
func (i *Installer) installOne(file string, mode string) (string, error) {
	if _, err := os.Stat(file); err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(file))
	switch runtime.GOOS {
	case "windows":
		switch ext {
		case ".exe":
			return i.installExe(file, mode)
		case ".msi":
			return i.installMsi(file, mode)
		case ".inf":
			return i.installInf(file, mode)
		default:
			return "", fmt.Errorf("unsupported file type on Windows: %s", ext)
		}
	case "linux":
		switch ext {
		case ".deb":
			return i.installDeb(file, mode)
		case ".run":
			return i.installRun(file, mode)
		case ".tar", ".tar.gz", ".tar.bz2", ".tgz":
			return i.installTar(file, mode)
		default:
			return "", fmt.Errorf("unsupported file type on Linux: %s", ext)
		}
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// ---------- Windows ----------
func (i *Installer) installExe(file string, mode string) (string, error) {
	var args string
	if mode == "quiet" {
		args = i.args[".exe"]
		if args == "" {
			args = "/S"
		}
	} else {
		args = ""
	}
	cmd := exec.Command(file, args)
	return runCmd(cmd)
}

func (i *Installer) installMsi(file string, mode string) (string, error) {
	var args []string
	if mode == "quiet" {
		args = []string{"/i", file, "/qn"}
	} else {
		args = []string{"/i", file, "/qb"} // базовый интерфейс
	}
	cmd := exec.Command("msiexec", args...)
	return runCmd(cmd)
}

func (i *Installer) installInf(file string, mode string) (string, error) {
	cmd := exec.Command("pnputil", "-i", "-a", file)
	out, err := runCmd(cmd)
	if err == nil {
		return out, nil
	}
	i.logger.Warn("pnputil failed, trying dism", "error", err)
	cmd2 := exec.Command("dism", "/online", "/add-driver", "/driver:"+file)
	return runCmd(cmd2)
}

// ---------- Linux ----------
func (i *Installer) installDeb(file string, mode string) (string, error) {
	cmd := exec.Command("sudo", "dpkg", "-i", file)
	out, err := runCmd(cmd)
	if err != nil {
		i.logger.Warn("dpkg failed, trying apt-get -f install", "error", err)
		cmd2 := exec.Command("sudo", "apt-get", "-f", "install", "-y")
		out2, err2 := runCmd(cmd2)
		return out + "\n" + out2, err2
	}
	return out, nil
}

func (i *Installer) installRun(file string, mode string) (string, error) {
	cmd := exec.Command("sudo", "bash", file)
	return runCmd(cmd)
}

func (i *Installer) installTar(file string, mode string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "installer-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	var cmd *exec.Cmd
	switch {
	case strings.HasSuffix(file, ".tar.gz") || strings.HasSuffix(file, ".tgz"):
		cmd = exec.Command("tar", "-xzf", file, "-C", tmpDir)
	case strings.HasSuffix(file, ".tar.bz2"):
		cmd = exec.Command("tar", "-xjf", file, "-C", tmpDir)
	default:
		cmd = exec.Command("tar", "-xf", file, "-C", tmpDir)
	}
	out, err := runCmd(cmd)
	if err != nil {
		return out, fmt.Errorf("tar extraction failed: %w", err)
	}

	var script string
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			base := filepath.Base(path)
			if base == "install.sh" || base == "setup.sh" {
				script = path
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil || script == "" {
		return out, fmt.Errorf("no install.sh or setup.sh found in tar archive")
	}
	if err := os.Chmod(script, 0755); err != nil {
		return out, err
	}
	cmd2 := exec.Command("sudo", "bash", script)
	out2, err2 := runCmd(cmd2)
	return out + "\n" + out2, err2
}

// ---------- Вспомогательные ----------
func runCmd(cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}
	return output, err
}

// createRestorePoint – создаёт точку восстановления в Windows
func createRestorePoint() error {
    if runtime.GOOS != "windows" {
        return nil
    }
    // Проверяем, есть ли права администратора
    cmd := exec.Command("powershell", "-Command", "Checkpoint-Computer -Description 'LANdapter restore point' -RestorePointType MODIFY_SETTINGS")
    if err := cmd.Run(); err != nil {
        // Если функция отключена, это не ошибка – просто предупреждение
        if strings.Contains(err.Error(), "0x80042302") {
            return nil // System Restore disabled
        }
        return fmt.Errorf("failed to create restore point: %w", err)
    }
    return nil
}

// isAdmin – проверка прав администратора (Windows)
func isAdmin() bool {
	if runtime.GOOS != "windows" {
		return true
	}
	cmd := exec.Command("net", "session")
	err := cmd.Run()
	return err == nil
}