{
  config,
  pkgs,
  lib,
  ...
}:

let
  cfg = config.services.ancaeus;
  ancaeusPkg = pkgs.callPackage ./default.nix { };
  geolocateUrl = "http://${cfg.listenAddress}/geolocate";
  timezoneUrl = "http://${cfg.listenAddress}/time-zone";
  chromiumExt = pkgs.callPackage ./chromium-extension {
    serverUrl = geolocateUrl;
  };

  providerArgs = [
    "--provider ${lib.escapeShellArg cfg.provider}"
  ]
  ++ lib.optionals (cfg.provider == "beacondb") [
    "--beacondb-url ${lib.escapeShellArg cfg.beaconDBUrl}"
  ]
  ++ lib.optionals (cfg.provider == "google") [
    "--google-geo-token-file ${cfg.googleGeoTokenFile}"
  ];
in
{
  options.services.ancaeus = {
    enable = lib.mkEnableOption "Ancaeus Wi-Fi geolocation and timezone helper";

    package = lib.mkOption {
      type = lib.types.package;
      default = ancaeusPkg;
      description = "ancaeus package.";
    };

    listenAddress = lib.mkOption {
      type = lib.types.str;
      default = "127.0.0.1:7609";
      description = "Listen address for the HTTP API (keep on localhost).";
    };

    provider = lib.mkOption {
      type = lib.types.enum [
        "beacondb"
        "google"
        "apple"
      ];
      default = "beacondb";
      description = ''
        Geolocation backend: `beacondb` (default), `google` (needs googleGeoTokenFile).
        `apple` is unofficial/unsupported — use at your own risk; not documented for general use.
      '';
    };

    beaconDBUrl = lib.mkOption {
      type = lib.types.str;
      default = "https://api.beacondb.net/v1/geolocate";
      description = "BeaconDB (or self-hosted) MLS geolocate URL.";
    };

    googleGeoTokenFile = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = "File containing Google Geolocation API key. Required when provider = \"google\".";
    };

    geoclue = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description = "Point GeoClue2 at Ancaeus /geolocate.";
      };
    };

    timezone = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description = "Update system timezone from Ancaeus /time-zone on boot and Wi-Fi up.";
      };
    };

    chromiumExtension = lib.mkOption {
      type = lib.types.nullOr (
        lib.types.submodule {
          options = {
            id = lib.mkOption { type = lib.types.str; };
            crxPath = lib.mkOption { type = lib.types.path; };
            version = lib.mkOption { type = lib.types.str; };
          };
        }
      );
      default = null;
      description = ''
        Ready-to-use Chromium extension attr for home-manager
        `programs.chromium.extensions`. Set when `services.ancaeus.enable`.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    assertions = [
      {
        assertion = cfg.provider != "google" || cfg.googleGeoTokenFile != null;
        message = "services.ancaeus.provider = \"google\" requires services.ancaeus.googleGeoTokenFile";
      }
    ];

    services.ancaeus.chromiumExtension = {
      id = chromiumExt.extensionId;
      crxPath = "${chromiumExt}/ancaeus-extension.crx";
      version = chromiumExt.version;
    };

    systemd.services.ancaeus = {
      description = "Ancaeus Wi-Fi geolocation provider";
      wantedBy = [ "multi-user.target" ];
      after = [ "network-online.target" ];
      wants = [ "network-online.target" ];
      serviceConfig = {
        ExecStart = "${cfg.package}/bin/ancaeus --listen ${cfg.listenAddress} ${lib.concatStringsSep " " providerArgs}";
        CacheDirectory = "ancaeus";
        Restart = "on-failure";
        Type = "simple";
        DynamicUser = true;
        AmbientCapabilities = "CAP_NET_ADMIN CAP_NET_RAW";
        CapabilityBoundingSet = "CAP_NET_ADMIN CAP_NET_RAW";
        NoNewPrivileges = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        PrivateTmp = true;
        RestrictAddressFamilies = [
          "AF_UNIX"
          "AF_INET"
          "AF_INET6"
          "AF_NETLINK"
        ];
        RestrictNamespaces = true;
        LockPersonality = true;
        MemoryDenyWriteExecute = true;
      };
    };

    systemd.services.geoclue = lib.mkIf cfg.geoclue.enable {
      after = lib.mkAfter [ "ancaeus.service" ];
      wants = lib.mkAfter [ "ancaeus.service" ];
    };

    services.geoclue2 = lib.mkIf cfg.geoclue.enable {
      enable = true;
      geoProviderUrl = geolocateUrl;
    };

    time.timeZone = lib.mkIf cfg.timezone.enable (lib.mkForce null);

    systemd.services.ancaeus-timezone-update = lib.mkIf cfg.timezone.enable {
      description = "Ancaeus timezone update";
      wantedBy = [ "multi-user.target" ];
      requires = [ "ancaeus.service" ];
      after = [
        "ancaeus.service"
        "network-online.target"
      ];
      serviceConfig = {
        Type = "oneshot";
        TimeoutStartSec = "120";
      };
      script = ''
        for i in $(seq 1 60); do
          if zone="$(${pkgs.curl}/bin/curl -fsS --max-time 5 ${timezoneUrl})"; then
            if [ -n "$zone" ]; then
              timedatectl set-timezone "$zone"
              exit 0
            fi
            echo "empty timezone" >&2
          fi
          sleep 2
        done
        echo "timezone update timed out" >&2
        exit 1
      '';
    };

    environment.etc = lib.mkIf cfg.timezone.enable {
      "NetworkManager/dispatcher.d/10-ancaeus-wifi-up".source = pkgs.writeShellScript "ancaeus-wifi-up" ''
        if [ "$2" = "up" ]; then
          systemctl try-restart ancaeus-timezone-update.service
        fi
      '';
    };
  };
}
