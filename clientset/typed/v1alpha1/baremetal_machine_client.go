package v1alpha1

import (
	"context"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

// baremetalMachineClient is the project-scoped client for BaremetalMachine.
type baremetalMachineClient struct {
	rest    *rest.Client
	project string
}

func newBaremetalMachineClient(restClient *rest.Client, project string) *baremetalMachineClient {
	return &baremetalMachineClient{rest: restClient, project: project}
}

func (c *baremetalMachineClient) scope() convert.InfraProjectScope {
	return convert.InfraProjectScope{Project: c.project}
}

func (c *baremetalMachineClient) Create(ctx context.Context, obj *apiv1.BaremetalMachine, _ gpupaas.CreateOptions) (*apiv1.BaremetalMachine, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "baremetal machine is nil"}
	}
	obj.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	wire := convert.ToInfraBaremetalMachine(stripStatusBaremetalMachine(obj), c.project)
	collection, _, _ := convert.BaremetalMachinePaths(c.scope())
	var out convert.InfraBaremetalMachine
	if err := c.rest.Post(ctx, collection(), wire, &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromInfraBaremetalMachine(&out), nil
	}
	return c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{})
}

func (c *baremetalMachineClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.BaremetalMachine, error) {
	_, item, _ := convert.BaremetalMachinePaths(c.scope())
	var out convert.InfraBaremetalMachine
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromInfraBaremetalMachine(&out), nil
}

func (c *baremetalMachineClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.BaremetalMachineList, error) {
	collection, _, _ := convert.BaremetalMachinePaths(c.scope())
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.InfraBaremetalMachineList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromInfraBaremetalMachineList(&out), nil
}

func (c *baremetalMachineClient) Delete(ctx context.Context, name string, _ gpupaas.DeleteOptions) error {
	_, item, _ := convert.BaremetalMachinePaths(c.scope())
	return c.rest.Delete(ctx, item(name))
}

// PowerOn / PowerOff / Reboot / Provision are GET sub-routes; the optional
// ActionOptions argument is accepted for interface parity but is not sent on
// the wire (no body for GET).
func (c *baremetalMachineClient) PowerOn(ctx context.Context, name string, _ gpupaas.ActionOptions) (*apiv1.BaremetalMachine, error) {
	return c.executeGetAction(ctx, name, "powerOn")
}

func (c *baremetalMachineClient) PowerOff(ctx context.Context, name string, _ gpupaas.ActionOptions) (*apiv1.BaremetalMachine, error) {
	return c.executeGetAction(ctx, name, "powerOff")
}

func (c *baremetalMachineClient) Reboot(ctx context.Context, name string, _ gpupaas.ActionOptions) (*apiv1.BaremetalMachine, error) {
	return c.executeGetAction(ctx, name, "reboot")
}

func (c *baremetalMachineClient) Provision(ctx context.Context, name string, _ gpupaas.ActionOptions) (*apiv1.BaremetalMachine, error) {
	return c.executeGetAction(ctx, name, "provision")
}

func (c *baremetalMachineClient) ReinstallOS(ctx context.Context, name string, image *apiv1.BaremetalImage, _ gpupaas.ActionOptions) (*apiv1.BaremetalMachine, error) {
	if image == nil || image.URL == "" {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "image url is required for ReinstallOS"}
	}
	_, _, subroute := convert.BaremetalMachinePaths(c.scope())
	payload := convert.ToInfraBaremetalImage(image)
	var out convert.InfraBaremetalMachine
	if err := c.rest.Post(ctx, subroute(name, "reinstallOS"), payload, &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromInfraBaremetalMachine(&out), nil
	}
	return c.Get(ctx, name, gpupaas.GetOptions{})
}

func (c *baremetalMachineClient) CreateConsoleSession(ctx context.Context, name string, req *apiv1.BaremetalConsoleSessionRequest, _ gpupaas.ActionOptions) (*apiv1.BaremetalConsoleSession, error) {
	if req == nil || req.ComputeID == "" {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "compute_id is required for CreateConsoleSession"}
	}
	_, _, subroute := convert.BaremetalMachinePaths(c.scope())
	payload := convert.ToInfraBaremetalConsoleSessionRequest(req)
	var out convert.InfraBaremetalConsoleSession
	if err := c.rest.Post(ctx, subroute(name, "consoleSessions"), payload, &out); err != nil {
		return nil, err
	}
	return convert.FromInfraBaremetalConsoleSession(&out), nil
}

func (c *baremetalMachineClient) GetStatusInfo(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.BaremetalMachineInfo, error) {
	_, _, subroute := convert.BaremetalMachinePaths(c.scope())
	var out convert.InfraBaremetalMachineInfo
	if err := c.rest.Get(ctx, subroute(name, "status"), &out); err != nil {
		return nil, err
	}
	return convert.FromInfraBaremetalMachineInfo(&out), nil
}

func (c *baremetalMachineClient) executeGetAction(ctx context.Context, name, action string) (*apiv1.BaremetalMachine, error) {
	_, _, subroute := convert.BaremetalMachinePaths(c.scope())
	var out convert.InfraBaremetalMachine
	if err := c.rest.Get(ctx, subroute(name, action), &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromInfraBaremetalMachine(&out), nil
	}
	return c.Get(ctx, name, gpupaas.GetOptions{})
}

func stripStatusBaremetalMachine(obj *apiv1.BaremetalMachine) *apiv1.BaremetalMachine {
	cp := *obj
	cp.Status = apiv1.BaremetalMachineStatus{}
	return &cp
}
