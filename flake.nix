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

