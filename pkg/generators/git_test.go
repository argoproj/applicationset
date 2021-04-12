package generators

import (
	"context"
	"fmt"
	"testing"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// type clientSet struct {
// 	RepoServerServiceClient apiclient.RepoServerServiceClient
// }

// func (c *clientSet) NewRepoServerClient() (io.Closer, apiclient.RepoServerServiceClient, error) {
// 	return io.NewCloser(func() error { return nil }), c.RepoServerServiceClient, nil
// }

type argoCDServiceMock struct {
	mock *mock.Mock
}

func (a argoCDServiceMock) GetApps(ctx context.Context, repoURL string, revision string) ([]string, error) {
	args := a.mock.Called(ctx, repoURL, revision)

	return args.Get(0).([]string), args.Error(1)
}

func (a argoCDServiceMock) GetFilePaths(ctx context.Context, repoURL string, revision string, pattern string) ([]string, error) {
	args := a.mock.Called(ctx, repoURL, revision, pattern)

	return args.Get(0).([]string), args.Error(1)
}

func (a argoCDServiceMock) GetFileContent(ctx context.Context, repoURL string, revision string, path string) ([]byte, error) {
	args := a.mock.Called(ctx, repoURL, revision, path)

	return args.Get(0).([]byte), args.Error(1)
}

func (a argoCDServiceMock) GetDirectories(ctx context.Context, repoURL string, revision string) ([]string, error) {
	args := a.mock.Called(ctx, repoURL, revision)
	return args.Get(0).([]string), args.Error(1)
}

func TestGitGenerateParamsFromDirectories(t *testing.T) {

	cases := []struct {
		name          string
		directories   []argoprojiov1alpha1.GitDirectoryGeneratorItem
		repoApps      []string
		repoError     error
		expected      []map[string]string
		expectedError error
	}{
		{
			name:        "happy flow - created apps",
			directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{Path: "*"}},
			repoApps: []string{
				"app1",
				"app2",
				"p1/app3",
			},
			repoError: nil,
			expected: []map[string]string{
				{"path": "app1", "path.basename": "app1"},
				{"path": "app2", "path.basename": "app2"},
			},
			expectedError: nil,
		},
		{
			name:        "It filters application according to the paths",
			directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{Path: "p1/*"}, {Path: "p1/*/*"}},
			repoApps: []string{
				"app1",
				"p1/app2",
				"p1/p2/app3",
				"p1/p2/p3/app4",
			},
			repoError: nil,
			expected: []map[string]string{
				{"path": "p1/app2", "path.basename": "app2"},
				{"path": "p1/p2/app3", "path.basename": "app3"},
			},
			expectedError: nil,
		},
		{
			name:        "It filters application according to the paths with Exclude",
			directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{Path: "p1/*", Exclude: true}, {Path: "*"}, {Path: "*/*"}},
			repoApps: []string{
				"app1",
				"app2",
				"p1/app2",
				"p1/app3",
				"p2/app3",
			},
			repoError: nil,
			expected: []map[string]string{
				{"path": "app1", "path.basename": "app1"},
				{"path": "app2", "path.basename": "app2"},
				{"path": "p2/app3", "path.basename": "app3"},
			},
			expectedError: nil,
		},
		{
			name:          "handles empty response from repo server",
			directories:   []argoprojiov1alpha1.GitDirectoryGeneratorItem{{Path: "*"}},
			repoApps:      []string{},
			repoError:     nil,
			expected:      []map[string]string{},
			expectedError: nil,
		},
		{
			name:          "handles error from repo server",
			directories:   []argoprojiov1alpha1.GitDirectoryGeneratorItem{{Path: "*"}},
			repoApps:      []string{},
			repoError:     fmt.Errorf("error"),
			expected:      []map[string]string{},
			expectedError: fmt.Errorf("error"),
		},
	}

	for _, c := range cases {
		cc := c
		t.Run(cc.name, func(t *testing.T) {
			argoCDServiceMock := argoCDServiceMock{mock: &mock.Mock{}}

			argoCDServiceMock.mock.On("GetDirectories", mock.Anything, mock.Anything, mock.Anything).Return(c.repoApps, c.repoError)

			var gitGenerator = NewGitGenerator(argoCDServiceMock)
			applicationSetInfo := argoprojiov1alpha1.ApplicationSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "set",
				},
				Spec: argoprojiov1alpha1.ApplicationSetSpec{
					Generators: []argoprojiov1alpha1.ApplicationSetGenerator{{
						Git: &argoprojiov1alpha1.GitGenerator{
							RepoURL:     "RepoURL",
							Revision:    "Revision",
							Directories: c.directories,
						},
					}},
				},
			}

			got, err := gitGenerator.GenerateParams(&applicationSetInfo.Spec.Generators[0])

			if c.expectedError != nil {
				assert.EqualError(t, err, c.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expected, got)
			}

			argoCDServiceMock.mock.AssertExpectations(t)
		})
	}

}

func TestGitGenerateParamsFromFiles(t *testing.T) {

	cases := []struct {
		name string
		// files is the list of paths/globs to match
		files []argoprojiov1alpha1.GitFileGeneratorItem
		// repoPaths is the list of matching paths in the simulated git repository
		repoPaths []string
		// repoFileContents maps repo path to the literal contents of that path
		repoFileContents map[string][]byte
		// if repoPathsError is non-nil, the call to GetPaths(...) will return this error value
		repoPathsError error
		// if repoFileContentsErrors contains a path key, the error value will be returned on the call to GetFileContents(...)
		repoFileContentsErrors map[string]error
		expected               []map[string]string
		expectedError          error
	}{
		{
			name:  "happy flow: create params from git files",
			files: []argoprojiov1alpha1.GitFileGeneratorItem{{Path: "**/config.json"}},
			repoPaths: []string{
				"cluster-config/production/config.json",
				"cluster-config/staging/config.json",
			},
			repoFileContents: map[string][]byte{
				"cluster-config/production/config.json": []byte(`{
   "cluster": {
       "owner": "john.doe@example.com",
       "name": "production",
       "address": "https://kubernetes.default.svc"
   },
   "key1": "val1",
   "key2": {
       "key2_1": "val2_1",
       "key2_2": {
           "key2_2_1": "val2_2_1"
       }
   }
}`),
				"cluster-config/staging/config.json": []byte(`{
   "cluster": {
       "owner": "foo.bar@example.com",
       "name": "staging",
       "address": "https://kubernetes.default.svc"
   }
}`),
			},
			repoPathsError:         nil,
			repoFileContentsErrors: nil,
			expected: []map[string]string{
				{
					"cluster.owner":        "john.doe@example.com",
					"cluster.name":         "production",
					"cluster.address":      "https://kubernetes.default.svc",
					"key1":                 "val1",
					"key2.key2_1":          "val2_1",
					"key2.key2_2.key2_2_1": "val2_2_1",
				},
				{
					"cluster.owner":   "foo.bar@example.com",
					"cluster.name":    "staging",
					"cluster.address": "https://kubernetes.default.svc",
				},
			},
			expectedError: nil,
		},
		{
			name:                   "handles error during getting repo paths",
			files:                  []argoprojiov1alpha1.GitFileGeneratorItem{{Path: "**/config.json"}},
			repoPaths:              []string{},
			repoFileContents:       map[string][]byte{},
			repoPathsError:         fmt.Errorf("paths error"),
			repoFileContentsErrors: nil,
			expected:               []map[string]string{},
			expectedError:          fmt.Errorf("paths error"),
		},
		{
			name:  "handles error during getting repo file contents",
			files: []argoprojiov1alpha1.GitFileGeneratorItem{{Path: "**/config.json"}},
			repoPaths: []string{
				"cluster-config/production/config.json",
				"cluster-config/staging/config.json",
			},
			repoFileContents: map[string][]byte{
				"cluster-config/production/config.json": []byte(`{
   "cluster": {
       "owner": "john.doe@example.com",
       "name": "production",
       "address": "https://kubernetes.default.svc"
   }
}`),
				"cluster-config/staging/config.json": nil,
			},
			repoPathsError: nil,
			repoFileContentsErrors: map[string]error{
				"cluster-config/production/config.json": nil,
				"cluster-config/staging/config.json":    fmt.Errorf("staging config file get content error"),
			},
			expected:      []map[string]string{},
			expectedError: fmt.Errorf("unable to process file 'cluster-config/staging/config.json': staging config file get content error"),
		},
		{
			name:  "test invalid JSON file returns error",
			files: []argoprojiov1alpha1.GitFileGeneratorItem{{Path: "**/config.json"}},
			repoPaths: []string{
				"cluster-config/production/config.json",
			},
			repoFileContents: map[string][]byte{
				"cluster-config/production/config.json": []byte(`invalid json file`),
			},
			repoPathsError:         nil,
			repoFileContentsErrors: map[string]error{},
			expected:               []map[string]string{},
			expectedError:          fmt.Errorf("unable to process file 'cluster-config/production/config.json': unable to parse JSON file: invalid character 'i' looking for beginning of value"),
		},
		{
			name:  "test JSON array",
			files: []argoprojiov1alpha1.GitFileGeneratorItem{{Path: "**/config.json"}},
			repoPaths: []string{
				"cluster-config/production/config.json",
			},
			repoFileContents: map[string][]byte{
				"cluster-config/production/config.json": []byte(`
				[
					{
						"cluster": {
							"owner": "john.doe@example.com",
							"name": "production",
							"address": "https://kubernetes.default.svc",
							"inner": {
								"one" : "two"
							}
						}
					},
					{
						"cluster": {
							"owner": "john.doe@example.com",
							"name": "staging",
							"address": "https://kubernetes.default.svc"
						}
					}
				]`),
			},
			repoPathsError:         nil,
			repoFileContentsErrors: map[string]error{},
			expected: []map[string]string{
				{
					"cluster.owner":     "john.doe@example.com",
					"cluster.name":      "production",
					"cluster.address":   "https://kubernetes.default.svc",
					"cluster.inner.one": "two",
				},
				{
					"cluster.owner":   "john.doe@example.com",
					"cluster.name":    "staging",
					"cluster.address": "https://kubernetes.default.svc",
				},
			},
			expectedError: nil,
		},
	}

	for _, c := range cases {
		cc := c
		t.Run(cc.name, func(t *testing.T) {
			argoCDServiceMock := argoCDServiceMock{mock: &mock.Mock{}}
			argoCDServiceMock.mock.On("GetFilePaths", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(c.repoPaths, c.repoPathsError)
			if c.repoPaths != nil {
				for _, repoPath := range c.repoPaths {
					argoCDServiceMock.mock.On("GetFileContent", mock.Anything, mock.Anything, mock.Anything, repoPath).
						Return(c.repoFileContents[repoPath], c.repoFileContentsErrors[repoPath]).Once()
				}
			}

			var gitGenerator = NewGitGenerator(argoCDServiceMock)
			applicationSetInfo := argoprojiov1alpha1.ApplicationSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "set",
				},
				Spec: argoprojiov1alpha1.ApplicationSetSpec{
					Generators: []argoprojiov1alpha1.ApplicationSetGenerator{{
						Git: &argoprojiov1alpha1.GitGenerator{
							RepoURL:  "RepoURL",
							Revision: "Revision",
							Files:    c.files,
						},
					}},
				},
			}

			got, err := gitGenerator.GenerateParams(&applicationSetInfo.Spec.Generators[0])
			fmt.Println(got, err)

			if c.expectedError != nil {
				assert.EqualError(t, err, c.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, c.expected, got)
			}

			argoCDServiceMock.mock.AssertExpectations(t)
		})
	}

}
