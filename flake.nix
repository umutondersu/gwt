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
        latestRelease = builtins.tryEval (
          builtins.fromJSON (builtins.readFile (builtins.fetchurl "https://api.github.com/repos/umutondersu/gwt/releases/latest"))
        );
        gwtVersion =
          if latestRelease.success then
            latestRelease.value.tag_name
          else if self ? shortRev then
            "0.0.0+${self.shortRev}"
          else if self ? dirtyShortRev then
            "0.0.0+${self.dirtyShortRev}"
          else
            "dev";
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "gwt";
          version = gwtVersion;
          src = ./.;
          vendorHash = null;
          nativeBuildInputs = [ pkgs.git ];
          ldflags = [ "-X github.com/umutondersu/gwt/cmd.version=${gwtVersion}" ];
        };
        packages.gwt = self.packages.${system}.default;

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [ go ginkgo ];
        };
      });
}