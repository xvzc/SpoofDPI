{
  pkgs ? import <nixpkgs> { },
}:
let
  unstableTarball = builtins.fetchTarball "https://github.com/NixOS/nixpkgs/archive/nixos-unstable.tar.gz";

  unstablePkgs = import unstableTarball { };
  t-cmd = pkgs.writeShellScriptBin "t" ''
    exec task run -- "$@"
  '';
in
pkgs.mkShell {
  buildInputs = with pkgs; [
    libpcap
  ];
  packages = with pkgs; [
    go_1_26
    goreleaser
    go-task
    gopls
    golangci-lint-langserver
    unstablePkgs.golangci-lint
    (pkgs.python312.withPackages (pyPkgs: with pyPkgs; [ mkdocs-material ]))

    t-cmd
  ];

  shellHook = # sh
    ''
      export name="nix:spoofdpi"
      alias t='task run'
    '';
}
