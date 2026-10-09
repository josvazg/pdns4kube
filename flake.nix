# Copyright 2026 Jose Vazquez
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

{
  description = "Go Kubernetes Operator Development Environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            git
            go
            docker
            kind
            kubectl
            mage
	    kubernetes-controller-tools
          ];

          shellHook = ''
            echo "K8s Operator Dev Environment Loaded"
            echo "Go:      $(go version)"
            echo "Kind:    $(kind version)"
            echo "Kubectl: $(kubectl version --client --output=yaml | grep gitVersion | head -n 1 | awk '{print $2}')"
            alias oc=opencode
	  '';
        };
      }
    );
}

