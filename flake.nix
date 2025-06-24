
{
  description = "TerraLink";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        pname = "terralink";
        version = "1.0.0";  # x-release-please-version
        rev = self.rev or "dirty";

      in {
        # 'packages' is the standard output for things you can build and install.
        # Devbox looks for this.
        packages = {
          terralink = pkgs.buildGoModule {
            inherit pname version;
            src = ./.;
            #vendorHash = "sha256-cm9sg/whp/jrChcHlI1bAC9RQ48N3C8YTQ5Doy96Zvk=";
            ldflags = [
              "-s"
              "-w"
              "-X main.version=${version}-${rev}"
            ];
          };
        };

        devShells = {
          default = pkgs.mkShell {
            packages = [
              pkgs.go
              pkgs.gopls # Go language server
              pkgs.gotools
              pkgs.delve # Go debugger
            ];
          };
        };
      });
}