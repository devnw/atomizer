{
  description = "development flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
    _1password-shell-plugins.url = "github:1Password/shell-plugins";
    gomod2nix = {
      url = "github:tweag/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.utils.follows = "utils";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      gomod2nix,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          config = {
            allowUnfree = true;
          };
          system = system;
          overlays = [
            gomod2nix.overlays.default
            (self: super: {
              go = super.go_1_23;
              python = super.python3.withPackages (
                subpkgs: with subpkgs; [
                  openapi-spec-validator
                  detect-secrets
                  requests
                  python-dotenv
                ]
              );
            })
          ];
        };

        pkglist = with pkgs; [
          _1password
          gibberish-detector
          addlicense
          shfmt
          git
          pre-commit
          shellcheck
          automake
          act
          gcc

          python

          go
          gopls
          gotools
          go-tools
          gomod2nix.packages.${system}.default
          sqlite-interactive

          delve
          golangci-lint
          goreleaser
          go-licenses
        ];
      in
      {
        packages.default = pkgs.buildGoApplication {
          pname = "{{module_name}}";
          version = "0.1";

          pwd = ./.;
          src = ./.;
          modules = ./gomod2nix.toml;
          buildInputs = pkglist;

          buildPhase = ''
            make build
          '';
          installPhase = ''
            make install
          '';
        };

        devShells.default = pkgs.mkShell { buildInputs = pkglist; };
      }
    );
}
