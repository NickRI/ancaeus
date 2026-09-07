// content.js — inject page script early, bridge getGeo <-> background
(function () {
  const script = document.createElement("script");
  script.src = chrome.runtime.getURL("inject.js");
  script.onload = function () {
    this.remove();
  };
  (document.head || document.documentElement).appendChild(script);
})();

window.addEventListener("message", (e) => {
  if (e.source !== window || !e.data || e.data.type !== "getGeo") return;
  const requestId = e.data.requestId;
  chrome.runtime.sendMessage({ type: "geo" }, (result) => {
    if (chrome.runtime.lastError) {
      window.postMessage(
        { type: "geoResult", requestId, error: chrome.runtime.lastError.message },
        "*"
      );
      return;
    }
    window.postMessage({ type: "geoResult", requestId, data: result }, "*");
  });
});
