{ pkgs, ... }:

pkgs.buildGoModule {
  pname = "ancaeus";
  version = "0.1.2";

  src = ./.;

  vendorHash = "sha256-7blwUQgdziPnRdwfv7gkKh8VjSdAaRrS+QXL12WSYDA=";

  meta = with pkgs.lib; {
    description = "Wi-Fi geolocation and timezone helper for GeoClue and browsers";
    homepage = "https://github.com/NickRI/ancaeus";
    license = licenses.mit;
    mainProgram = "ancaeus";
    platforms = platforms.linux;
  };
}
