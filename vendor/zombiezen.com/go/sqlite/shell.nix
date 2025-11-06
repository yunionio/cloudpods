{ pkgs ? import (fetchTarball "https://github.com/NixOS/nixpkgs/archive/8c66bd1b68f4708c90dcc97c6f7052a5a7b33257.tar.gz") {}
}:

pkgs.mkShell {
  packages = [
    pkgs.go_1_20
  ];
}
