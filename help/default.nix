{  }: with import <nixpkgs> {};

buildGoPackage rec {
  name = "SDM";
  src = ./..;
  buildInputs = [ tree ];
  goPackagePath = "github.com/pallavagarwal07/sdm";
  preInstall = ''
    mkdir -p $bin/share/applications
    mkdir -p $bin/share/icons/hicolor/scalable/apps
    cp $(find . -name sdm.desktop) $bin/share/applications
    cp $(find . -name icon.svg) $bin/share/icons/hicolor/scalable/apps/sdm_icon.svg
  '';
}
