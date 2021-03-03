package core

import (
	"errors"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/opendexnetwork/opendex-launcher/build"
	"github.com/opendexnetwork/opendex-launcher/utils"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	DefaultConfigFilename = "opendex-docker.conf"
)

var (
	ErrHomeDirEmpty = errors.New("homeDir is empty")
	ErrNetworkEmpty = errors.New("network is empty")
	Debug = false
)

func init() {
	value, present := os.LookupEnv("DEBUG")
	if present {
		value = strings.TrimSpace(value)
		value = strings.ToLower(value)
		if value == "true" || value == "on" || value == "1" {
			Debug = true
		}
	}
}

type Launcher struct {

	network string
	branch  string

	homeDir             string
	networkDir          string
	launcherDir         string
	launcherVersionsDir string

	configFile string
	config     *Config

	github *GithubClient
}

func getHomeDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	switch runtime.GOOS {
	case "linux":
		return filepath.Join(homeDir, ".opendex-docker"), nil
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "OpendexDocker"), nil
	case "windows":
		return filepath.Join(homeDir, "AppData", "Local", "OpendexDocker"), nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func getNetwork() string {
	if value, ok := os.LookupEnv("NETWORK"); ok {
		return value
	}
	return "mainnet"
}

func getBranch() string {
	if value, ok := os.LookupEnv("BRANCH"); ok {
		return value
	}
	return "master"
}

func NewLauncher() *Launcher {
	value, present := os.LookupEnv("DEBUG")
	if present {
		value = strings.TrimSpace(value)
		value = strings.ToLower(value)
		if value == "true" || value == "on" || value == "1" {
			Debug = true
		}
	}

	return &Launcher{}
}

func (t *Launcher) init() error {
	if _, err := os.Stat(t.launcherDir); os.IsNotExist(err) {
		if err := os.MkdirAll(t.launcherDir, 0755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
	}
	err := os.Chdir(t.launcherDir)
	if err != nil {
		return fmt.Errorf("chdir: %w", err)
	}

	return nil
}

func (t *Launcher) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (t *Launcher) parseConfig() error {
	t.configFile = filepath.Join(t.homeDir, DefaultConfigFilename)
	exists, err := utils.FileExists(t.configFile)
	if err != nil {
		return err
	}
	if !exists {
		// no config file presented
		return nil
	}

	var c *Config

	f, err := os.Open(t.configFile)
	if err != nil {
		return err
	}
	defer f.Close()
	c, err = parseConfig(f)
	if err != nil {
		return err
	}
	t.config = c
	return nil
}

// checkDir checks if path is a writable folder or creates a new folder when path missing.
func (t *Launcher) checkDir(path string) error {
	exists, err := utils.FileExists(path)
	if err != nil {
		return err
	}
	if !exists {
		if err := os.Mkdir(path, 0755); err != nil {
			return err
		}
	}
	dir, err := utils.IsDir(path)
	if err != nil {
		return err
	}
	if !dir {
		return fmt.Errorf("not a folder: " + path)
	}
	writable, err := utils.IsWritable(path)
	if err != nil {
		return err
	}
	if !writable {
		return fmt.Errorf("not writable: " + path)
	}
	return nil
}

func (t *Launcher) ensureHomeDir() error {
	homeDir, err := getHomeDir()
	if err != nil {
		return err
	}
	if err := t.checkDir(homeDir); err != nil {
		return err
	}
	t.homeDir = homeDir
	return nil
}

func (t *Launcher) ensureNetworkDir() error {
	if t.homeDir == "" {
		return ErrHomeDirEmpty
	}
	if t.network == "" {
		return ErrNetworkEmpty
	}
	networkDir := filepath.Join(t.homeDir, t.network)
	if err := t.checkDir(networkDir); err != nil {
		return err
	}
	t.networkDir = networkDir
	return nil
}

func (t *Launcher) ensureLauncherDir() error {
	if t.homeDir == "" {
		return ErrHomeDirEmpty
	}
	launcherDir := filepath.Join(t.homeDir, "launcher")
	if err := t.checkDir(launcherDir); err != nil {
		return err
	}
	t.launcherDir = launcherDir

	versionsDir := filepath.Join(launcherDir, "versions")
	if err := t.checkDir(versionsDir); err != nil {
		return err
	}
	t.launcherVersionsDir = versionsDir

	return nil
}

func (t *Launcher) ensureDirs() error {
	if err := t.ensureHomeDir(); err != nil {
		return err
	}
	if err := t.ensureLauncherDir(); err != nil {
		return err
	}

	t.network = getNetwork()
	if t.network != "" {
		if err := t.ensureNetworkDir(); err != nil {
			return err
		}
	}
	return nil
}

func (t *Launcher) Start() error {
	if err := t.ensureDirs(); err != nil {
		return err
	}

	if err := t.parseConfig(); err != nil {
		return err
	}
	t.github = NewGithubClient(t.config.GitHub.AccessToken)

	t.branch = getBranch()

	args := os.Args

	commit, err := t.github.GetHeadCommit(t.branch)
	if err != nil {
		return fmt.Errorf("get branch head: %w", err)
	}

	if Debug {
		fmt.Printf("Branch: %s (%s)\n", t.branch, commit)
		fmt.Printf("Network: %s (%s)\n", t.network, t.networkDir)
	}

	var launcher string
	if runtime.GOOS == "windows" {
		launcher = filepath.Join(t.launcherVersionsDir, commit, "launcher.exe")
	} else {
		launcher = filepath.Join(t.launcherVersionsDir, commit, "launcher")
	}

	exists, err := utils.FileExists(launcher)
	if err != nil {
		return err
	}
	if !exists {
		if err := t.github.DownloadLatestBinary(t.branch, commit, t.launcherVersionsDir); err != nil {
			return err
		}
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		executable, err := utils.IsExecutable(launcher)
		if err != nil {
			return err
		}
		if ! executable {
			if err := os.Chmod(launcher, 0755); err != nil {
				return err
			}
		}
	}

	if Debug {
		fmt.Printf("Launcher: %s\n", launcher)
	}

	if len(args) == 2 && args[1] == "version" {
		fmt.Printf("opendex-launcher %s-%s\n", build.Version, build.GitCommit[:7])
	}

	if err := t.Run(launcher, args[1:]...); err != nil {
		return err
	}

	return nil
}
