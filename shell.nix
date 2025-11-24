{
  pkgs ? import <nixpkgs> { },
}:
let
  unstableTarball = builtins.fetchTarball "https://github.com/NixOS/nixpkgs/archive/nixos-unstable.tar.gz";

  unstablePkgs = import unstableTarball { };
in
pkgs.mkShell {
  buildInputs = with pkgs; [
    libpcap
  ];
  packages = with pkgs; [
    go
    goreleaser
    gopls
    golangci-lint-langserver
    unstablePkgs.golangci-lint
    (pkgs.python312.withPackages (pyPkgs: with pyPkgs; [ mkdocs-material ]))
  ];

  shellHook = # sh
    ''
      export name="nix:SpoofDPI"
    '';
}
