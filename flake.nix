{
  description = "gwt: Git worktree manager";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "gwt";
          version = "0.1.0";
          src = ./.;
          vendorHash = null;
        };
        packages.gwt = self.packages.${system}.default;

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [ go ginkgo ];
        };
      });
}