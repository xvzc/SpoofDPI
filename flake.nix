{
  description = "spoofdpi - Simple and fast anti-censorship tool to bypass DPI";

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
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "spoofdpi";
          version = "dev";
          src = self;
          vendorHash = "sha256-FcepbOIB3CvHmTPiGWXukPg41uueQQYdZeVKmzjRuwA=";
          subPackages = [ "cmd/spoofdpi" ];
          buildInputs = pkgs.lib.optionals pkgs.stdenv.isDarwin [ pkgs.libpcap ];
          env.CGO_ENABLED = if pkgs.stdenv.isLinux then "0" else "1";
          ldflags = [
            "-s"
            "-w"
            "-X main.build=flake"
            "-X main.commit=${self.shortRev or "dirty"}"
          ];
          meta = {
            description = "Simple and fast anti-censorship tool written in Go";
            homepage = "https://github.com/xvzc/SpoofDPI";
            license = pkgs.lib.licenses.asl20;
          };
        };
      }
    );
}
