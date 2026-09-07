<div align="center">

  <img src="logo.png" alt="Ancaeus" height="120">
  <h1>Ancaeus</h1>
  
  <p>
    <strong>Wi-Fi geolocation and timezone helper for Linux</strong>
  </p>

  <hr/>

  <h3>
    <a href="#nixos-flake">NixOS</a>
    <span> | </span>
    <a href="#other-linux-systemd">Other Linux</a>
    <span> | </span>
    <a href="#options-nixos">Options</a>
  </h3>

</div>

## Introduction

Named after **Ancaeus**, navigator of the Argo: it scans nearby Wi‑Fi, resolves coordinates via a pluggable geolocation provider, and can set the system timezone from that position.

Default provider is **[beaconDB](https://beacondb.net/)** (MLS-compatible). Google Geolocation API is optional. Coverage depends on the provider — for beaconDB you can improve your area with [NeoStumbler](https://github.com/mjaakko/NeoStumbler).

Endpoints (default `127.0.0.1:1223`):

- `POST /geolocate` — Google/GeoClue-compatible location response
- `GET /time-zone` — IANA timezone string

## NixOS (flake)

```nix
{
  inputs.ancaeus.url = "github:NickRI/ancaeus";
  inputs.ancaeus.inputs.nixpkgs.follows = "nixpkgs";

  # modules = [ inputs.ancaeus.nixosModules.default ];
}
```

```nix
{
  services.ancaeus.enable = true;
  # services.ancaeus.provider = "beacondb"; # default
  # services.ancaeus.beaconDBUrl = "https://api.beacondb.net/v1/geolocate";
  # services.ancaeus.provider = "google";
  # services.ancaeus.googleGeoTokenFile = config.sops.secrets.google-geo-key.path;
  # services.ancaeus.timezone.enable = false;
}
```

### Chromium extension

Experimental: overrides `navigator.geolocation` and talks to local Ancaeus.

**NixOS + home-manager** (HM as a NixOS module — `osConfig` is the system config):

```nix
# home.nix
{ osConfig, ... }:
{
  programs.chromium.extensions = [
    osConfig.services.ancaeus.chromiumExtension
  ];
}
```

**Standalone home-manager** (no `osConfig`): build the CRX yourself. The Ancaeus **daemon** must still run on the host (NixOS module or `packaging/` systemd units) — HM alone cannot replace Wi‑Fi scan + capabilities.

```nix
# home.nix
{ pkgs, inputs, ... }:
let
  ext = pkgs.callPackage "${inputs.ancaeus}/chromium-extension" {
    serverUrl = "http://127.0.0.1:1223/geolocate";
  };
in
{
  programs.chromium.extensions = [
    {
      id = ext.extensionId;
      crxPath = "${ext}/ancaeus-extension.crx";
      version = ext.version;
    }
  ];
}
```

### Firefox

Two options (Ancaeus must be running):

1. **Via GeoClue** (preferred on Linux if `services.ancaeus.geoclue.enable` / GeoClue already points at Ancaeus):

   In `about:config` set:

   - `geo.provider.use_geoclue` → `true`

2. **Direct URL** to Ancaeus:

   In `about:config` set:

   - `geo.provider.network.url` → `http://127.0.0.1:1223/geolocate`

   (Use your `listenAddress` if it differs.)

Do not point Firefox at beaconDB directly if you want Wi‑Fi scan enrichment — the browser cannot list BSSIDs; Ancaeus (or GeoClue→Ancaeus) does that locally.

## Other Linux (systemd)

Release assets (tag a GitHub release):

| File | Contents |
|------|----------|
| `ancaeus` / `ancaeus-linux-amd64.tar.gz` | Binary |
| `ancaeus-packaging.zip` | systemd units, NetworkManager hook, GeoClue snippet |
| `ancaeus-chromium-extension.zip` | Unpacked extension (URL baked to `http://127.0.0.1:1223/geolocate`) |

Minimal install:

```sh
sudo install -m755 ancaeus /usr/local/bin/ancaeus
sudo cp packaging/systemd/*.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now ancaeus.service

# GeoClue: merge packaging/geoclue/geoclue.conf.snippet into your GeoClue config
# Timezone (optional):
sudo cp packaging/networkmanager/10-ancaeus-wifi-up /etc/NetworkManager/dispatcher.d/
sudo chmod +x /etc/NetworkManager/dispatcher.d/10-ancaeus-wifi-up
sudo systemctl start ancaeus-timezone-update.service
```

```sh
# providers
ancaeus --provider beacondb
ancaeus --provider google --google-geo-token-file /path/to/key
```

Unset a fixed timezone if your distro pins one, so `timedatectl set-timezone` can manage it.

Needs network access for the geolocation provider, and permission to query Wi‑Fi (nl80211). Timezone updates need privileges for `timedatectl`.

## Privileges

| Component | Runs as | Why |
|-----------|---------|-----|
| `ancaeus.service` | **not root** (`DynamicUser`) + `CAP_NET_ADMIN` / `CAP_NET_RAW` | nl80211 Wi‑Fi scan without a full root process |
| `ancaeus-timezone-update.service` | **root** (default oneshot) | `timedatectl set-timezone` |
| Chromium / Firefox | your user | only HTTP to `127.0.0.1` |

## Options (NixOS)

| Option | Default | Description |
|--------|---------|-------------|
| `services.ancaeus.enable` | — | Enable service |
| `services.ancaeus.provider` | `beacondb` | `beacondb` \| `google` |
| `services.ancaeus.beaconDBUrl` | beaconDB public API | MLS geolocate URL (self-host OK) |
| `services.ancaeus.googleGeoTokenFile` | `null` | Google API key file (required for `google`) |
| `services.ancaeus.listenAddress` | `127.0.0.1:1223` | HTTP listen |
| `services.ancaeus.geoclue.enable` | `true` | Wire GeoClue2 provider URL |
| `services.ancaeus.timezone.enable` | `true` | Wi-Fi-up timezone updates |
| `services.ancaeus.chromiumExtension` | set when enabled | `{ id, crxPath, version }` for Chromium |

## Develop

```sh
go build -o ancaeus .
make release-artifacts
nix build
```
