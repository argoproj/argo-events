package triggers

import (
	"fmt"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/riferrei/srclient"
)

// GetSchemaFromRegistry returns a schema from registry
func GetSchemaFromRegistry(sr *apicommon.SchemaRegistryConfig) (*srclient.Schema, error) {
	schemaRegistryClient := srclient.CreateSchemaRegistryClient(sr.URL)
	if sr.Auth.Username != nil && sr.Auth.Password != nil {
		user, _ := common.GetSecretFromVolume(sr.Auth.Username)
		password, _ := common.GetSecretFromVolume(sr.Auth.Password)
		schemaRegistryClient.SetCredentials(user, password)
	}
	schema, err := schemaRegistryClient.GetSchema(int(sr.SchemaID))
	if err != nil {
		return nil, fmt.Errorf("error getting the schema with id '%d' %s", sr.SchemaID, err)
	}
	return schema, nil
}
