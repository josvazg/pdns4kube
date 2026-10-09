// Package bases provides embedded access to the generated CRD bases shipped
// under config/crd/bases so that consumers can parse them without duplicating
// the YAML.
package bases

import (
	_ "embed"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

//go:embed pdns.example.io.yaml
var PDNSExampleIOYAML []byte

// CustomResourceDefinition parses the embedded pdns.example.io.yaml base into
// an apiextensionsv1.CustomResourceDefinition.
func CustomResourceDefinition() (*apiextensionsv1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(PDNSExampleIOYAML, crd); err != nil {
		return nil, err
	}
	return crd, nil
}
