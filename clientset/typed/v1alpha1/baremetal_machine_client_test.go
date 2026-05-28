package v1alpha1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

func TestBaremetalMachineClientCRUD(t *testing.T) {
	const base = "/apis/infra.k8smgmt.io/v3/projects/demo/baremetalmachines"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(convert.InfraBaremetalMachineList{
				APIVersion: convert.InfraAPIVersion,
				Kind:       convert.InfraBaremetalMachineListKind,
				Items: []convert.InfraBaremetalMachine{{
					APIVersion: convert.InfraAPIVersion,
					Kind:       convert.InfraBaremetalMachineKind,
					Metadata:   convert.InfraMetadata{Name: "bm1", Project: "demo"},
					Spec:       convert.InfraBaremetalMachineSpec{Hostname: "bm1-host"},
				}},
			})
		case r.Method == http.MethodGet && r.URL.Path == base+"/bm1":
			_ = json.NewEncoder(w).Encode(convert.InfraBaremetalMachine{
				APIVersion: convert.InfraAPIVersion,
				Kind:       convert.InfraBaremetalMachineKind,
				Metadata:   convert.InfraMetadata{Name: "bm1", Project: "demo"},
				Spec:       convert.InfraBaremetalMachineSpec{Hostname: "bm1-host"},
				Status: convert.InfraBaremetalMachineStatus{
					Conditions: []convert.InfraBaremetalMachineCondition{{Type: "Provisioned", Status: "Success"}},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == base:
			var in convert.InfraBaremetalMachine
			_ = json.NewDecoder(r.Body).Decode(&in)
			// Echo back the created resource with a status field.
			in.Status = convert.InfraBaremetalMachineStatus{
				Conditions: []convert.InfraBaremetalMachineCondition{{Type: "Pending", Status: "NotSet"}},
			}
			_ = json.NewEncoder(w).Encode(in)
		case r.Method == http.MethodDelete && r.URL.Path == base+"/bm1":
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	bms := cs.V1alpha1().BaremetalMachines("demo")

	list, err := bms.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Spec.Hostname != "bm1-host" {
		t.Fatalf("list: %+v", list.Items)
	}

	got, err := bms.Get(ctx, "bm1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.Name != "bm1" || len(got.Status.Conditions) != 1 {
		t.Fatalf("get: %+v", got)
	}

	online := true
	created, err := bms.Create(ctx, &apiv1.BaremetalMachine{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindBaremetalMachine},
		Metadata: apiv1.ObjectMeta{Name: "bm-new", Project: "demo"},
		Spec: apiv1.BaremetalMachineSpec{
			Hostname: "bm-new-host",
			Online:   &online,
			Image:    &apiv1.BaremetalImage{URL: "http://images/example.qcow2", Format: "qcow2"},
		},
	}, gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if created.Metadata.Name != "bm-new" || len(created.Status.Conditions) == 0 {
		t.Fatalf("created: %+v", created)
	}

	if err := bms.Delete(ctx, "bm1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestBaremetalMachineClientSubActions(t *testing.T) {
	const base = "/apis/infra.k8smgmt.io/v3/projects/demo/baremetalmachines/bm1"

	calls := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base+"/powerOn":
			calls["powerOn"]++
			_ = json.NewEncoder(w).Encode(convert.InfraBaremetalMachine{
				APIVersion: convert.InfraAPIVersion,
				Kind:       convert.InfraBaremetalMachineKind,
				Metadata:   convert.InfraMetadata{Name: "bm1", Project: "demo"},
			})
		case r.Method == http.MethodGet && r.URL.Path == base+"/powerOff":
			calls["powerOff"]++
			_ = json.NewEncoder(w).Encode(convert.InfraBaremetalMachine{
				APIVersion: convert.InfraAPIVersion,
				Kind:       convert.InfraBaremetalMachineKind,
				Metadata:   convert.InfraMetadata{Name: "bm1", Project: "demo"},
			})
		case r.Method == http.MethodGet && r.URL.Path == base+"/reboot":
			calls["reboot"]++
			_ = json.NewEncoder(w).Encode(convert.InfraBaremetalMachine{
				APIVersion: convert.InfraAPIVersion,
				Kind:       convert.InfraBaremetalMachineKind,
				Metadata:   convert.InfraMetadata{Name: "bm1", Project: "demo"},
			})
		case r.Method == http.MethodGet && r.URL.Path == base+"/provision":
			calls["provision"]++
			_ = json.NewEncoder(w).Encode(convert.InfraBaremetalMachine{
				APIVersion: convert.InfraAPIVersion,
				Kind:       convert.InfraBaremetalMachineKind,
				Metadata:   convert.InfraMetadata{Name: "bm1", Project: "demo"},
			})
		case r.Method == http.MethodPost && r.URL.Path == base+"/reinstallOS":
			calls["reinstallOS"]++
			var img convert.InfraBaremetalImage
			_ = json.NewDecoder(r.Body).Decode(&img)
			if img.URL == "" {
				t.Errorf("reinstallOS: expected URL in body")
			}
			_ = json.NewEncoder(w).Encode(convert.InfraBaremetalMachine{
				APIVersion: convert.InfraAPIVersion,
				Kind:       convert.InfraBaremetalMachineKind,
				Metadata:   convert.InfraMetadata{Name: "bm1", Project: "demo"},
			})
		case r.Method == http.MethodPost && r.URL.Path == base+"/consoleSessions":
			calls["consoleSessions"]++
			var req convert.InfraBaremetalConsoleSessionRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.ComputeID != "compute-42" {
				t.Errorf("consoleSessions: expected compute_id=compute-42, got %q", req.ComputeID)
			}
			_ = json.NewEncoder(w).Encode(convert.InfraBaremetalConsoleSession{
				AgentSessionID: "agent-1",
				ConsoleURL:     "wss://example/console",
				SessionID:      "session-1",
			})
		case r.Method == http.MethodGet && r.URL.Path == base+"/status":
			calls["status"]++
			_ = json.NewEncoder(w).Encode(convert.InfraBaremetalMachineInfo{
				Data: convert.InfraBaremetalMachineData{
					Fields: map[string]interface{}{"hardware": map[string]interface{}{"cpus": 96}},
				},
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	bms := cs.V1alpha1().BaremetalMachines("demo")

	if _, err := bms.PowerOn(ctx, "bm1", gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := bms.PowerOff(ctx, "bm1", gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := bms.Reboot(ctx, "bm1", gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := bms.Provision(ctx, "bm1", gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := bms.ReinstallOS(ctx, "bm1", &apiv1.BaremetalImage{
		URL:    "http://images/example.qcow2",
		Format: "qcow2",
	}, gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
	session, err := bms.CreateConsoleSession(ctx, "bm1", &apiv1.BaremetalConsoleSessionRequest{
		ComputeID: "compute-42",
	}, gpupaas.ActionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if session == nil || session.SessionID != "session-1" || session.ConsoleURL != "wss://example/console" {
		t.Fatalf("console session: %+v", session)
	}
	info, err := bms.GetStatusInfo(ctx, "bm1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if info == nil || info.Data.Fields["hardware"] == nil {
		t.Fatalf("status info: %+v", info)
	}

	// Validate that input validation triggers without hitting the server.
	if _, err := bms.ReinstallOS(ctx, "bm1", &apiv1.BaremetalImage{}, gpupaas.ActionOptions{}); err == nil {
		t.Fatal("expected error for empty reinstall image url")
	}
	if _, err := bms.CreateConsoleSession(ctx, "bm1", &apiv1.BaremetalConsoleSessionRequest{}, gpupaas.ActionOptions{}); err == nil {
		t.Fatal("expected error for missing compute_id")
	}

	for _, want := range []string{"powerOn", "powerOff", "reboot", "provision", "reinstallOS", "consoleSessions", "status"} {
		if calls[want] != 1 {
			t.Errorf("expected exactly 1 call to %s, got %d", want, calls[want])
		}
	}
}
