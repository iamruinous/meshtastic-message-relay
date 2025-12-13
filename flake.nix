{
  description = "Meshtastic Message Relay - Forward Meshtastic messages to various endpoints";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};

        # Build the Go module
        meshtastic-relay = pkgs.buildGoModule {
          pname = "meshtastic-relay";
          version = "0.2.0";

          src = ./.;

          # Vendor hash for Go dependencies
          # To update: run `go mod vendor && nix hash path vendor`
          vendorHash = "sha256-5hO7c4uhMDNPxVebh1Z5Jl1OZDLxmWYCpT6xiAvLtFI=";

          # Build the main binary
          subPackages = ["cmd/relay"];

          # Rename the binary
          postInstall = ''
            mv $out/bin/relay $out/bin/meshtastic-relay
          '';

          meta = with pkgs.lib; {
            description = "Meshtastic Message Relay - Forward messages to HTTP endpoints, files, and more";
            homepage = "https://github.com/iamruinous/meshtastic-message-relay";
            license = licenses.mit;
            maintainers = [];
            mainProgram = "meshtastic-relay";
          };
        };
      in {
        packages = {
          default = meshtastic-relay;
          meshtastic-relay = meshtastic-relay;
        };

        apps = {
          default = {
            type = "app";
            program = "${meshtastic-relay}/bin/meshtastic-relay";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain (uses latest available in nixpkgs)
            go
            gopls
            gotools
            golangci-lint
            delve

            # Build tools
            gnumake
          ];

          shellHook = ''
            echo "Meshtastic Message Relay development environment"
            echo "Go version: $(go version)"
            echo ""
            echo "Available commands:"
            echo "  go run ./cmd/relay       - Run the relay"
            echo "  go test ./...            - Run tests"
            echo "  go build ./cmd/relay     - Build the binary"
            echo "  nix build                - Build with Nix"
          '';
        };
      }
    )
    // {
      # Overlay for including in other flakes
      overlays.default = final: prev: {
        meshtastic-relay = self.packages.${prev.system}.meshtastic-relay;
      };
    };
}
