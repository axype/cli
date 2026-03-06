package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/urfave/cli/v3"
)

const (
	ClientRepoURL  = "https://github.com/axype/paste-template.git"
	DefaultRepoURL = "https://github.com/axype/paste-template-noclient.git"

	ApiURL = "https://axype.darkceius.dev/api/"
)

var (
	RequiredBinaries = []string{"git", "rokit"}
	ResourceLinks    = map[string]string{
		"rokit": "https://github.com/rojo-rbx/rokit/releases/latest",
	}

	CliTheme         = huh.ThemeCatppuccin()
	ProjectNameRegex = regexp.MustCompile("[^a-zA-Z0-9-]")
)

func getAxypePath(createDir bool) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		showError("Failed to find home directory")
		return ""
	}

	axypePath := filepath.Join(homeDir, ".axype")
	_, err = os.Stat(axypePath)

	if createDir && errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(axypePath, 0o700); err != nil {
			showError("Failed to create `.axype` folder >" + err.Error())
			return ""
		}
	} else if err != nil {
		return ""
	}

	return axypePath
}

func getAuthToken() string {
	axypePath := getAxypePath(false)
	if axypePath == "" {
		return ""
	}

	secretContent, err := os.ReadFile(filepath.Join(axypePath, "secret"))
	if err != nil {
		showError("Could not read secret file" + err.Error())
		return ""
	}

	return string(secretContent)
}

func setAuthToken(token string) bool {
	axypePath := getAxypePath(true)
	if axypePath == "" {
		return false
	}

	secretPath := filepath.Join(axypePath, "secret")
	if err := os.WriteFile(secretPath, []byte(token), 0o600); err != nil {
		showError("Failed to write secret file")
		return false
	}

	return true
}

func initCommand(ctx context.Context, cmd *cli.Command) error {
	// required binaries
	for _, req := range RequiredBinaries {
		if !hasExecutable(req) {
			showError(req + " is required to create a project!")

			if link, found := ResourceLinks[req]; found {
				showError("> Download it here: " + link)
			}

			return nil
		}
	}

	// init config
	rawProjectName := ""
	useGitRepo := true
	useClient := false
	useClientLuau := false

	if err := huh.NewForm(
		huh.NewGroup(huh.NewInput().Title("What will the project be named?").Value(&rawProjectName)),
		huh.NewGroup(huh.NewConfirm().Title("Initialize git repository?").Value(&useGitRepo)),
		huh.NewGroup(huh.NewConfirm().Title("Add client-side support? > `src/client`").Value(&useClient)),
	).WithTheme(CliTheme).Run(); err != nil {
		return nil
	}

	if useClient {
		if err := huh.NewConfirm().Title("Use Luau syntax for client? (not recommended)").Value(&useClientLuau).WithTheme(CliTheme).Run(); err != nil {
			return nil
		}
	}

	// paths
	projectName := strings.TrimSpace(ProjectNameRegex.ReplaceAllString(rawProjectName, ""))
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	targetPath := filepath.Join(cwd, projectName)

	// final confirmation
	proceed := false

	if err := huh.NewConfirm().Title("Are you sure you want to initialize the repository in " + targetPath).Value(&proceed).WithTheme(CliTheme).Run(); err != nil {
		return nil
	}

	if !proceed {
		showError("Cancelled!")
		return nil
	}

	if _, err := os.Stat(targetPath); err == nil {
		showError("Couldn't init because a file with the same name exists in this directory!")
		return nil
	}

	// cool spinner
	spinnerContext, stopSpinner := context.WithCancel(ctx)
	spinner := spinner.New().Title("Creating project...")
	go func() {
		spinner.Context(spinnerContext).Style(CliTheme.Group.Title).Run()
	}()
	defer stopSpinner()

	// cloning repo
	{
		spinner.Title("Cloning template repo...")

		var repoURL string
		if useClient {
			repoURL = ClientRepoURL
		} else {
			repoURL = DefaultRepoURL
		}

		if err := exec.Command("git", "clone", repoURL, projectName).Run(); err != nil {
			showError("Git clone command failed >", err.Error())
			return nil
		}
	}

	// updating rojo json file
	spinner.Title("Updating project files...")

	{
		defaultProject, err := os.Open(filepath.Join(targetPath, "default.project.json"))
		if err != nil {
			panic("Failed to open default.project.json")
		}

		defer defaultProject.Close()

		var content map[string]any
		if err := json.NewDecoder(defaultProject).Decode(&content); err != nil {
			panic("Failed to parse default.project.json")
		}

		content["name"] = projectName

		encoder := json.NewEncoder(defaultProject)
		encoder.SetIndent("", "\t\t")
		encoder.Encode(content)
	}

	// swapping client script extension
	if useClient && useClientLuau {
		os.Rename(
			filepath.Join(targetPath, "src", "client", "init.lua"),
			filepath.Join(targetPath, "src", "client", "init.luau"),
		)
	}

	// updating README
	{
		readMePath := filepath.Join(targetPath, "README.md")
		readMe, err := os.ReadFile(readMePath)
		if err != nil {
			panic("could not read README file (crazy)")
		}

		toApply := strings.ReplaceAll(string(readMe), "# paste-template", "# "+projectName)
		os.WriteFile(readMePath, []byte(toApply), 0o664)
	}

	// initializing Rokit
	{
		spinner.Title("Initializing Rokit...")
		execAtCwd(targetPath, "rokit", "install")
	}

	// initializing Git
	{
		spinner.Title("Deleting existing git directory...")

		gitPath := filepath.Join(targetPath, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			os.RemoveAll(gitPath)
		}

		if useGitRepo {
			spinner.Title("Creating empty git project...")

			execAtCwd(targetPath, "git", "init")
			execAtCwd(targetPath, "git", "add", ".")
			execAtCwd(targetPath, "git", "commit", "-m", "feat: init paste codebase")
		}
	}

	stopSpinner()
	showSuccess("Successfully created project " + projectName)

	openInVSC := true
	if err := huh.NewConfirm().Title("Open project in VSCode?").Value(&openInVSC).WithTheme(CliTheme).Run(); err != nil {
		panic(err)
	}

	if openInVSC {
		execAtCwd(targetPath, "code", ".", "./src/server/init.luau")
	} else {
		showTitle("★ You can always find it in " + targetPath)
	}

	return nil
}

func setTokenCommand(ctx context.Context, cmd *cli.Command) error {
	token := cmd.StringArg("token")
	if token == "" {
		showError("`token` argument is required")
		return nil
	}

	if setAuthToken(token) {
		showSuccess("Successfully updated authentication token!")
	}

	return nil
}

func clearTokenCommand(ctx context.Context, cmd *cli.Command) error {
	if setAuthToken("") {
		showSuccess("Successfully cleared your authentication!")
	}

	return nil
}

func clearCommand(ctx context.Context, cmd *cli.Command) error {
	axypePath := getAxypePath(false)
	if axypePath == "" {
		showError("Already cleaned up!")
		return nil
	}

	if err := os.RemoveAll(axypePath); err != nil {
		showError("Failed to remove directory")
		return nil
	}

	showSuccess("Successfully removed .axype folder!")
	return nil
}

func publishCommand(ctx context.Context, cmd *cli.Command) error {
	name := cmd.StringArg("name")
	file := cmd.StringArg("file")
	if name == "" {
		showError("`name` argument is required!")
		return nil
	}

	// authentication token
	authToken := getAuthToken()
	if authToken == "" {
		showError("Missing authentication")
		return nil
	}

	// reading file
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	source, err := os.ReadFile(filepath.Join(cwd, file))
	if err != nil {
		showError("Unknown input file " + file)
		return nil
	}

	// spinner
	spinnerContext, stopSpinner := context.WithCancel(ctx)
	spinner := spinner.New().Title("Publishing paste \"" + name + "\"")
	go func() {
		spinner.Context(spinnerContext).Style(CliTheme.Group.Title).Run()
	}()
	defer stopSpinner()

	// preparing request
	reqBody, err := json.Marshal(map[string]string{
		"source": string(source),
	})
	if err != nil {
		panic("failed to create json body")
	}

	req, err := http.NewRequest("POST", ApiURL+"setSource", bytes.NewBuffer(reqBody))
	if err != nil {
		panic("failed to create request")
	}

	req.Header.Set("script", name)
	req.Header.Set("authentication", strings.TrimSpace(authToken))
	req.Header.Set("Content-Type", "application/json")

	// executing request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		showError("Failed to publish source!")
		showError(err.Error())
		return nil
	}

	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		showSuccess("Successfully updated the source of " + name)
		return nil
	}

	showError("Failed to update source!")
	body, _ := io.ReadAll(response.Body)
	showError("Error >", string(body))

	return nil
}

func main() {
	cmd := &cli.Command{
		Name:    "axype",
		Version: "v1.0.0",
		Usage:   "CLI for the Axype pasteloader",

		Commands: []*cli.Command{
			{
				Name:        "init",
				Aliases:     []string{"i", "create"},
				Description: "Initializes a template Axype project",

				Action: initCommand,
			},
			{
				Name:        "set-token",
				Aliases:     []string{"st"},
				Description: "Updates your local Axype API token; it's used by the `publish` command",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "token",
						UsageText: "API token",
					},
				},

				Action: setTokenCommand,
			},
			{
				Name:        "clear-token",
				Description: "Removes your local Axype API token",

				Action: clearTokenCommand,
			},
			{
				Name:        "clear",
				Aliases:     []string{"uninstall"},
				Description: "Removes the home directory created by the CLI",

				Action: clearCommand,
			},
			{
				Name:        "publish",
				Aliases:     []string{"p"},
				Description: "Publishes the current paste source to the Axype API",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "name",
						UsageText: "The paste name",
					},
					&cli.StringArg{
						Name:      "file",
						Value:     "output/server.luau",
						UsageText: "The lua file to upload to the server",
					},
				},

				Action: publishCommand,
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
