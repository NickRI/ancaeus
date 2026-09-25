{ pkgs, ... }:

pkgs.buildGoModule {
  pname = "ancaeus";
  version = "0.2.0";

  src = ./.;

  subPackages = [ "cmd/ancaeus" ];

  vendorHash = "sha256-hK6Yv/uP32jLrBTaX38BLZ55TRzD3HeA8wB5/Z67EL0=";

  meta = with pkgs.lib; {
    description = "Wi-Fi geolocation and timezone helper for GeoClue and browsers";
    homepage = "https://github.com/NickRI/ancaeus";
    license = licenses.mit;
    mainProgram = "ancaeus";
    platforms = platforms.linux;
  };
}
