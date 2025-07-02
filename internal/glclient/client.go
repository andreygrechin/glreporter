package glclient

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/andreygrechin/glreporter/internal/worker"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// ErrInvalidGroupID is returned when an invalid group ID is provided.
var ErrInvalidGroupID = errors.New("invalid group ID")

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
func (c *Client) GetGroupsRecursively(groupID int) ([]*gitlab.Group, error) {
	if groupID < 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidGroupID, groupID)
	}

	// If no group ID is provided, fetch all accessible groups
	if groupID == 0 {
		return c.GetAllGroups()
	}

	if c.debug {
		fmt.Printf("DEBUG: starting recursive group fetch for group ID %d\n", groupID)
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
func (c *Client) GetProjectsRecursively(groupID int) ([]*gitlab.Project, error) {
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
			groupProjects, err := c.fetchProjectsForGroupWithDedupe(group.ID)
			if err != nil {
				if c.debug {
					fmt.Printf("DEBUG: error fetching projects for group %d: %v\n", group.ID, err)
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
func (c *Client) GetGroupAccessTokens(groupID int, includeInactive bool) ([]*GroupAccessTokenWithGroup, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidGroupID, groupID)
	}

	// Get the group information first
	group, _, err := c.client.Groups.GetGroup(groupID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get group info: %w", err)
	}

	return c.listTokensForGroup(groupID, group, includeInactive)
}

// GetGroupAccessTokensRecursively fetches all access tokens for all groups within a group and its subgroups.
func (c *Client) GetGroupAccessTokensRecursively(
	groupID int,
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

		groupID := group.ID
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
func (c *Client) GetProjectAccessTokens(projectID int, includeInactive bool) ([]*ProjectAccessTokenWithProject, error) {
	if c.debug {
		fmt.Printf("DEBUG: fetching project access tokens for project %d\n", projectID)
	}

	// First, get the project information
	project, _, err := c.client.Projects.GetProject(projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get project %d: %w", projectID, err)
	}

	tokens, err := c.listTokensForProject(projectID, project, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens for project %d: %w", projectID, err)
	}

	return tokens, nil
}

// GetProjectAccessTokensRecursively fetches all access tokens for all projects within a group and its subgroups.
func (c *Client) GetProjectAccessTokensRecursively(
	groupID int,
	includeInactive bool,
) ([]*ProjectAccessTokenWithProject, error) {
	if c.debug {
		fmt.Printf("DEBUG: starting recursive project access token fetch for group ID %d\n", groupID)
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

		projectID := project.ID

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
func (c *Client) GetPipelineTriggers(projectID int) ([]*PipelineTriggerWithProject, error) {
	if c.debug {
		fmt.Printf("DEBUG: fetching pipeline trigger tokens for project ID %d\n", projectID)
	}

	// First get project info
	project, _, err := c.client.Projects.GetProject(projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return c.listTriggersForProject(projectID, project)
}

// GetPipelineTriggersRecursively fetches all pipeline triggers for all projects within a group and its subgroups.
func (c *Client) GetPipelineTriggersRecursively(groupID int) ([]*PipelineTriggerWithProject, error) {
	if c.debug {
		fmt.Printf("DEBUG: starting recursive pipeline trigger tokens fetch for group ID %d\n", groupID)
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

		projectID := project.ID

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
func (c *Client) GetProjectVariables(projectID int) ([]*ProjectVariableWithProject, error) {
	if c.debug {
		fmt.Printf("DEBUG: fetching project variables for project %d\n", projectID)
	}

	// First, get the project information
	project, _, err := c.client.Projects.GetProject(projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get project %d: %w", projectID, err)
	}

	variables, err := c.listVariablesForProject(projectID, project)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables for project %d: %w", projectID, err)
	}

	return variables, nil
}

// GetProjectVariablesRecursively fetches all CI/CD variables for all projects within a group and its subgroups.
func (c *Client) GetProjectVariablesRecursively(groupID int) ([]*ProjectVariableWithProject, error) {
	if c.debug {
		fmt.Printf("DEBUG: starting recursive project variables fetch for group ID %d\n", groupID)
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

		projectID := project.ID
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

func (c *Client) listTokensForGroup(
	groupID int,
	group *gitlab.Group,
	includeInactive bool,
) ([]*GroupAccessTokenWithGroup, error) {
	if c.debug {
		fmt.Printf("DEBUG: fetching group access tokens for group ID %d\n", groupID)
	}

	opt := &gitlab.ListGroupAccessTokensOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPageSize,
			Page:    1,
		},
	}

	var state gitlab.AccessTokenState
	if !includeInactive {
		state = "active"
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
			fmt.Printf("DEBUG: fetched %d group access tokens for group %d\n", len(tokens), groupID)
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

func (c *Client) fetchSubgroups(parentID int, groups *[]*gitlab.Group, mu *sync.Mutex, wg *sync.WaitGroup) {
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
				fmt.Printf("DEBUG: error fetching subgroups for group %d: %v\n", parentID, err)
			}

			return
		}

		mu.Lock()
		*groups = append(*groups, subgroups...)
		mu.Unlock()

		if c.debug {
			fmt.Printf("DEBUG: fetched %d subgroups for group %d\n", len(subgroups), parentID)
		}

		for _, subgroup := range subgroups {
			wg.Add(1)

			subgroupID := subgroup.ID

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

func (c *Client) fetchProjectsForGroupWithDedupe(groupID int) ([]*gitlab.Project, error) {
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
				fmt.Printf("DEBUG: error fetching projects for group %d: %v\n", groupID, err)
			}

			return allProjects, fmt.Errorf("failed to fetch projects for group %d: %w", groupID, err)
		}

		allProjects = append(allProjects, groupProjects...)

		if c.debug {
			fmt.Printf("DEBUG: fetched %d projects for group %d\n", len(groupProjects), groupID)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allProjects, nil
}

func (c *Client) fetchTokensForGroup(
	groupID int,
	group *gitlab.Group,
	includeInactive bool,
	tokens *[]*GroupAccessTokenWithGroup,
	mu *sync.Mutex,
) {
	groupTokens, err := c.listTokensForGroup(groupID, group, includeInactive)
	if err != nil {
		if c.debug {
			fmt.Printf("DEBUG: error fetching tokens for group %d: %v\n", groupID, err)
		}

		return
	}

	mu.Lock()
	*tokens = append(*tokens, groupTokens...)
	mu.Unlock()
}

func (c *Client) listTokensForProject(
	projectID int,
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
		state := "active"
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
			fmt.Printf("DEBUG: fetched %d project access tokens for project %d\n", len(tokens), projectID)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allTokens, nil
}

func (c *Client) fetchTokensForProject(
	projectID int,
	project *gitlab.Project,
	includeInactive bool,
	tokens *[]*ProjectAccessTokenWithProject,
	mu *sync.Mutex,
) {
	projectTokens, err := c.listTokensForProject(projectID, project, includeInactive)
	if err != nil {
		if c.debug {
			fmt.Printf("DEBUG: error fetching tokens for project %d: %v\n", projectID, err)
		}

		return
	}

	mu.Lock()
	*tokens = append(*tokens, projectTokens...)
	mu.Unlock()
}

func (c *Client) listTriggersForProject(
	projectID int,
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
			fmt.Printf("DEBUG: fetched %d pipeline trigger tokens for project %d\n", len(triggers), projectID)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allTriggers, nil
}

func (c *Client) fetchTriggersForProject(
	projectID int,
	project *gitlab.Project,
	triggers *[]*PipelineTriggerWithProject,
	mu *sync.Mutex,
) {
	projectTriggers, err := c.listTriggersForProject(projectID, project)
	if err != nil {
		if c.debug {
			fmt.Printf("DEBUG: error fetching trigger tokens for project %d: %v\n", projectID, err)
		}

		return
	}

	mu.Lock()
	*triggers = append(*triggers, projectTriggers...)
	mu.Unlock()
}

func (c *Client) listVariablesForProject(
	projectID int,
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
			fmt.Printf("DEBUG: fetched %d project variables for project %d\n", len(variables), projectID)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allVariables, nil
}

func (c *Client) fetchVariablesForProject(
	projectID int,
	project *gitlab.Project,
	variables *[]*ProjectVariableWithProject,
	mu *sync.Mutex,
) {
	projectVariables, err := c.listVariablesForProject(projectID, project)
	if err != nil {
		if c.debug {
			fmt.Printf("DEBUG: error fetching variables for project %d: %v\n", projectID, err)
		}

		return
	}

	mu.Lock()
	*variables = append(*variables, projectVariables...)
	mu.Unlock()
}
