// Copyright 2026 Jose Vazquez
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
