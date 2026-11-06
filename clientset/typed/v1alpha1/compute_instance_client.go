package v1alpha1

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

// computeInstanceClient implements ComputeInstanceInterface (read-only) for project scope
// (workspace == "") or workspace scope.
type computeInstanceClient struct {
	devScopedClient
}

func newComputeInstanceClient(restClient *rest.Client, project, workspace string) *computeInstanceClient {
	return &computeInstanceClient{devScopedClient{rest: restClient, project: project, workspace: workspace}}
}

func (c *computeInstanceClient) paths() (collection func() string, item func(string) string) {
	return convert.ComputeInstancePaths(c.scope())
}

func (c *computeInstanceClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.ComputeInstance, error) {
	_, item := c.paths()
	var out convert.PaaSComputeInstance
	if err := c.rest.Get(ctx, withFirstClass(item(name)), &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSComputeInstance(&out, c.project, c.workspace), nil
}

func (c *computeInstanceClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.ComputeInstanceList, error) {
	collection, _ := c.paths()
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	path = withFirstClass(path)
	var out convert.PaaSComputeInstanceList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSComputeInstanceList(&out, c.project, c.workspace), nil
}

func (c *computeInstanceClient) FailureLog(ctx context.Context, name string, maxLines int) (*apiv1.ComputeFailureLog, error) {
	if maxLines <= 0 {
		maxLines = 15
	}
	inst, err := c.Get(ctx, name, gpupaas.GetOptions{})
	if err != nil {
		return nil, err
	}
	out := &apiv1.ComputeFailureLog{}
	st := inst.Status
	if st.Environment == nil || st.Environment.Name == "" || st.TriggerID() == "" {
		return out, nil
	}
	out.Environment = st.Environment.Name
	project := firstNonEmpty(inst.Metadata.Project, c.project)

	var event struct {
		WorkflowDetails struct {
			Activities []struct {
				ID             string `json:"id"`
				Name           string `json:"name"`
				CompletedState string `json:"completedState"`
			} `json:"activities"`
		} `json:"workflowDetails"`
	}
	eventPath := fmt.Sprintf(convert.EaaSEnvironmentEventPath, url.PathEscape(project), url.PathEscape(st.Environment.Name), url.PathEscape(st.TriggerID()))
	if err := c.rest.Get(ctx, eventPath, &event); err != nil {
		return nil, err
	}
	var lines []string
	for _, a := range event.WorkflowDetails.Activities {
		if !strings.EqualFold(a.CompletedState, "failed") || a.ID == "" {
			continue
		}
		out.Activities = append(out.Activities, a.Name)
		stdout := fmt.Sprintf(convert.EaaSEnvironmentActivityStdoutFmt, url.PathEscape(project), url.PathEscape(st.Environment.Name), url.PathEscape(st.TriggerID()), url.PathEscape(a.ID))
		raw, err := c.rest.GetRaw(ctx, stdout+"?limit=0&offset=0&order=DESC&orderBy=createdAt")
		if err != nil {
			return nil, err
		}
		lines = append(lines, filterLogLines(string(raw))...)
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	out.Lines = lines
	return out, nil
}

// outputProblemPattern matches the plain (not "level=") lines that carry the cause of a failure,
// for example the output of a command the template ran ("error: A device with MAC ... already exists").
var outputProblemPattern = regexp.MustCompile(`(?i)\berror:|already exists|not found|permission denied|timed out|unreachable|connection refused|no such`)

// filterLogLines keeps WARNING and ERROR lines of a "time=... level=... msg=..."
// activity log, reduced to their message and shortened, plus plain output lines that
// report a problem (OUTPUT). Lines that mention secrets are dropped.
func filterLogLines(raw string) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		level := ""
		switch {
		case strings.Contains(line, "level=ERROR"):
			level = "ERROR"
		case strings.Contains(line, "level=WARNING"):
			level = "WARNING"
		case !strings.Contains(line, "level=") && outputProblemPattern.MatchString(line):
			level = "OUTPUT"
		default:
			continue
		}
		msg := line
		if i := strings.Index(line, "msg="); i >= 0 {
			msg = line[i+len("msg="):]
		}
		if low := strings.ToLower(msg); strings.Contains(low, "password") || strings.Contains(low, "secret") || strings.Contains(low, "token") {
			continue
		}
		if len(msg) > 240 {
			msg = msg[:240] + "..."
		}
		out = append(out, level+": "+strings.TrimSpace(msg))
	}
	return out
}

// withFirstClass adds include_first_class=true to a compute path. Without it the platform hides
// the compute instances (and profiles) that back first-class bare metal, VMs and clusters: a GET
// by name answers 404 and only the list shows them. Asking for them is harmless otherwise.
func withFirstClass(path string) string {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + "include_first_class=true"
}
