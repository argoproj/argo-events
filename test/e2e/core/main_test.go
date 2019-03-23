package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	/*

	   "net/http"
	   "net/url"
	*/

	common "github.com/argoproj/argo-events/common"
	e2ecommon "github.com/argoproj/argo-events/test/e2e/common"
)

var (
	client       *e2ecommon.E2EClient
	e2eID        string
	tmpNamespace string
)

func setup() error {
	e2eID = e2ecommon.GetE2EID()
	cli, err := e2ecommon.NewE2EClient(e2eID)
	if err != nil {
		return err
	}
	client = cli
	namespace, err := client.CreateTmpNamespace()
	if err != nil {
		return err
	}
	tmpNamespace = namespace
	fmt.Printf("tmpNamespace: %v\n", tmpNamespace)
	return nil
}

func teardown() error {
	if !e2ecommon.KeepNamespace() {
		err := client.DeleteNamespaces()
		if err != nil {
			return err
		}
	}
	return nil
}

func TestMain(m *testing.M) {
	err := setup()
	if err != nil {
		panic(err)
	}

	ret := m.Run()

	err = teardown()
	if err != nil {
		fmt.Errorf("%+v\n", err)
	}

	os.Exit(ret)
}

func TestGeneralUseCase(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	dir, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		t.Fatal(err)
	}
	manifestsDir := filepath.Join(dir, "manifests")

	gateway, err := e2ecommon.ReadResource(filepath.Join(manifestsDir, "webhook-gateway.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	e2ecommon.SetLabel(gateway, common.LabelKeyGatewayControllerInstanceID, e2eID)

	cm, err := e2ecommon.ReadResource(filepath.Join(manifestsDir, "webhook-gateway-configmap.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	sensor, err := e2ecommon.ReadResource(filepath.Join(manifestsDir, "webhook-sensor.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	e2ecommon.SetLabel(sensor, common.LabelKeySensorControllerInstanceID, e2eID)

	_, err = client.Create(tmpNamespace, gateway)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Create(tmpNamespace, cm)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Create(tmpNamespace, sensor)
	if err != nil {
		t.Fatal(err)
	}
	/*

	   req, err := http.NewRequest(method, endpointURL.String(), body)
	   if err != nil {
	       return nil, err
	   }

	   req = req.WithContext(ctx)

	*/

}
