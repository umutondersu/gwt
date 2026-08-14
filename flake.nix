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
        gwtVersion =
          if builtins.pathExists ./VERSION then
            builtins.replaceStrings [ "\n" ] [ "" ] (builtins.readFile ./VERSION)
          else if self ? shortRev then
            "0.0.0+${self.shortRev}"
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