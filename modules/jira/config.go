package jira

import (
	// Stdlib
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"

	// Vendor
	"github.com/bgentry/speakeasy"
	"github.com/fatih/color"
	"github.com/salsita/go-jira/v2/jira"
)

// Configuration ===============================================================

type moduleConfig struct {
	ServerURL  *url.URL
	Username   string
	Password   string
	ProjectKey string
}

func loadConfig() (*moduleConfig, error) {
	spec := newConfigSpec()
	if err := loader.LoadConfig(spec); err != nil {
		return nil, err
	}

	// Parse the server URL string.
	serverURL, _ := url.Parse(spec.local.ServerURL)

	// Get the credentials.
	creds := credentialsForServerURL(spec.global.Credentials, serverURL)
	if creds == nil {
		return nil, fmt.Errorf("credentials missing for JIRA server URL: %v", serverURL)
	}

	return &moduleConfig{
		ServerURL:  serverURL,
		Username:   creds.Username,
		Password:   creds.Password,
		ProjectKey: spec.local.ProjectKey,
	}, nil
}

// Configuration spec ----------------------------------------------------------

type configSpec struct {
	global *GlobalConfig
	local  *LocalConfig
}

func newConfigSpec() *configSpec {
	return &configSpec{}
}

// ConfigKey is a part of loader.ModuleConfigSpec interface.
func (spec *configSpec) ConfigKey() string {
	return ModuleId
}

// ModuleKind is a part of loader.ModuleConfigSpec interface.
func (spec *configSpec) ModuleKind() loader.ModuleKind {
	return ModuleKind
}

// GlobalConfig is a part of loader.ModuleConfigSpec interface.
func (spec *configSpec) GlobalConfig() loader.ConfigContainer {
	spec.global = &GlobalConfig{spec: spec}
	return spec.global
}

// LocalConfig is a part of loader.ModuleConfigSpec interface.
func (spec *configSpec) LocalConfig() loader.ConfigContainer {
	if spec.local == nil {
		spec.local = &LocalConfig{spec: spec}
	}
	return spec.local
}

// Global configuration --------------------------------------------------------

type GlobalConfig struct {
	spec *configSpec

	validateCalled bool
	validateResult error

	currentServerURL *url.URL
	currentCreds     *Credentials

	Credentials []*Credentials `json:"credentials"`
}

type Credentials struct {
	ServerPrefix string `json:"server_prefix"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

func (global *GlobalConfig) PromptUserForConfig() error {
	// Make sure Validate is called.
	// We need this for its side-effects for potentially
	// filling the existing server URL.
	global.Validate("")

	var (
		serverURL = global.currentServerURL
		creds     = global.currentCreds
	)

	// A few helper functions.
	invalid := func() {
		fmt.Println()
		color.Yellow("You inserted a value that is not valid, please try again!")
		fmt.Println()

		serverURL = nil
		creds = nil
	}

	checkPromptError := func(err error) error {
		if err == prompt.ErrCanceled {
			prompt.PanicCancel()
		}
		return err
	}

	// Prompt for necessary information.
	for {
		// Prompt for the server URL if necessary.
		if serverURL == nil {
			answer, err := prompt.Prompt("Insert the address of the chosen JIRA server: ")
			if err != nil {
				return checkPromptError(err)
			}
			serverURL, err = url.Parse(answer)
			if err != nil {
				invalid()
				continue
			}
			fmt.Println()
		}

		// Try to find matching credentials.
		fmt.Println("Checking available JIRA credentials ...")
		if creds == nil {
			creds = credentialsForServerURL(global.Credentials, serverURL)
			if creds != nil {
				fmt.Printf("Matching JIRA credentials found (username = %v)\n", creds.Username)
				break
			}
		}

		// In case there are no matching credentials, prompt for it.
		var err error
		fmt.Println("No matching JIRA credential found.")
		fmt.Println("Please insert the credentials for the given JIRA server.")
		fmt.Println()
		username, err := prompt.Prompt("==> Username: ")
		if err != nil {
			return checkPromptError(err)
		}
		password, err := speakeasy.Ask("==> Password: ")
		if err != nil {
			return err
		}
		if password == "" {
			prompt.PanicCancel()
		}

		// Connect to the JIRA server to verify the credentials.
		fmt.Println()
		task := "Connect to given JIRA server to validate the information"
		log.Run(task)

		client := newClient(&moduleConfig{
			ServerURL: serverURL,
			Username:  username,
			Password:  password,
		})
		if _, _, err = client.Myself.Get(); err != nil {
			log.Fail(task)
			invalid()
			continue
		}

		// Set the new credentials.
		creds = &Credentials{
			ServerPrefix: path.Join(serverURL.Host, serverURL.Path),
			Username:     username,
			Password:     password,
		}

		global.Credentials = append(global.Credentials, creds)
		break
	}

	// Store the received information in the cache and return.
	// This fields are used by the local config prompt later.
	global.currentServerURL = serverURL
	global.currentCreds = creds
	return nil
}

func (global *GlobalConfig) Validate(sectionPath string) (err error) {
	// Make sure Validate() is called only once.
	if global.validateCalled {
		return global.validateResult
	}
	defer func() {
		global.validateCalled = true
		global.validateResult = err
	}()

	// We need to manually read the local config, then find and unmarshal local config
	// so that we can use the server URL potentially stored there to search for matching
	// credentials. In case such credentials are found, global config is considered valid.
	local, err := config.ReadLocalConfig()
	if err != nil {
		return err
	}

	record, err := local.ConfigRecord(ModuleId)
	if err != nil {
		return err
	}

	c := LocalConfig{spec: global.spec}
	if err := config.Unmarshal(record.RawConfig, &c); err != nil {
		return err
	}
	// Store the unmarshalled local config for later use.
	// That is why LocalConfig.Unmarshal is not doing anything.
	global.spec.local = &c

	serverURL, err := url.Parse(c.ServerURL)
	if err != nil {
		return err
	}
	// Store the current server URL for later use.
	global.currentServerURL = serverURL

	creds := credentialsForServerURL(global.Credentials, serverURL)
	if creds == nil {
		return fmt.Errorf("not JIRA credentials for given server URL: %v", serverURL)
	}
	// Store the credentials for later use.
	global.currentCreds = creds

	return nil
}

// credentialsForServerURL finds the credentials matching the given server URL the best,
// i.e. the associated server prefix is the longest available.
func credentialsForServerURL(credList []*Credentials, serverURL *url.URL) *Credentials {
	var (
		longestMatch *Credentials
		prefix       = path.Join(serverURL.Host, serverURL.Path)
	)
	for _, cred := range credList {
		// Drop the scheme.
		credServerURL, err := url.Parse(cred.ServerPrefix)
		if err != nil {
			continue
		}
		credPrefix := path.Join(credServerURL.Host, credServerURL.Path)

		// Continue if the prefixes do not match at all.
		if !strings.HasPrefix(credPrefix, prefix) {
			continue
		}
		// Replace only if the current server prefix is longer.
		if longestMatch == nil || len(longestMatch.ServerPrefix) < len(cred.ServerPrefix) {
			longestMatch = cred
		}
	}
	return longestMatch
}

// Local configuration -------------------------------------------------------

type LocalConfig struct {
	spec *configSpec

	ServerURL  string `json:"server_url"`
	ProjectKey string `json:"project_key"`
}

func (local *LocalConfig) PromptUserForConfig() error {
	// Prompt the user for the project key.
	client := newClient(&moduleConfig{
		ServerURL: local.spec.global.currentServerURL,
		Username:  local.spec.global.currentCreds.Username,
		Password:  local.spec.global.currentCreds.Password,
	})

	task := "Fetch available JIRA projects"
	log.Run(task)

	projects, _, err := client.Projects.List()
	if err != nil {
		return errs.NewError(task, err)
	}
	sort.Sort(jiraProjects(projects))

	fmt.Println()
	fmt.Println("Available JIRA projects:")
	fmt.Println()
	for i, project := range projects {
		fmt.Printf("  [%v] %v (%v)\n", i+1, project.Name, project.Key)
	}
	fmt.Println()
	fmt.Println("Choose the project to associate this repository with.")
	index, err := prompt.PromptIndex("Project number: ", 1, len(projects))
	if err != nil {
		if err == prompt.ErrCanceled {
			prompt.PanicCancel()
		}
		return err
	}
	projectKey := projects[index-1].Key

	// Store the results.
	local.ServerURL = local.spec.global.currentServerURL.String()
	local.ProjectKey = projectKey
	return nil
}

func (local *LocalConfig) Unmarshal(unmarshal func(interface{}) error) error {
	// The config is already cached (check Global.Validate)
	return nil
}

func (local *LocalConfig) Validate(sectionPath string) error {
	// Make sure all fields are filled.
	if err := config.EnsureValueFilled(local, sectionPath); err != nil {
		return err
	}

	// Make sure the server URL string is a valid URL.
	_, err := url.Parse(local.ServerURL)
	if err != nil {
		return &config.ErrKeyInvalid{sectionPath + ".server_url", local.ServerURL}
	}

	return nil
}

// Implement sort.Interface to sort projects alphabetically by key.
type jiraProjects []*jira.Project

func (ps jiraProjects) Len() int {
	return len(ps)
}

func (ps jiraProjects) Less(i, j int) bool {
	return ps[i].Key < ps[j].Key
}

func (ps jiraProjects) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}
