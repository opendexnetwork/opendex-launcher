package core

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/opendexnetwork/opendex-launcher/utils"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

var (
	ErrNotFound = errors.New("not found")

	ReleaseRef = regexp.MustCompile(`^\d{2}\.\d{2}\.\d{2}.*$`)
)

type GithubClient struct {
	Client      *http.Client
	Logger      *logrus.Entry
	AccessToken string
}

func NewGithubClient(accessToken string) *GithubClient {
	return &GithubClient{
		Client:      http.DefaultClient,
		Logger:      logrus.NewEntry(logrus.StandardLogger()).WithField("name", "github"),
		AccessToken: accessToken,
	}
}

func (t *GithubClient) getResponseError(resp *http.Response) error {
	var err error
	if resp.StatusCode != http.StatusOK {
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return fmt.Errorf("decode: %w", err)
		}
		return errors.New(result["message"].(string))
	}
	return nil
}

func (t *GithubClient) doGet(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := t.getResponseError(resp); err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (t *GithubClient) GetHeadCommit(branch string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/opendexnetwork/opendex-docker/commits/%s", branch)
	body, err := t.doGet(url)
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}
	return result["sha"].(string), nil
}

type Artifact struct {
	Name               string `json:"name"`
	SizeInBytes        uint   `json:"size_in_bytes"`
	ArchiveDownloadUrl string `json:"archive_download_url"`
}

type ArtifactList struct {
	TotalCount uint       `json:"total_count"`
	Artifacts  []Artifact `json:"artifacts"`
}

type WorkflowRun struct {
	Id         uint   `json:"id"`
	CreatedAt  string `json:"created_at"`
	HeadBranch string `json:"head_branch"`
	HeadSha    string `json:"head_sha"`
}

type WorkflowRunList struct {
	TotalCount   uint          `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

func (t *GithubClient) getWorkflowDownloadUrl(runId uint) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/opendexnetwork/opendex-docker/actions/runs/%d/artifacts", runId)
	body, err := t.doGet(url)
	if err != nil {
		return "", err
	}
	var result ArtifactList
	err = json.Unmarshal(body, &result)
	for _, artifact := range result.Artifacts {
		name := fmt.Sprintf("%s-amd64", runtime.GOOS)
		if name == artifact.Name {
			return artifact.ArchiveDownloadUrl, nil
		}
	}
	return "", ErrNotFound
}

func (t *GithubClient) getLastRunOfBranch(branch string, commit string) (*WorkflowRun, error) {
	url := fmt.Sprintf("https://api.github.com/repos/opendexnetwork/opendex-docker/actions/workflows/build.yml/runs?branch=%s", branch)
	body, err := t.doGet(url)
	if err != nil {
		return nil, err
	}
	var result WorkflowRunList
	err = json.Unmarshal(body, &result)
	if len(result.WorkflowRuns) == 0 {
		return nil, ErrNotFound
	}
	run := &result.WorkflowRuns[0]
	if run.HeadSha != commit {
		return nil, ErrNotFound
	}
	return run, nil
}

func (t *GithubClient) getDownloadUrl(branch string, commit string) (string, error) {
	var url string

	if ReleaseRef.Match([]byte(branch)) {
		url = fmt.Sprintf("https://github.com/opendexnetwork/opendex-docker/releases/download/%s/launcher-%s-%s.zip", branch, runtime.GOOS, runtime.GOARCH)
	} else {
		run, err := t.getLastRunOfBranch(branch, commit)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return "", fmt.Errorf("no launcher build for commit %s (The branch \"%s\" does not have a binary launcher)", commit, branch)
			}
			return "", err
		}

		url, err = t.getWorkflowDownloadUrl(run.Id)
		if err != nil {
			return "", nil
		}
		t.Logger.Debugf("Download launcher.zip from %s", url)
	}

	return url, nil
}

func (t *GithubClient) ensureCommitDir(commit string, launcherVersionsDir string) (string, error) {
	commitDir := filepath.Join(launcherVersionsDir, commit)

	exists, err := utils.FileExists(commitDir)
	if err != nil {
		return "", err
	}

	if ! exists {
		if err := os.Mkdir(commitDir, 0755); err != nil {
			return "", err
		}
	}

	return commitDir, nil
}

func (t *GithubClient) downloadLauncher(url string, commit string, commitDir string) error {
	var err error

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := os.Chdir(commitDir); err != nil {
		return err
	}
	defer os.Chdir(wd)

	if err = t.downloadFile(url, "launcher.zip"); err != nil {
		return err
	}

	if err = t.unzip("launcher.zip"); err != nil {
		return err
	}

	return nil
}

func (t *GithubClient) DownloadLatestBinary(branch string, commit string, launcherVersionsDir string) error {
	var err error
	var url string

	if url, err = t.getDownloadUrl(branch, commit); err != nil {
		return err
	}
	if Debug {
		fmt.Printf("Download: %s\n", url)
	}

	commitDir, err := t.ensureCommitDir(commit, launcherVersionsDir)
	if err != nil {
		return err
	}

	if err = t.downloadLauncher(url, commit, commitDir); err != nil {
		return err
	}

	return nil
}

func (t *GithubClient) unzip(file string) error {
	var filenames []string

	r, err := zip.OpenReader(file)
	if err != nil {
		return fmt.Errorf("open reader: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		t.Logger.Debugf("Extracting %s", f.Name)

		fpath := f.Name

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("mkdir all: %w", err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		_ = outFile.Close()
		_ = rc.Close()

		if err != nil {
			return fmt.Errorf("copy: %w", err)
		}
	}
	return nil
}

func (t *GithubClient) downloadFile(url string, file string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Add("Authorization", "token "+t.AccessToken)
	resp, err := t.Client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read all: %w", err)
		}
		return errors.New(string(body))
	}

	out, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return nil
}
