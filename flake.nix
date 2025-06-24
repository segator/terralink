
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
        pkgPath ="${pname}/cmd";
        version = "0.2.0";  # x-release-please-version
        rev = self.rev or "dirty";

      in {
        # 'packages' is the standard output for things you can build and install.
        # Devbox looks for this.
        packages = {
          terralink = pkgs.buildGoModule {
            inherit pname version;
            src = ./.;
            vendorHash = null;
            ldflags = [
              "-s"
              "-w"
              "-X '${pkgPath}.Version=${version}'"
              "-X '${pkgPath}.Commit=${self.shortRev or "dirty"}'"
              "-X '${pkgPath}.BuildDate=${self.lastModifiedDate}'"
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