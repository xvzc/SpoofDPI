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
    gopls
    golangci-lint-langserver
    unstablePkgs.golangci-lint
  ];

  shellHook = # sh
    ''
      export name="nix:SpoofDPI"
    '';
}
