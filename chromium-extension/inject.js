// inject.js
(function () {
  function createGeolocationError(code, message) {
    const error = new Error(message);
    error.code = code;
    error.PERMISSION_DENIED = 1;
    error.POSITION_UNAVAILABLE = 2;
    error.TIMEOUT = 3;
    return error;
  }

  function requestPosition(success, error) {
    const requestId = Math.random().toString(36).slice(2);
    window.postMessage({ type: "getGeo", requestId }, "*");

    const timeout = setTimeout(() => {
      window.removeEventListener("message", handler);
      if (error) error(createGeolocationError(3, "Request timeout"));
    }, 10000);

    function handler(e) {
      if (e.source !== window || !e.data || e.data.type !== "geoResult") return;
      if (e.data.requestId && e.data.requestId !== requestId) return;
      clearTimeout(timeout);
      window.removeEventListener("message", handler);
      if (e.data.error) {
        if (error) error(createGeolocationError(2, e.data.error));
        return;
      }
      if (!e.data.data || !e.data.data.coords) {
        if (error) error(createGeolocationError(2, "Invalid response format"));
        return;
      }
      success({
        coords: {
          latitude: e.data.data.coords.latitude,
          longitude: e.data.data.coords.longitude,
          accuracy: e.data.data.coords.accuracy,
          altitude: null,
          altitudeAccuracy: null,
          heading: null,
          speed: null,
        },
        timestamp: e.data.data.timestamp || Date.now(),
      });
    }

    window.addEventListener("message", handler);
  }

  navigator.geolocation.getCurrentPosition = function (success, error) {
    requestPosition(success, error);
  };

  const watches = new Map();
  let nextWatchId = 1;

  navigator.geolocation.watchPosition = function (success, error) {
    const id = nextWatchId++;
    const tick = () => requestPosition(success, error);
    tick();
    watches.set(id, setInterval(tick, 5000));
    return id;
  };

  navigator.geolocation.clearWatch = function (id) {
    const handle = watches.get(id);
    if (handle) {
      clearInterval(handle);
      watches.delete(id);
    }
  };
})();
