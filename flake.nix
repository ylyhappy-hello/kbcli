{
  description = "my project description";

  outputs = { self, nixpkgs }:
    let pkgs = nixpkgs.legacyPackages.x86_64-linux;
    in let
      kbcli = pkgs.stdenv.mkDerivation rec {
        name = "kbcli";
        version = "0.8.2";
        src = pkgs.fetchzip {
          url =
            "https://github.com/apecloud/kbcli/releases/download/v${version}/kbcli-linux-amd64-v${version}.tar.gz";
          hash = "sha256-lZWSxXap88+Gw69vQm3EEpc/Ed+B0GXdX7i+3OoDOqI=";
        };
        installPhase = ''
          install -m111 -D $src/kbcli $out/bin/kbcli
        '';
      };
    in {
      packages.x86_64-linux.default = kbcli;

    };
}
