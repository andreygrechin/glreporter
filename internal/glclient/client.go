package glclient

import (
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/andreygrechin/glreporter/internal/worker"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GroupAccessTokenWithGroup represents a group access token with associated group information.
type GroupAccessTokenWithGroup struct {
	*gitlab.GroupAccessToken
	GroupName   string `json:"group_name"`
	GroupPath   string `json:"group_path"`
	GroupWebURL string `json:"group_web_url"`
}

// ProjectAccessTokenWithProject represents a project access token with associated project information.
type ProjectAccessTokenWithProject struct {
	*gitlab.ProjectAccessToken
	ProjectName      string `json:"project_name"`
	ProjectPath      string `json:"project_path"`
	ProjectNamespace string `json:"project_namespace"`
	ProjectWebURL    string `json:"project_web_url"`
}

// PipelineTriggerWithProject represents a pipeline trigger with associated project information.
type PipelineTriggerWithProject struct {
	*gitlab.PipelineTrigger
	ProjectName      string `json:"project_name"`
	ProjectPath      string `json:"project_path"`
	ProjectNamespace string `json:"project_namespace"`
	ProjectWebURL    string `json:"project_web_url"`
}

// ProjectVariableWithProject represents a project variable with associated project information.
type ProjectVariableWithProject struct {
	*gitlab.ProjectVariable
	ProjectName      string `json:"project_name"`
	ProjectPath      string `json:"project_path"`
	ProjectNamespace string `json:"project_namespace"`
	ProjectWebURL    string `json:"project_web_url"`
}

// GroupVariableWithGroup represents a GitLab group variable with additional group information.
type GroupVariableWithGroup struct {
	*gitlab.GroupVariable
	GroupName     string `json:"group_name"`
	GroupPath     string `json:"group_path"`
	GroupWebURL   string `json:"group_web_url"`
	GroupFullPath string `json:"group_full_path"`
}

// VariableWithSource represents a CI/CD variable from either a project or group with source identification.
type VariableWithSource struct {
	Key              string `json:"key"`
	Value            string `json:"value"`
	VariableType     string `json:"variable_type"`
	Protected        bool   `json:"protected"`
	Masked           bool   `json:"masked"`
	Hidden           bool   `json:"hidden"`
	Raw              bool   `json:"raw"`
	EnvironmentScope string `json:"environment_scope"`
	Description      string `json:"description"`
	Source           string `json:"source"` // "project" or "group"
	SourceName       string `json:"source_name"`
	SourcePath       string `json:"source_path"`
	SourceWebURL     string `json:"source_web_url"`
	SourceNamespace  string `json:"source_namespace,omitempty"` // only for projects
}

// ProjectVariableWithProjectFiltered represents a project variable without the Value field for security.
type ProjectVariableWithProjectFiltered struct {
	Key              string                   `json:"key"`
	VariableType     gitlab.VariableTypeValue `json:"variable_type"`
	Protected        bool                     `json:"protected"`
	Masked           bool                     `json:"masked"`
	Hidden           bool                     `json:"hidden"`
	Raw              bool                     `json:"raw"`
	EnvironmentScope string                   `json:"environment_scope"`
	Description      string                   `json:"description"`
	ProjectName      string                   `json:"project_name"`
	ProjectPath      string                   `json:"project_path"`
	ProjectNamespace string                   `json:"project_namespace"`
	ProjectWebURL    string                   `json:"project_web_url"`
}

// GroupVariableWithGroupFiltered represents a group variable without the Value field for security.
type GroupVariableWithGroupFiltered struct {
	Key              string                   `json:"key"`
	VariableType     gitlab.VariableTypeValue `json:"variable_type"`
	Protected        bool                     `json:"protected"`
	Masked           bool                     `json:"masked"`
	Hidden           bool                     `json:"hidden"`
	Raw              bool                     `json:"raw"`
	EnvironmentScope string                   `json:"environment_scope"`
	Description      string                   `json:"description"`
	GroupName        string                   `json:"group_name"`
	GroupPath        string                   `json:"group_path"`
	GroupWebURL      string                   `json:"group_web_url"`
	GroupFullPath    string                   `json:"group_full_path"`
}

// VariableWithSourceFiltered represents a variable without the Value field for security.
type VariableWithSourceFiltered struct {
	Key              string `json:"key"`
	VariableType     string `json:"variable_type"`
	Protected        bool   `json:"protected"`
	Masked           bool   `json:"masked"`
	Hidden           bool   `json:"hidden"`
	Raw              bool   `json:"raw"`
	EnvironmentScope string `json:"environment_scope"`
	Description      string `json:"description"`
	Source           string `json:"source"` // "project" or "group"
	SourceName       string `json:"source_name"`
	SourcePath       string `json:"source_path"`
	SourceWebURL     string `json:"source_web_url"`
	SourceNamespace  string `json:"source_namespace,omitempty"` // only for projects
}

// ConvertProjectVariableToUnified converts a ProjectVariableWithProject to VariableWithSource.
func ConvertProjectVariableToUnified(pv *ProjectVariableWithProject) *VariableWithSource {
	return &VariableWithSource{
		Key:              pv.Key,
		Value:            pv.Value,
		VariableType:     string(pv.VariableType),
		Protected:        pv.Protected,
		Masked:           pv.Masked,
		Hidden:           pv.Hidden,
		Raw:              pv.Raw,
		EnvironmentScope: pv.EnvironmentScope,
		Description:      pv.Description,
		Source:           "project",
		SourceName:       pv.ProjectName,
		SourcePath:       pv.ProjectPath,
		SourceWebURL:     pv.ProjectWebURL,
		SourceNamespace:  pv.ProjectNamespace,
	}
}

// ConvertGroupVariableToUnified converts a GroupVariableWithGroup to VariableWithSource.
func ConvertGroupVariableToUnified(gv *GroupVariableWithGroup) *VariableWithSource {
	return &VariableWithSource{
		Key:              gv.Key,
		Value:            gv.Value,
		VariableType:     string(gv.VariableType),
		Protected:        gv.Protected,
		Masked:           gv.Masked,
		Hidden:           gv.Hidden,
		Raw:              gv.Raw,
		EnvironmentScope: gv.EnvironmentScope,
		Description:      gv.Description,
		Source:           "group",
		SourceName:       gv.GroupName,
		SourcePath:       gv.GroupFullPath,
		SourceWebURL:     gv.GroupWebURL,
	}
}

// Client is a wrapper around the GitLab API client that includes a worker pool for concurrent operations.
type Client struct {
	client *gitlab.Client
	pool   *worker.Pool
	debug  bool
}

const (
	maxPageSize   = 50  // Maximum number of items per page
	maxNumWorkers = 100 // Maximum number of concurrent workers
)

// NewClient creates a new GitLab client with a worker pool.
func NewClient(token string, debug bool) (*Client, error) {
	client, err := gitlab.NewClient(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &Client{
		client: client,
		pool:   worker.NewPool(maxNumWorkers),
		debug:  debug,
	}, nil
}

// NewClientWithGitLabClient creates a new client with a provided GitLab client (useful for testing).
func NewClientWithGitLabClient(gitlabClient *gitlab.Client, debug bool) *Client {
	return &Client{
		client: gitlabClient,
		pool:   worker.NewPool(maxNumWorkers),
		debug:  debug,
	}
}

// GetGroupsRecursively fetches all groups and their subgroups starting from a given group ID.
// If groupID is negative, return an error.
func (c *Client) GetGroupsRecursively(groupID string) ([]*gitlab.Group, error) {
	// If no group ID is provided, fetch all accessible groups
	if groupID == "" {
		return c.GetAllGroups()
	}

	if c.debug {
		fmt.Printf("DEBUG: starting recursive group fetch for group ID %s\n", groupID)
	}

	var (
		groups []*gitlab.Group
		mu     sync.Mutex
		wg     sync.WaitGroup
	)

	rootGroup, _, err := c.client.Groups.GetGroup(groupID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get root group: %w", err)
	}

	groups = append(groups, rootGroup)

	wg.Add(1)
	c.pool.Submit(func() {
		defer wg.Done()
		c.fetchSubgroups(groupID, &groups, &mu, &wg)
	})

	wg.Wait()

	if c.debug {
		fmt.Printf("DEBUG: completed group fetch, found %d groups\n", len(groups))
	}

	return groups, nil
}

// GetAllGroups fetches all accessible groups.
func (c *Client) GetAllGroups() ([]*gitlab.Group, error) {
	if c.debug {
		fmt.Printf("DEBUG: fetching all accessible groups\n")
	}

	opt := &gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPageSize,
			Page:    1,
		},
	}

	var allGroups []*gitlab.Group

	for {
		groups, resp, err := c.client.Groups.ListGroups(opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list groups: %w", err)
		}

		allGroups = append(allGroups, groups...)

		if c.debug {
			fmt.Printf("DEBUG: fetched %d groups on page %d\n", len(groups), opt.Page)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	if c.debug {
		fmt.Printf("DEBUG: completed fetching all groups, found %d groups\n", len(allGroups))
	}

	return allGroups, nil
}

// GetProjectsRecursively fetches all projects within a group and its subgroups.
func (c *Client) GetProjectsRecursively(groupID string) ([]*gitlab.Project, error) {
	groups, err := c.GetGroupsRecursively(groupID)
	if err != nil {
		return nil, err
	}

	if c.debug {
		fmt.Printf("DEBUG: starting project fetch for %d groups\n", len(groups))
	}

	// Use a map to track unique projects by ID
	projectMap := make(map[int]*gitlab.Project)

	var (
		wg    sync.WaitGroup
		mapMu sync.Mutex
	)

	for _, group := range groups {
		wg.Add(1)

		c.pool.Submit(func() {
			defer wg.Done()
			// Fetch projects for this group
			groupProjects, err := c.fetchProjectsForGroupWithDedupe(group.FullPath)
			if err != nil {
				if c.debug {
					fmt.Printf("DEBUG: error fetching projects for group %s: %v\n", group.FullPath, err)
				}
				// Continue with other groups even if one fails
				return
			}

			// Add unique projects to the map
			mapMu.Lock()
			for _, project := range groupProjects {
				if _, exists := projectMap[project.ID]; !exists {
					projectMap[project.ID] = project
				}
			}
			mapMu.Unlock()
		})
	}

	wg.Wait()

	// Convert map to slice and sort by ID for deterministic order
	projects := make([]*gitlab.Project, 0, len(projectMap))
	for _, project := range projectMap {
		projects = append(projects, project)
	}

	// Sort projects by ID to ensure deterministic order
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].ID < projects[j].ID
	})

	if c.debug {
		fmt.Printf("DEBUG: completed project fetch, found %d unique projects\n", len(projects))
	}

	return projects, nil
}

// GetGroupAccessTokens fetches all access tokens for a specific group.
func (c *Client) GetGroupAccessTokens(groupID string, includeInactive bool) ([]*GroupAccessTokenWithGroup, error) {
	// Get the group information first
	group, _, err := c.client.Groups.GetGroup(groupID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get group info: %w", err)
	}

	return c.listTokensForGroup(groupID, group, includeInactive)
}

// GetGroupAccessTokensRecursively fetches all access tokens for all groups within a group and its subgroups.
func (c *Client) GetGroupAccessTokensRecursively(
	groupID string,
	includeInactive bool,
) ([]*GroupAccessTokenWithGroup, error) {
	groups, err := c.GetGroupsRecursively(groupID)
	if err != nil {
		return nil, err
	}

	if c.debug {
		fmt.Printf("DEBUG: starting token fetch for %d groups\n", len(groups))
	}

	var (
		tokens []*GroupAccessTokenWithGroup
		mu     sync.Mutex
		wg     sync.WaitGroup
	)

	for _, group := range groups {
		wg.Add(1)

		groupID := strconv.Itoa(group.ID)
		groupCopy := group

		c.pool.Submit(func() {
			defer wg.Done()
			c.fetchTokensForGroup(groupID, groupCopy, includeInactive, &tokens, &mu)
		})
	}

	wg.Wait()

	if c.debug {
		fmt.Printf("DEBUG: completed recursive token fetch, found %d tokens\n", len(tokens))
	}

	return tokens, nil
}

// GetProjectAccessTokens fetches all access tokens for a specific project.
func (c *Client) GetProjectAccessTokens(
	projectID string,
	includeInactive bool,
) ([]*ProjectAccessTokenWithProject, error) {
	if c.debug {
		fmt.Printf("DEBUG: fetching project access tokens for project %s\n", projectID)
	}

	// First, get the project information
	project, _, err := c.client.Projects.GetProject(projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get project %s: %w", projectID, err)
	}

	tokens, err := c.listTokensForProject(projectID, project, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens for project %s: %w", projectID, err)
	}

	return tokens, nil
}

// GetProjectAccessTokensRecursively fetches all access tokens for all projects within a group and its subgroups.
func (c *Client) GetProjectAccessTokensRecursively(
	groupID string,
	includeInactive bool,
) ([]*ProjectAccessTokenWithProject, error) {
	if c.debug {
		fmt.Printf("DEBUG: starting recursive project access token fetch for group ID %s\n", groupID)
	}

	// First, get all projects recursively
	projects, err := c.GetProjectsRecursively(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects recursively: %w", err)
	}

	var (
		allTokens []*ProjectAccessTokenWithProject
		mu        sync.Mutex
		wg        sync.WaitGroup
	)

	for _, project := range projects {
		wg.Add(1)

		projectID := strconv.Itoa(project.ID)

		c.pool.Submit(func() {
			defer wg.Done()
			c.fetchTokensForProject(projectID, project, includeInactive, &allTokens, &mu)
		})
	}

	wg.Wait()

	if c.debug {
		fmt.Printf("DEBUG: completed recursive project access token fetch, found %d tokens\n", len(allTokens))
	}

	return allTokens, nil
}

// GetPipelineTriggers fetches all pipeline triggers for a specific project.
func (c *Client) GetPipelineTriggers(projectID string) ([]*PipelineTriggerWithProject, error) {
	if c.debug {
		fmt.Printf("DEBUG: fetching pipeline trigger tokens for project ID %s\n", projectID)
	}

	// First get project info
	project, _, err := c.client.Projects.GetProject(projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return c.listTriggersForProject(projectID, project)
}

// GetPipelineTriggersRecursively fetches all pipeline triggers for all projects within a group and its subgroups.
func (c *Client) GetPipelineTriggersRecursively(groupID string) ([]*PipelineTriggerWithProject, error) {
	if c.debug {
		fmt.Printf("DEBUG: starting recursive pipeline trigger tokens fetch for group ID %s\n", groupID)
	}

	// First, get all projects recursively
	projects, err := c.GetProjectsRecursively(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects recursively: %w", err)
	}

	var (
		allTriggers []*PipelineTriggerWithProject
		mu          sync.Mutex
		wg          sync.WaitGroup
	)

	for _, project := range projects {
		wg.Add(1)

		projectID := strconv.Itoa(project.ID)

		c.pool.Submit(func() {
			defer wg.Done()
			c.fetchTriggersForProject(projectID, project, &allTriggers, &mu)
		})
	}

	wg.Wait()

	if c.debug {
		fmt.Printf("DEBUG: completed recursive pipeline trigger tokens fetch, found %d trigger tokens\n", len(allTriggers))
	}

	return allTriggers, nil
}

// GetProjectVariables fetches all CI/CD variables for a specific project.
func (c *Client) GetProjectVariables(projectID string) ([]*ProjectVariableWithProject, error) {
	if c.debug {
		fmt.Printf("DEBUG: fetching project variables for project %s\n", projectID)
	}

	// First, get the project information
	project, _, err := c.client.Projects.GetProject(projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get project %s: %w", projectID, err)
	}

	variables, err := c.listVariablesForProject(projectID, project)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables for project %s: %w", projectID, err)
	}

	return variables, nil
}

// GetProjectVariablesRecursively fetches all CI/CD variables for all projects within a group and its subgroups.
func (c *Client) GetProjectVariablesRecursively(groupID string) ([]*ProjectVariableWithProject, error) {
	if c.debug {
		fmt.Printf("DEBUG: starting recursive project variables fetch for group ID %s\n", groupID)
	}

	// First, get all projects recursively
	projects, err := c.GetProjectsRecursively(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects recursively: %w", err)
	}

	var (
		allVariables []*ProjectVariableWithProject
		mu           sync.Mutex
		wg           sync.WaitGroup
	)

	for _, project := range projects {
		wg.Add(1)

		projectID := strconv.Itoa(project.ID)
		projectCopy := project

		c.pool.Submit(func() {
			defer wg.Done()
			c.fetchVariablesForProject(projectID, projectCopy, &allVariables, &mu)
		})
	}

	wg.Wait()

	if c.debug {
		fmt.Printf("DEBUG: completed recursive project variables fetch, found %d variables\n", len(allVariables))
	}

	return allVariables, nil
}

// GetGroupVariables fetches all CI/CD variables for a specific group.
func (c *Client) GetGroupVariables(groupID string) ([]*GroupVariableWithGroup, error) {
	if c.debug {
		fmt.Printf("DEBUG: fetching group variables for group %s\n", groupID)
	}

	// First, get the group information
	group, _, err := c.client.Groups.GetGroup(groupID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get group %s: %w", groupID, err)
	}

	variables, err := c.listVariablesForGroup(groupID, group)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables for group %s: %w", groupID, err)
	}

	return variables, nil
}

// GetGroupVariablesRecursively fetches all group CI/CD variables
// for all groups within a parent group and its subgroups.
func (c *Client) GetGroupVariablesRecursively(groupID string) ([]*GroupVariableWithGroup, error) {
	if c.debug {
		fmt.Printf("DEBUG: starting recursive group variables fetch for group ID %s\n", groupID)
	}

	// First, get all groups recursively
	groups, err := c.GetGroupsRecursively(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups recursively: %w", err)
	}

	var (
		allVariables []*GroupVariableWithGroup
		mu           sync.Mutex
		wg           sync.WaitGroup
	)

	for _, group := range groups {
		wg.Add(1)

		groupID := strconv.Itoa(group.ID)
		groupCopy := group

		c.pool.Submit(func() {
			defer wg.Done()
			c.fetchVariablesForGroup(groupID, groupCopy, &allVariables, &mu)
		})
	}

	wg.Wait()

	if c.debug {
		fmt.Printf("DEBUG: completed recursive group variables fetch, found %d variables\n", len(allVariables))
	}

	return allVariables, nil
}

func (c *Client) listTokensForGroup(
	groupID string,
	group *gitlab.Group,
	includeInactive bool,
) ([]*GroupAccessTokenWithGroup, error) {
	if c.debug {
		fmt.Printf("DEBUG: fetching group access tokens for group ID %s\n", groupID)
	}

	opt := &gitlab.ListGroupAccessTokensOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPageSize,
			Page:    1,
		},
	}

	if !includeInactive {
		state := gitlab.AccessTokenStateActive
		opt.State = &state
	}

	var allTokens []*GroupAccessTokenWithGroup

	for {
		tokens, resp, err := c.client.GroupAccessTokens.ListGroupAccessTokens(groupID, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list group access tokens: %w", err)
		}

		// Wrap each token with group information
		for _, token := range tokens {
			tokenWithGroup := &GroupAccessTokenWithGroup{
				GroupAccessToken: token,
				GroupName:        group.Name,
				GroupPath:        group.FullPath,
				GroupWebURL:      group.WebURL,
			}
			allTokens = append(allTokens, tokenWithGroup)
		}

		if c.debug {
			fmt.Printf("DEBUG: fetched %d group access tokens for group %s\n", len(tokens), groupID)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	if c.debug {
		fmt.Printf("DEBUG: completed token fetch, found %d tokens\n", len(allTokens))
	}

	return allTokens, nil
}

func (c *Client) fetchSubgroups(parentID string, groups *[]*gitlab.Group, mu *sync.Mutex, wg *sync.WaitGroup) {
	opt := &gitlab.ListSubGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPageSize,
			Page:    1,
		},
	}

	for {
		subgroups, resp, err := c.client.Groups.ListSubGroups(parentID, opt)
		if err != nil {
			if c.debug {
				fmt.Printf("DEBUG: error fetching subgroups for group %s: %v\n", parentID, err)
			}

			return
		}

		mu.Lock()
		*groups = append(*groups, subgroups...)
		mu.Unlock()

		if c.debug {
			fmt.Printf("DEBUG: fetched %d subgroups for group %s\n", len(subgroups), parentID)
		}

		for _, subgroup := range subgroups {
			wg.Add(1)

			subgroupID := strconv.Itoa(subgroup.ID)

			c.pool.Submit(func() {
				defer wg.Done()
				c.fetchSubgroups(subgroupID, groups, mu, wg)
			})
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}
}

func (c *Client) fetchProjectsForGroupWithDedupe(groupID string) ([]*gitlab.Project, error) {
	opt := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPageSize,
			Page:    1,
		},
	}

	var allProjects []*gitlab.Project

	for {
		groupProjects, resp, err := c.client.Groups.ListGroupProjects(groupID, opt)
		if err != nil {
			if c.debug {
				fmt.Printf("DEBUG: error fetching projects for group %s: %v\n", groupID, err)
			}

			return allProjects, fmt.Errorf("failed to fetch projects for group %s: %w", groupID, err)
		}

		allProjects = append(allProjects, groupProjects...)

		if c.debug {
			fmt.Printf("DEBUG: fetched %d projects for group %s\n", len(groupProjects), groupID)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allProjects, nil
}

func (c *Client) fetchTokensForGroup(
	groupID string,
	group *gitlab.Group,
	includeInactive bool,
	tokens *[]*GroupAccessTokenWithGroup,
	mu *sync.Mutex,
) {
	groupTokens, err := c.listTokensForGroup(groupID, group, includeInactive)
	if err != nil {
		if c.debug {
			fmt.Printf("DEBUG: error fetching tokens for group %s: %v\n", groupID, err)
		}

		return
	}

	mu.Lock()
	*tokens = append(*tokens, groupTokens...)
	mu.Unlock()
}

func (c *Client) listTokensForProject(
	projectID string,
	project *gitlab.Project,
	includeInactive bool,
) ([]*ProjectAccessTokenWithProject, error) {
	var allTokens []*ProjectAccessTokenWithProject

	opt := &gitlab.ListProjectAccessTokensOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPageSize,
			Page:    1,
		},
	}

	if !includeInactive {
		state := string(gitlab.AccessTokenStateActive)
		opt.State = &state
	}

	for {
		tokens, resp, err := c.client.ProjectAccessTokens.ListProjectAccessTokens(projectID, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list project access tokens: %w", err)
		}

		// Wrap each token with project information
		for _, token := range tokens {
			// Skip inactive tokens if not requested
			if !includeInactive && !token.Active {
				continue
			}

			tokenWithProject := &ProjectAccessTokenWithProject{
				ProjectAccessToken: token,
				ProjectName:        project.Name,
				ProjectPath:        project.PathWithNamespace,
				ProjectNamespace:   project.Namespace.FullPath,
				ProjectWebURL:      project.WebURL,
			}
			allTokens = append(allTokens, tokenWithProject)
		}

		if c.debug {
			fmt.Printf("DEBUG: fetched %d project access tokens for project %s\n", len(tokens), projectID)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allTokens, nil
}

func (c *Client) fetchTokensForProject(
	projectID string,
	project *gitlab.Project,
	includeInactive bool,
	tokens *[]*ProjectAccessTokenWithProject,
	mu *sync.Mutex,
) {
	projectTokens, err := c.listTokensForProject(projectID, project, includeInactive)
	if err != nil {
		if c.debug {
			fmt.Printf("DEBUG: error fetching tokens for project %s: %v\n", projectID, err)
		}

		return
	}

	mu.Lock()
	*tokens = append(*tokens, projectTokens...)
	mu.Unlock()
}

func (c *Client) listTriggersForProject(
	projectID string,
	project *gitlab.Project,
) ([]*PipelineTriggerWithProject, error) {
	var allTriggers []*PipelineTriggerWithProject

	opt := &gitlab.ListPipelineTriggersOptions{
		PerPage: maxPageSize,
		Page:    1,
	}

	for {
		triggers, resp, err := c.client.PipelineTriggers.ListPipelineTriggers(projectID, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list pipeline trigger tokens: %w", err)
		}

		// Wrap each trigger with project information
		for _, trigger := range triggers {
			triggerWithProject := &PipelineTriggerWithProject{
				PipelineTrigger:  trigger,
				ProjectName:      project.Name,
				ProjectPath:      project.PathWithNamespace,
				ProjectNamespace: project.Namespace.FullPath,
				ProjectWebURL:    project.WebURL,
			}
			allTriggers = append(allTriggers, triggerWithProject)
		}

		if c.debug {
			fmt.Printf("DEBUG: fetched %d pipeline trigger tokens for project %s\n", len(triggers), projectID)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allTriggers, nil
}

func (c *Client) fetchTriggersForProject(
	projectID string,
	project *gitlab.Project,
	triggers *[]*PipelineTriggerWithProject,
	mu *sync.Mutex,
) {
	projectTriggers, err := c.listTriggersForProject(projectID, project)
	if err != nil {
		if c.debug {
			fmt.Printf("DEBUG: error fetching trigger tokens for project %s: %v\n", projectID, err)
		}

		return
	}

	mu.Lock()
	*triggers = append(*triggers, projectTriggers...)
	mu.Unlock()
}

func (c *Client) listVariablesForProject(
	projectID string,
	project *gitlab.Project,
) ([]*ProjectVariableWithProject, error) {
	var allVariables []*ProjectVariableWithProject

	opt := &gitlab.ListProjectVariablesOptions{
		PerPage: maxPageSize,
		Page:    1,
	}

	for {
		variables, resp, err := c.client.ProjectVariables.ListVariables(projectID, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list project variables: %w", err)
		}

		// Wrap each variable with project information
		for _, variable := range variables {
			variableWithProject := &ProjectVariableWithProject{
				ProjectVariable:  variable,
				ProjectName:      project.Name,
				ProjectPath:      project.PathWithNamespace,
				ProjectNamespace: project.Namespace.FullPath,
				ProjectWebURL:    project.WebURL,
			}
			allVariables = append(allVariables, variableWithProject)
		}

		if c.debug {
			fmt.Printf("DEBUG: fetched %d project variables for project %s\n", len(variables), projectID)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allVariables, nil
}

func (c *Client) fetchVariablesForProject(
	projectID string,
	project *gitlab.Project,
	variables *[]*ProjectVariableWithProject,
	mu *sync.Mutex,
) {
	projectVariables, err := c.listVariablesForProject(projectID, project)
	if err != nil {
		if c.debug {
			fmt.Printf("DEBUG: error fetching variables for project %s: %v\n", projectID, err)
		}

		return
	}

	mu.Lock()
	*variables = append(*variables, projectVariables...)
	mu.Unlock()
}

func (c *Client) listVariablesForGroup(
	groupID string,
	group *gitlab.Group,
) ([]*GroupVariableWithGroup, error) {
	var allVariables []*GroupVariableWithGroup

	opt := &gitlab.ListGroupVariablesOptions{
		PerPage: maxPageSize,
		Page:    1,
	}

	for {
		variables, resp, err := c.client.GroupVariables.ListVariables(groupID, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list group variables: %w", err)
		}

		// Wrap each variable with group information
		for _, variable := range variables {
			variableWithGroup := &GroupVariableWithGroup{
				GroupVariable: variable,
				GroupName:     group.Name,
				GroupPath:     group.Path,
				GroupWebURL:   group.WebURL,
				GroupFullPath: group.FullPath,
			}
			allVariables = append(allVariables, variableWithGroup)
		}

		if c.debug {
			fmt.Printf("DEBUG: fetched %d group variables for group %s\n", len(variables), groupID)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allVariables, nil
}

func (c *Client) fetchVariablesForGroup(
	groupID string,
	group *gitlab.Group,
	variables *[]*GroupVariableWithGroup,
	mu *sync.Mutex,
) {
	groupVariables, err := c.listVariablesForGroup(groupID, group)
	if err != nil {
		if c.debug {
			fmt.Printf("DEBUG: error fetching variables for group %s: %v\n", groupID, err)
		}

		return
	}

	mu.Lock()
	*variables = append(*variables, groupVariables...)
	mu.Unlock()
}
