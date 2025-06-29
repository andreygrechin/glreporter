package glclient_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/andreygrechin/glreporter/internal/glclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"
	"go.uber.org/mock/gomock"
)

var errAPI = errors.New("API error")

// testClient creates a new test client with mocked GitLab services.
func testClient(t *testing.T) (*glclient.Client, *gitlabtesting.TestClient) {
	t.Helper()
	mockClient := gitlabtesting.NewTestClient(t)
	client := glclient.NewClientWithGitLabClient(mockClient.Client, false)

	return client, mockClient
}

// testClientWithDebug creates a new test client with debug mode enabled.
func testClientWithDebug(t *testing.T) (*glclient.Client, *gitlabtesting.TestClient) {
	t.Helper()
	mockClient := gitlabtesting.NewTestClient(t)
	client := glclient.NewClientWithGitLabClient(mockClient.Client, true)

	return client, mockClient
}

func TestNewClient(t *testing.T) {
	t.Run("creates client successfully", func(t *testing.T) {
		client, err := glclient.NewClient("test-token", false)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("creates client with debug mode", func(t *testing.T) {
		client, err := glclient.NewClient("test-token", true)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})
}

func TestGetGroupsRecursively(t *testing.T) {
	t.Run("fetches single group without subgroups", func(t *testing.T) {
		client, mockClient := testClient(t)

		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "root-group",
			FullPath: "root-group",
			WebURL:   "https://gitlab.com/root-group",
		}

		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		groups, err := client.GetGroupsRecursively(1)
		require.NoError(t, err)
		assert.Len(t, groups, 1)
		assert.Equal(t, rootGroup, groups[0])
	})

	t.Run("fetches group with subgroups recursively", func(t *testing.T) {
		client, mockClient := testClient(t)

		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "root-group",
			FullPath: "root-group",
			WebURL:   "https://gitlab.com/root-group",
		}

		subGroup1 := &gitlab.Group{
			ID:       2,
			Name:     "sub-group-1",
			FullPath: "root-group/sub-group-1",
			WebURL:   "https://gitlab.com/root-group/sub-group-1",
		}

		subGroup2 := &gitlab.Group{
			ID:       3,
			Name:     "sub-group-2",
			FullPath: "root-group/sub-group-2",
			WebURL:   "https://gitlab.com/root-group/sub-group-2",
		}

		nestedSubGroup := &gitlab.Group{
			ID:       4,
			Name:     "nested-sub-group",
			FullPath: "root-group/sub-group-1/nested-sub-group",
			WebURL:   "https://gitlab.com/root-group/sub-group-1/nested-sub-group",
		}

		// Root group
		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		// Subgroups of root
		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return([]*gitlab.Group{subGroup1, subGroup2}, &gitlab.Response{}, nil)

		// Subgroups of subGroup1
		mockClient.MockGroups.EXPECT().
			ListSubGroups(2, gomock.Any()).
			Return([]*gitlab.Group{nestedSubGroup}, &gitlab.Response{}, nil)

		// Subgroups of subGroup2 (empty)
		mockClient.MockGroups.EXPECT().
			ListSubGroups(3, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		// Subgroups of nestedSubGroup (empty)
		mockClient.MockGroups.EXPECT().
			ListSubGroups(4, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		groups, err := client.GetGroupsRecursively(1)
		require.NoError(t, err)
		assert.Len(t, groups, 4)

		// Verify all groups are present
		groupIDs := make(map[int]bool)
		for _, g := range groups {
			groupIDs[g.ID] = true
		}

		assert.True(t, groupIDs[1])
		assert.True(t, groupIDs[2])
		assert.True(t, groupIDs[3])
		assert.True(t, groupIDs[4])
	})

	t.Run("handles pagination", func(t *testing.T) {
		client, mockClient := testClient(t)

		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "root-group",
			FullPath: "root-group",
			WebURL:   "https://gitlab.com/root-group",
		}

		// Create many subgroups to test pagination
		var page1Groups []*gitlab.Group
		for i := 2; i <= 51; i++ {
			page1Groups = append(page1Groups, &gitlab.Group{
				ID:       i,
				Name:     fmt.Sprintf("sub-group-%d", i),
				FullPath: fmt.Sprintf("root-group/sub-group-%d", i),
			})
		}

		var page2Groups []*gitlab.Group
		for i := 52; i <= 75; i++ {
			page2Groups = append(page2Groups, &gitlab.Group{
				ID:       i,
				Name:     fmt.Sprintf("sub-group-%d", i),
				FullPath: fmt.Sprintf("root-group/sub-group-%d", i),
			})
		}

		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		// First page
		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return(page1Groups, &gitlab.Response{NextPage: 2}, nil).
			Times(1)

		// Second page
		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return(page2Groups, &gitlab.Response{}, nil).
			Times(1)

		// Each subgroup has no children
		for i := 2; i <= 75; i++ {
			mockClient.MockGroups.EXPECT().
				ListSubGroups(i, gomock.Any()).
				Return([]*gitlab.Group{}, &gitlab.Response{}, nil).
				AnyTimes()
		}

		groups, err := client.GetGroupsRecursively(1)
		require.NoError(t, err)
		assert.Len(t, groups, 75) // 1 root + 74 subgroups
	})

	t.Run("handles invalid group ID", func(t *testing.T) {
		client, _ := testClient(t)

		groups, err := client.GetGroupsRecursively(0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid group ID")
		assert.Nil(t, groups)
	})

	t.Run("handles API error", func(t *testing.T) {
		client, mockClient := testClient(t)

		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(nil, nil, errAPI)

		groups, err := client.GetGroupsRecursively(1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get root group")
		assert.Nil(t, groups)
	})
}

func TestGetProjectsRecursively(t *testing.T) {
	t.Run("fetches projects from single group", func(t *testing.T) {
		client, mockClient := testClient(t)

		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "root-group",
			FullPath: "root-group",
		}

		project1 := &gitlab.Project{
			ID:                1,
			Name:              "project-1",
			PathWithNamespace: "root-group/project-1",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "root-group"},
			WebURL:            "https://gitlab.com/root-group/project-1",
		}

		project2 := &gitlab.Project{
			ID:                2,
			Name:              "project-2",
			PathWithNamespace: "root-group/project-2",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "root-group"},
			WebURL:            "https://gitlab.com/root-group/project-2",
		}

		// Groups setup
		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		// Projects setup
		mockClient.MockGroups.EXPECT().
			ListGroupProjects(1, gomock.Any()).
			Return([]*gitlab.Project{project1, project2}, &gitlab.Response{}, nil)

		projects, err := client.GetProjectsRecursively(1)
		require.NoError(t, err)
		assert.Len(t, projects, 2)
		assert.Equal(t, project1, projects[0])
		assert.Equal(t, project2, projects[1])
	})

	t.Run("fetches projects from multiple groups", func(t *testing.T) {
		client, mockClient := testClient(t)

		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "root-group",
			FullPath: "root-group",
		}

		subGroup := &gitlab.Group{
			ID:       2,
			Name:     "sub-group",
			FullPath: "root-group/sub-group",
		}

		rootProject := &gitlab.Project{
			ID:                1,
			Name:              "root-project",
			PathWithNamespace: "root-group/root-project",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "root-group"},
		}

		subProject := &gitlab.Project{
			ID:                2,
			Name:              "sub-project",
			PathWithNamespace: "root-group/sub-group/sub-project",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "root-group/sub-group"},
		}

		// Groups setup
		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return([]*gitlab.Group{subGroup}, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(2, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		// Projects setup
		mockClient.MockGroups.EXPECT().
			ListGroupProjects(1, gomock.Any()).
			Return([]*gitlab.Project{rootProject}, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListGroupProjects(2, gomock.Any()).
			Return([]*gitlab.Project{subProject}, &gitlab.Response{}, nil)

		projects, err := client.GetProjectsRecursively(1)
		require.NoError(t, err)
		assert.Len(t, projects, 2)

		// Verify both projects are present
		projectIDs := make(map[int]bool)
		for _, p := range projects {
			projectIDs[p.ID] = true
		}

		assert.True(t, projectIDs[1])
		assert.True(t, projectIDs[2])
	})

	t.Run("handles groups without projects", func(t *testing.T) {
		client, mockClient := testClient(t)

		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "root-group",
			FullPath: "root-group",
		}

		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListGroupProjects(1, gomock.Any()).
			Return([]*gitlab.Project{}, &gitlab.Response{}, nil)

		projects, err := client.GetProjectsRecursively(1)
		require.NoError(t, err)
		assert.Empty(t, projects)
	})
}

func TestGetGroupAccessTokens(t *testing.T) {
	t.Run("fetches active tokens for single group", func(t *testing.T) {
		client, mockClient := testClient(t)

		group := &gitlab.Group{
			ID:       1,
			Name:     "test-group",
			FullPath: "test-group",
			WebURL:   "https://gitlab.com/test-group",
		}

		expiresAt1 := gitlab.ISOTime(time.Now().Add(30 * 24 * time.Hour))
		expiresAt2 := gitlab.ISOTime(time.Now().Add(60 * 24 * time.Hour))

		token1 := &gitlab.GroupAccessToken{
			PersonalAccessToken: gitlab.PersonalAccessToken{
				ID:        1,
				Name:      "token-1",
				Active:    true,
				ExpiresAt: &expiresAt1,
			},
			AccessLevel: gitlab.MaintainerPermissions,
		}

		token2 := &gitlab.GroupAccessToken{
			PersonalAccessToken: gitlab.PersonalAccessToken{
				ID:        2,
				Name:      "token-2",
				Active:    true,
				ExpiresAt: &expiresAt2,
			},
			AccessLevel: gitlab.MaintainerPermissions,
		}

		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(group, &gitlab.Response{}, nil)

		activeState := gitlab.AccessTokenState("active")
		mockClient.MockGroupAccessTokens.EXPECT().
			ListGroupAccessTokens(1, &gitlab.ListGroupAccessTokensOptions{
				ListOptions: gitlab.ListOptions{PerPage: 50, Page: 1},
				State:       &activeState,
			}).
			Return([]*gitlab.GroupAccessToken{token1, token2}, &gitlab.Response{}, nil)

		tokens, err := client.GetGroupAccessTokens(1, false)
		require.NoError(t, err)
		assert.Len(t, tokens, 2)

		// Verify token wrapping
		assert.Equal(t, token1, tokens[0].GroupAccessToken)
		assert.Equal(t, "test-group", tokens[0].GroupName)
		assert.Equal(t, "test-group", tokens[0].GroupPath)
		assert.Equal(t, "https://gitlab.com/test-group", tokens[0].GroupWebURL)
	})

	t.Run("fetches all tokens including inactive", func(t *testing.T) {
		client, mockClient := testClient(t)

		group := &gitlab.Group{
			ID:       1,
			Name:     "test-group",
			FullPath: "test-group",
			WebURL:   "https://gitlab.com/test-group",
		}

		activeToken := &gitlab.GroupAccessToken{
			PersonalAccessToken: gitlab.PersonalAccessToken{
				ID:     1,
				Name:   "active-token",
				Active: true,
			},
			AccessLevel: gitlab.MaintainerPermissions,
		}

		inactiveToken := &gitlab.GroupAccessToken{
			PersonalAccessToken: gitlab.PersonalAccessToken{
				ID:     2,
				Name:   "inactive-token",
				Active: false,
			},
			AccessLevel: gitlab.MaintainerPermissions,
		}

		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(group, &gitlab.Response{}, nil)

		mockClient.MockGroupAccessTokens.EXPECT().
			ListGroupAccessTokens(1, &gitlab.ListGroupAccessTokensOptions{
				ListOptions: gitlab.ListOptions{PerPage: 50, Page: 1},
			}).
			Return([]*gitlab.GroupAccessToken{activeToken, inactiveToken}, &gitlab.Response{}, nil)

		tokens, err := client.GetGroupAccessTokens(1, true)
		require.NoError(t, err)
		assert.Len(t, tokens, 2)
	})

	t.Run("handles invalid group ID", func(t *testing.T) {
		client, _ := testClient(t)

		tokens, err := client.GetGroupAccessTokens(0, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid group ID")
		assert.Nil(t, tokens)
	})
}

func TestGetGroupAccessTokensRecursively(t *testing.T) {
	t.Run("fetches tokens from multiple groups", func(t *testing.T) {
		client, mockClient := testClient(t)

		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "root-group",
			FullPath: "root-group",
			WebURL:   "https://gitlab.com/root-group",
		}

		subGroup := &gitlab.Group{
			ID:       2,
			Name:     "sub-group",
			FullPath: "root-group/sub-group",
			WebURL:   "https://gitlab.com/root-group/sub-group",
		}

		rootToken := &gitlab.GroupAccessToken{
			PersonalAccessToken: gitlab.PersonalAccessToken{
				ID:     1,
				Name:   "root-token",
				Active: true,
			},
			AccessLevel: gitlab.MaintainerPermissions,
		}

		subToken := &gitlab.GroupAccessToken{
			PersonalAccessToken: gitlab.PersonalAccessToken{
				ID:     2,
				Name:   "sub-token",
				Active: true,
			},
			AccessLevel: gitlab.MaintainerPermissions,
		}

		// Groups setup
		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return([]*gitlab.Group{subGroup}, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(2, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		// Tokens setup
		activeState := gitlab.AccessTokenState("active")

		mockClient.MockGroupAccessTokens.EXPECT().
			ListGroupAccessTokens(1, &gitlab.ListGroupAccessTokensOptions{
				ListOptions: gitlab.ListOptions{PerPage: 50, Page: 1},
				State:       &activeState,
			}).
			Return([]*gitlab.GroupAccessToken{rootToken}, &gitlab.Response{}, nil)

		mockClient.MockGroupAccessTokens.EXPECT().
			ListGroupAccessTokens(2, &gitlab.ListGroupAccessTokensOptions{
				ListOptions: gitlab.ListOptions{PerPage: 50, Page: 1},
				State:       &activeState,
			}).
			Return([]*gitlab.GroupAccessToken{subToken}, &gitlab.Response{}, nil)

		tokens, err := client.GetGroupAccessTokensRecursively(1, false)
		require.NoError(t, err)
		assert.Len(t, tokens, 2)

		// Verify tokens are from correct groups
		tokensByGroup := make(map[string]bool)
		for _, t := range tokens {
			tokensByGroup[t.GroupPath] = true
		}

		assert.True(t, tokensByGroup["root-group"])
		assert.True(t, tokensByGroup["root-group/sub-group"])
	})
}

func TestGetProjectAccessTokens(t *testing.T) {
	t.Run("fetches active tokens for single project", func(t *testing.T) {
		client, mockClient := testClient(t)

		project := &gitlab.Project{
			ID:                1,
			Name:              "test-project",
			PathWithNamespace: "group/test-project",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "group"},
			WebURL:            "https://gitlab.com/group/test-project",
		}

		token := &gitlab.ProjectAccessToken{
			PersonalAccessToken: gitlab.PersonalAccessToken{
				ID:     1,
				Name:   "project-token",
				Active: true,
			},
			AccessLevel: gitlab.MaintainerPermissions,
		}

		mockClient.MockProjects.EXPECT().
			GetProject(1, nil).
			Return(project, &gitlab.Response{}, nil)

		activeState := "active"
		mockClient.MockProjectAccessTokens.EXPECT().
			ListProjectAccessTokens(1, &gitlab.ListProjectAccessTokensOptions{
				ListOptions: gitlab.ListOptions{PerPage: 50, Page: 1},
				State:       &activeState,
			}).
			Return([]*gitlab.ProjectAccessToken{token}, &gitlab.Response{}, nil)

		tokens, err := client.GetProjectAccessTokens(1, false)
		require.NoError(t, err)
		assert.Len(t, tokens, 1)

		// Verify token wrapping
		assert.Equal(t, token, tokens[0].ProjectAccessToken)
		assert.Equal(t, "test-project", tokens[0].ProjectName)
		assert.Equal(t, "group/test-project", tokens[0].ProjectPath)
		assert.Equal(t, "group", tokens[0].ProjectNamespace)
		assert.Equal(t, "https://gitlab.com/group/test-project", tokens[0].ProjectWebURL)
	})

	t.Run("filters out inactive tokens when includeInactive is false", func(t *testing.T) {
		client, mockClient := testClient(t)

		project := &gitlab.Project{
			ID:                1,
			Name:              "test-project",
			PathWithNamespace: "group/test-project",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "group"},
			WebURL:            "https://gitlab.com/group/test-project",
		}

		activeToken := &gitlab.ProjectAccessToken{
			PersonalAccessToken: gitlab.PersonalAccessToken{
				ID:     1,
				Name:   "active-token",
				Active: true,
			},
			AccessLevel: gitlab.MaintainerPermissions,
		}

		// This inactive token should be filtered out
		inactiveToken := &gitlab.ProjectAccessToken{
			PersonalAccessToken: gitlab.PersonalAccessToken{
				ID:     2,
				Name:   "inactive-token",
				Active: false,
			},
			AccessLevel: gitlab.MaintainerPermissions,
		}

		mockClient.MockProjects.EXPECT().
			GetProject(1, nil).
			Return(project, &gitlab.Response{}, nil)

		activeState := "active"
		mockClient.MockProjectAccessTokens.EXPECT().
			ListProjectAccessTokens(1, &gitlab.ListProjectAccessTokensOptions{
				ListOptions: gitlab.ListOptions{PerPage: 50, Page: 1},
				State:       &activeState,
			}).
			Return([]*gitlab.ProjectAccessToken{activeToken, inactiveToken}, &gitlab.Response{}, nil)

		tokens, err := client.GetProjectAccessTokens(1, false)
		require.NoError(t, err)
		assert.Len(t, tokens, 1)
		assert.Equal(t, "active-token", tokens[0].Name)
	})
}

func TestGetProjectAccessTokensRecursively(t *testing.T) {
	t.Run("fetches tokens from all projects in group hierarchy", func(t *testing.T) {
		client, mockClient := testClient(t)

		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "root-group",
			FullPath: "root-group",
		}

		project1 := &gitlab.Project{
			ID:                1,
			Name:              "project-1",
			PathWithNamespace: "root-group/project-1",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "root-group"},
			WebURL:            "https://gitlab.com/root-group/project-1",
		}

		project2 := &gitlab.Project{
			ID:                2,
			Name:              "project-2",
			PathWithNamespace: "root-group/project-2",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "root-group"},
			WebURL:            "https://gitlab.com/root-group/project-2",
		}

		token1 := &gitlab.ProjectAccessToken{
			PersonalAccessToken: gitlab.PersonalAccessToken{
				ID:     1,
				Name:   "token-1",
				Active: true,
			},
			AccessLevel: gitlab.MaintainerPermissions,
		}

		token2 := &gitlab.ProjectAccessToken{
			PersonalAccessToken: gitlab.PersonalAccessToken{
				ID:     2,
				Name:   "token-2",
				Active: true,
			},
			AccessLevel: gitlab.MaintainerPermissions,
		}

		// Groups setup
		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		// Projects setup
		mockClient.MockGroups.EXPECT().
			ListGroupProjects(1, gomock.Any()).
			Return([]*gitlab.Project{project1, project2}, &gitlab.Response{}, nil)

		// Project access tokens
		activeState := "active"

		mockClient.MockProjectAccessTokens.EXPECT().
			ListProjectAccessTokens(1, &gitlab.ListProjectAccessTokensOptions{
				ListOptions: gitlab.ListOptions{PerPage: 50, Page: 1},
				State:       &activeState,
			}).
			Return([]*gitlab.ProjectAccessToken{token1}, &gitlab.Response{}, nil)

		mockClient.MockProjectAccessTokens.EXPECT().
			ListProjectAccessTokens(2, &gitlab.ListProjectAccessTokensOptions{
				ListOptions: gitlab.ListOptions{PerPage: 50, Page: 1},
				State:       &activeState,
			}).
			Return([]*gitlab.ProjectAccessToken{token2}, &gitlab.Response{}, nil)

		tokens, err := client.GetProjectAccessTokensRecursively(1, false)
		require.NoError(t, err)
		assert.Len(t, tokens, 2)

		// Verify tokens are from correct projects
		tokensByProject := make(map[string]bool)
		for _, t := range tokens {
			tokensByProject[t.ProjectPath] = true
		}

		assert.True(t, tokensByProject["root-group/project-1"])
		assert.True(t, tokensByProject["root-group/project-2"])
	})

	t.Run("handles invalid group ID", func(t *testing.T) {
		client, _ := testClient(t)

		tokens, err := client.GetProjectAccessTokensRecursively(0, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid group ID")
		assert.Nil(t, tokens)
	})
}

func TestGetPipelineTriggers(t *testing.T) {
	t.Run("fetches triggers for single project", func(t *testing.T) {
		client, mockClient := testClient(t)

		project := &gitlab.Project{
			ID:                1,
			Name:              "test-project",
			PathWithNamespace: "group/test-project",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "group"},
			WebURL:            "https://gitlab.com/group/test-project",
		}

		trigger1 := &gitlab.PipelineTrigger{
			ID:          1,
			Description: "Trigger 1",
			Token:       "token1",
		}

		trigger2 := &gitlab.PipelineTrigger{
			ID:          2,
			Description: "Trigger 2",
			Token:       "token2",
		}

		mockClient.MockProjects.EXPECT().
			GetProject(1, nil).
			Return(project, &gitlab.Response{}, nil)

		mockClient.MockPipelineTriggers.EXPECT().
			ListPipelineTriggers(1, &gitlab.ListPipelineTriggersOptions{
				PerPage: 50,
				Page:    1,
			}).
			Return([]*gitlab.PipelineTrigger{trigger1, trigger2}, &gitlab.Response{}, nil)

		triggers, err := client.GetPipelineTriggers(1)
		require.NoError(t, err)
		assert.Len(t, triggers, 2)

		// Verify trigger wrapping
		assert.Equal(t, trigger1, triggers[0].PipelineTrigger)
		assert.Equal(t, "test-project", triggers[0].ProjectName)
		assert.Equal(t, "group/test-project", triggers[0].ProjectPath)
		assert.Equal(t, "group", triggers[0].ProjectNamespace)
		assert.Equal(t, "https://gitlab.com/group/test-project", triggers[0].ProjectWebURL)
	})

	t.Run("handles project without triggers", func(t *testing.T) {
		client, mockClient := testClient(t)

		project := &gitlab.Project{
			ID:                1,
			Name:              "test-project",
			PathWithNamespace: "group/test-project",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "group"},
			WebURL:            "https://gitlab.com/group/test-project",
		}

		mockClient.MockProjects.EXPECT().
			GetProject(1, nil).
			Return(project, &gitlab.Response{}, nil)

		mockClient.MockPipelineTriggers.EXPECT().
			ListPipelineTriggers(1, gomock.Any()).
			Return([]*gitlab.PipelineTrigger{}, &gitlab.Response{}, nil)

		triggers, err := client.GetPipelineTriggers(1)
		require.NoError(t, err)
		assert.Empty(t, triggers)
	})
}

func TestGetPipelineTriggersRecursively(t *testing.T) {
	t.Run("fetches triggers from all projects in group hierarchy", func(t *testing.T) {
		client, mockClient := testClient(t)

		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "root-group",
			FullPath: "root-group",
		}

		project1 := &gitlab.Project{
			ID:                1,
			Name:              "project-1",
			PathWithNamespace: "root-group/project-1",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "root-group"},
			WebURL:            "https://gitlab.com/root-group/project-1",
		}

		project2 := &gitlab.Project{
			ID:                2,
			Name:              "project-2",
			PathWithNamespace: "root-group/project-2",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "root-group"},
			WebURL:            "https://gitlab.com/root-group/project-2",
		}

		trigger1 := &gitlab.PipelineTrigger{
			ID:          1,
			Description: "Project 1 Trigger",
		}

		trigger2 := &gitlab.PipelineTrigger{
			ID:          2,
			Description: "Project 2 Trigger",
		}

		// Groups setup
		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		// Projects setup
		mockClient.MockGroups.EXPECT().
			ListGroupProjects(1, gomock.Any()).
			Return([]*gitlab.Project{project1, project2}, &gitlab.Response{}, nil)

		// Pipeline triggers
		mockClient.MockPipelineTriggers.EXPECT().
			ListPipelineTriggers(1, gomock.Any()).
			Return([]*gitlab.PipelineTrigger{trigger1}, &gitlab.Response{}, nil)

		mockClient.MockPipelineTriggers.EXPECT().
			ListPipelineTriggers(2, gomock.Any()).
			Return([]*gitlab.PipelineTrigger{trigger2}, &gitlab.Response{}, nil)

		triggers, err := client.GetPipelineTriggersRecursively(1)
		require.NoError(t, err)
		assert.Len(t, triggers, 2)

		// Verify triggers are from correct projects
		triggersByProject := make(map[string]bool)
		for _, t := range triggers {
			triggersByProject[t.ProjectPath] = true
		}

		assert.True(t, triggersByProject["root-group/project-1"])
		assert.True(t, triggersByProject["root-group/project-2"])
	})

	t.Run("handles projects without triggers", func(t *testing.T) {
		client, mockClient := testClient(t)

		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "root-group",
			FullPath: "root-group",
		}

		project := &gitlab.Project{
			ID:                1,
			Name:              "project-without-triggers",
			PathWithNamespace: "root-group/project-without-triggers",
			Namespace:         &gitlab.ProjectNamespace{FullPath: "root-group"},
		}

		// Groups setup
		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		// Projects setup
		mockClient.MockGroups.EXPECT().
			ListGroupProjects(1, gomock.Any()).
			Return([]*gitlab.Project{project}, &gitlab.Response{}, nil)

		// No triggers
		mockClient.MockPipelineTriggers.EXPECT().
			ListPipelineTriggers(1, gomock.Any()).
			Return([]*gitlab.PipelineTrigger{}, &gitlab.Response{}, nil)

		triggers, err := client.GetPipelineTriggersRecursively(1)
		require.NoError(t, err)
		assert.Empty(t, triggers)
	})
}

func TestGetProjectVariables(t *testing.T) {
	t.Run("fetches variables for a single project", func(t *testing.T) {
		client, mockClient := testClient(t)

		project := &gitlab.Project{
			ID:                10,
			Name:              "test-project",
			PathWithNamespace: "group/test-project",
			WebURL:            "https://gitlab.com/group/test-project",
			Namespace: &gitlab.ProjectNamespace{
				FullPath: "group",
			},
		}

		variable1 := &gitlab.ProjectVariable{
			Key:              "VAR1",
			Value:            "value1",
			VariableType:     gitlab.VariableTypeValue("env_var"),
			Protected:        false,
			Masked:           false,
			EnvironmentScope: "*",
		}

		variable2 := &gitlab.ProjectVariable{
			Key:              "VAR2",
			Value:            "value2",
			VariableType:     gitlab.VariableTypeValue("file"),
			Protected:        true,
			Masked:           true,
			EnvironmentScope: "production",
		}

		// Mock expectations
		mockClient.MockProjects.EXPECT().
			GetProject(10, nil).
			Return(project, &gitlab.Response{}, nil)

		mockClient.MockProjectVariables.EXPECT().
			ListVariables(10, gomock.Any()).
			Return([]*gitlab.ProjectVariable{variable1, variable2}, &gitlab.Response{}, nil)

		// Execute
		variables, err := client.GetProjectVariables(10)
		require.NoError(t, err)
		require.Len(t, variables, 2)

		// Verify the results include project information
		for _, v := range variables {
			assert.Equal(t, "test-project", v.ProjectName)
			assert.Equal(t, "group/test-project", v.ProjectPath)
			assert.Equal(t, "group", v.ProjectNamespace)
			assert.Equal(t, "https://gitlab.com/group/test-project", v.ProjectWebURL)

			switch v.Key {
			case "VAR1":
				assert.Equal(t, "value1", v.Value)
				assert.Equal(t, gitlab.VariableTypeValue("env_var"), v.VariableType)
				assert.False(t, v.Protected)
				assert.False(t, v.Masked)
			case "VAR2":
				assert.Equal(t, "value2", v.Value)
				assert.Equal(t, gitlab.VariableTypeValue("file"), v.VariableType)
				assert.True(t, v.Protected)
				assert.True(t, v.Masked)
			}
		}
	})

	t.Run("handles API errors", func(t *testing.T) {
		client, mockClient := testClient(t)

		mockClient.MockProjects.EXPECT().
			GetProject(10, nil).
			Return(nil, nil, errAPI)

		variables, err := client.GetProjectVariables(10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get project")
		assert.Nil(t, variables)
	})

	t.Run("handles empty variables list", func(t *testing.T) {
		client, mockClient := testClient(t)

		project := &gitlab.Project{
			ID:                10,
			Name:              "test-project",
			PathWithNamespace: "group/test-project",
			WebURL:            "https://gitlab.com/group/test-project",
			Namespace: &gitlab.ProjectNamespace{
				FullPath: "group",
			},
		}

		mockClient.MockProjects.EXPECT().
			GetProject(10, nil).
			Return(project, &gitlab.Response{}, nil)

		mockClient.MockProjectVariables.EXPECT().
			ListVariables(10, gomock.Any()).
			Return([]*gitlab.ProjectVariable{}, &gitlab.Response{}, nil)

		variables, err := client.GetProjectVariables(10)
		require.NoError(t, err)
		assert.Empty(t, variables)
	})
}

func TestGetProjectVariablesRecursively(t *testing.T) {
	t.Run("fetches variables from multiple projects", func(t *testing.T) {
		client, mockClient := testClient(t)

		// Create test data
		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "root-group",
			FullPath: "root-group",
			WebURL:   "https://gitlab.com/root-group",
		}

		project1 := &gitlab.Project{
			ID:                10,
			Name:              "project-1",
			PathWithNamespace: "root-group/project-1",
			WebURL:            "https://gitlab.com/root-group/project-1",
			Namespace: &gitlab.ProjectNamespace{
				FullPath: "root-group",
			},
		}

		project2 := &gitlab.Project{
			ID:                11,
			Name:              "project-2",
			PathWithNamespace: "root-group/project-2",
			WebURL:            "https://gitlab.com/root-group/project-2",
			Namespace: &gitlab.ProjectNamespace{
				FullPath: "root-group",
			},
		}

		var1 := &gitlab.ProjectVariable{
			Key:              "VAR1",
			Value:            "value1",
			VariableType:     gitlab.VariableTypeValue("env_var"),
			Protected:        false,
			Masked:           false,
			EnvironmentScope: "*",
		}

		var2 := &gitlab.ProjectVariable{
			Key:              "VAR2",
			Value:            "value2",
			VariableType:     gitlab.VariableTypeValue("file"),
			Protected:        true,
			Masked:           true,
			EnvironmentScope: "production",
		}

		// Mock expectations
		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListGroupProjects(1, gomock.Any()).
			Return([]*gitlab.Project{project1, project2}, &gitlab.Response{}, nil)

		// Project 1 variables
		mockClient.MockProjectVariables.EXPECT().
			ListVariables(10, gomock.Any()).
			Return([]*gitlab.ProjectVariable{var1}, &gitlab.Response{}, nil)

		// Project 2 variables
		mockClient.MockProjectVariables.EXPECT().
			ListVariables(11, gomock.Any()).
			Return([]*gitlab.ProjectVariable{var2}, &gitlab.Response{}, nil)

		// Execute
		variables, err := client.GetProjectVariablesRecursively(1)
		require.NoError(t, err)
		require.Len(t, variables, 2)

		// Verify the results include project information
		var foundVar1, foundVar2 bool

		for _, v := range variables {
			switch v.Key {
			case "VAR1":
				foundVar1 = true

				assert.Equal(t, "project-1", v.ProjectName)
				assert.Equal(t, "root-group/project-1", v.ProjectPath)
			case "VAR2":
				foundVar2 = true

				assert.Equal(t, "project-2", v.ProjectName)
				assert.Equal(t, "root-group/project-2", v.ProjectPath)
			}
		}

		assert.True(t, foundVar1, "Variable 1 not found")
		assert.True(t, foundVar2, "Variable 2 not found")
	})

	t.Run("handles errors gracefully", func(t *testing.T) {
		client, mockClient := testClient(t)

		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(nil, nil, errAPI)

		variables, err := client.GetProjectVariablesRecursively(1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get root group")
		assert.Nil(t, variables)
	})

	t.Run("continues processing when individual project fails", func(t *testing.T) {
		client, mockClient := testClientWithDebug(t)

		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "root-group",
			FullPath: "root-group",
			WebURL:   "https://gitlab.com/root-group",
		}

		project1 := &gitlab.Project{
			ID:                10,
			Name:              "project-1",
			PathWithNamespace: "root-group/project-1",
			WebURL:            "https://gitlab.com/root-group/project-1",
			Namespace: &gitlab.ProjectNamespace{
				FullPath: "root-group",
			},
		}

		project2 := &gitlab.Project{
			ID:                11,
			Name:              "project-2",
			PathWithNamespace: "root-group/project-2",
			WebURL:            "https://gitlab.com/root-group/project-2",
			Namespace: &gitlab.ProjectNamespace{
				FullPath: "root-group",
			},
		}

		var2 := &gitlab.ProjectVariable{
			Key:              "VAR2",
			Value:            "value2",
			VariableType:     gitlab.VariableTypeValue("file"),
			Protected:        true,
			Masked:           true,
			EnvironmentScope: "production",
		}

		// Mock expectations
		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListGroupProjects(1, gomock.Any()).
			Return([]*gitlab.Project{project1, project2}, &gitlab.Response{}, nil)

		// Project 1 fails to get variables
		mockClient.MockProjectVariables.EXPECT().
			ListVariables(10, gomock.Any()).
			Return(nil, nil, errAPI)

		// Project 2 succeeds
		mockClient.MockProjectVariables.EXPECT().
			ListVariables(11, gomock.Any()).
			Return([]*gitlab.ProjectVariable{var2}, &gitlab.Response{}, nil)

		// Execute
		variables, err := client.GetProjectVariablesRecursively(1)
		require.NoError(t, err)
		require.Len(t, variables, 1)
		assert.Equal(t, "VAR2", variables[0].Key)
	})
}

func TestDebugMode(t *testing.T) {
	t.Run("prints debug messages when enabled", func(t *testing.T) {
		// This test just ensures debug mode doesn't break functionality
		// In a real scenario, we'd capture stdout to verify debug messages
		client, mockClient := testClientWithDebug(t)

		rootGroup := &gitlab.Group{
			ID:       1,
			Name:     "debug-test-group",
			FullPath: "debug-test-group",
		}

		mockClient.MockGroups.EXPECT().
			GetGroup(1, nil).
			Return(rootGroup, &gitlab.Response{}, nil)

		mockClient.MockGroups.EXPECT().
			ListSubGroups(1, gomock.Any()).
			Return([]*gitlab.Group{}, &gitlab.Response{}, nil)

		groups, err := client.GetGroupsRecursively(1)
		require.NoError(t, err)
		assert.Len(t, groups, 1)
	})
}
