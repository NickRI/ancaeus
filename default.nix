{ pkgs, ... }:

pkgs.buildGoModule {
  pname = "ancaeus";
  version = "0.1.2";

  src = ./.;

  subPackages = [ "./cmd/ancaeus" ];

  vendorHash = "sha256-rsoFI/tKBrJdbA7uA1blhXjwtERRqbCSEUeZcbvsv4k=";

  meta = with pkgs.lib; {
    description = "Wi-Fi geolocation and timezone helper for GeoClue and browsers";
    homepage = "https://github.com/NickRI/ancaeus";
    license = licenses.mit;
    mainProgram = "ancaeus";
    platforms = platforms.linux;
  };
}
