{
  description = "splitans - Parse ANSI/ASCII file and split text and CSI Sequence";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        splitans = pkgs.buildGoModule {
          pname = "splitans";
          # x-release-please-start-version
          version = "0.1.0";
          # x-release-please-end
          src = ./.;

          vendorHash = null;

          meta = with pkgs.lib; {
            description = "A tool for Parsing ANSI/ASCII file and split text and CSI Sequence";
            homepage = "https://github.com/badele/splitans";
            license = licenses.mit;
            maintainers = [ ];
          };
        };
      in
      {
        packages = {
          default = splitans;
          splitans = splitans;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go development
            go
            gopls
            gotools # goimports, godoc, etc.
            go-tools # staticcheck, etc.

            # Build tools
            just

            # Pre-commit hooks
            pre-commit

            # Docker linting
            hadolint

            splitans
          ];

          shellHook = ''
            echo "🚀 splitans development environment"
            echo "Go version: $(go version)"
            echo ""
            just
          '';
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/splitans";
        };
      }
    );
}
