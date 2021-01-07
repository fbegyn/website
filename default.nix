{ pkgs, stdenv, fetchFromGitHub }:

pkgs.buildGoModule rec {
  pname = "fbegyn-website";
  version = "0.1.0";
  src = /home/francis/Documents/projects/personal/website;

  vendorSha256 = null;
  runVend = true;

  meta = with stdenv.lib; {
    description = "Francis Begyn his website binary";
    homepage = "https://francis.begyn.be";
    license = licenses.mit;
  };
}
