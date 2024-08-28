{pkgs, ...}: let
  go = pkgs.go_1_23;
in {
  packageOverrides = pkgs: {
    buildGoModule = pkgs.buildGoModule.override {go = go;};
    go = go;
  };
}
